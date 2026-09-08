package webostv

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestClient_Connect(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		_, _ = upgrader.Upgrade(w, r, nil)
	}))
	defer s.Close()

	host := strings.TrimPrefix(s.URL, "http://")
	client := NewClient(host)
	err := client.Connect()
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer closeClient(t, client)

	if client.conn == nil {
		t.Error("connection should not be nil")
	}
}

func TestClient_ConnectDoesNotMutateDefaultDialer(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		_, _ = upgrader.Upgrade(w, r, nil)
	}))
	defer s.Close()

	originalTimeout := websocket.DefaultDialer.HandshakeTimeout
	host := strings.TrimPrefix(s.URL, "http://")
	client := NewClient(host, WithHandshakeTimeout(time.Second))
	if err := client.Connect(); err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer closeClient(t, client)

	if websocket.DefaultDialer.HandshakeTimeout != originalTimeout {
		t.Fatal("Connect mutated websocket.DefaultDialer")
	}
}

func TestClient_SendMessageWithoutConnection(t *testing.T) {
	client := NewClient("localhost")
	_, _, err := client.SendMessage(
		context.Background(),
		"",
		"request",
		"ssap://test",
		nil,
	)
	if !errors.Is(err, ErrConnectionClosed) {
		t.Fatalf("expected ErrConnectionClosed, got %v", err)
	}
}

func TestClient_URL(t *testing.T) {
	tests := []struct {
		name string
		host string
		want string
	}{
		{name: "hostname", host: "tv.local", want: "ws://tv.local:3000/"},
		{name: "custom port", host: "tv.local:1234", want: "ws://tv.local:1234/"},
		{name: "IPv6", host: "2001:db8::1", want: "ws://[2001:db8::1]:3000/"},
		{name: "bracketed IPv6", host: "[2001:db8::1]", want: "ws://[2001:db8::1]:3000/"},
		{name: "scoped IPv6", host: "fe80::1%en0", want: "ws://[fe80::1%en0]:3000/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewClient(tt.host).URL(); got != tt.want {
				t.Fatalf("URL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClient_RegisterRejectsNilStore(t *testing.T) {
	client := NewClient("localhost")
	statusChan, errChan := client.Register(context.Background(), nil)

	if _, ok := <-statusChan; ok {
		t.Fatal("status channel should be closed")
	}
	if err := <-errChan; err == nil {
		t.Fatal("expected nil store error")
	}
}

func TestClient_RegisterRemovesWaiter(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()

		_, data, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var request Message
		if err := json.Unmarshal(data, &request); err != nil {
			return
		}
		_ = conn.WriteJSON(Message{
			Type:    "registered",
			ID:      request.ID,
			Payload: json.RawMessage(`{"client-key":"test-key"}`),
		})
	}))
	defer s.Close()

	host := strings.TrimPrefix(s.URL, "http://")
	client := NewClient(host)
	if err := client.Connect(); err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer closeClient(t, client)

	store := map[string]string{}
	statusChan, errChan := client.Register(context.Background(), store)
	if status := <-statusChan; status != Registered {
		t.Fatalf("expected Registered, got %v", status)
	}
	if err := <-errChan; err != nil {
		t.Fatalf("registration failed: %v", err)
	}
	if store["client_key"] != "test-key" {
		t.Fatalf("client key = %q, want %q", store["client_key"], "test-key")
	}

	client.waiterMu.RLock()
	waiterCount := len(client.waiters)
	client.waiterMu.RUnlock()
	if waiterCount != 0 {
		t.Fatalf("registration left %d waiter(s)", waiterCount)
	}
}

func TestClient_Request(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}

			var m Message
			_ = json.Unmarshal(msg, &m)

			resp := Message{
				Type:    "response",
				ID:      m.ID,
				Payload: json.RawMessage(`{"status": "ok"}`),
			}
			_ = conn.WriteJSON(resp)
		}
	}))
	defer s.Close()

	host := strings.TrimPrefix(s.URL, "http://")
	client := NewClient(host)
	err := client.Connect()
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer closeClient(t, client)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resp, err := client.Request(ctx, "ssap://test", nil)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.Type != "response" {
		t.Errorf("expected response type, got %s", resp.Type)
	}

	var p map[string]string
	_ = json.Unmarshal(resp.Payload, &p)
	if p["status"] != "ok" {
		t.Errorf("expected status ok, got %v", p["status"])
	}
}

func TestClient_Subscribe(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}

			var m Message
			_ = json.Unmarshal(msg, &m)

			if m.Type == "subscribe" {
				resp := Message{
					Type:    "response",
					ID:      m.ID,
					Payload: json.RawMessage(`{"value": 42}`),
				}
				_ = conn.WriteJSON(resp)
			}
		}
	}))
	defer s.Close()

	host := strings.TrimPrefix(s.URL, "http://")
	client := NewClient(host)
	err := client.Connect()
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer closeClient(t, client)

	updateChan := make(chan int, 1)
	_, err = client.Subscribe("ssap://test/subscribe", func(p json.RawMessage) {
		var res struct {
			Value int `json:"value"`
		}
		if err := json.Unmarshal(p, &res); err == nil && res.Value != 0 {
			updateChan <- res.Value
		}
	})
	if err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}

	select {
	case val := <-updateChan:
		if val != 42 {
			t.Errorf("expected 42, got %d", val)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for update")
	}
}

func TestClient_UnsubscribeDoesNotCreateWaiter(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()

		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer s.Close()

	host := strings.TrimPrefix(s.URL, "http://")
	client := NewClient(host)
	if err := client.Connect(); err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer closeClient(t, client)

	id, err := client.Subscribe("ssap://test/subscribe", func(json.RawMessage) {})
	if err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}
	if err := client.Unsubscribe(id); err != nil {
		t.Fatalf("unsubscribe failed: %v", err)
	}

	client.waiterMu.RLock()
	waiterCount := len(client.waiters)
	client.waiterMu.RUnlock()
	if waiterCount != 0 {
		t.Fatalf("unsubscribe left %d waiter(s)", waiterCount)
	}
}

func closeClient(t *testing.T, client *Client) {
	t.Helper()
	if err := client.Close(); err != nil {
		t.Errorf("failed to close client: %v", err)
	}
}
