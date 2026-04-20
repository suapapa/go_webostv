package webostv

import (
	"context"
	"encoding/json"
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
	defer client.Close()

	if client.conn == nil {
		t.Error("connection should not be nil")
	}
}

func TestClient_Request(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

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
	defer client.Close()

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
		defer conn.Close()

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}

			var m Message
			_ = json.Unmarshal(msg, &m)

			if m.Type == "subscribe" {
				// Send an initial response
				resp := Message{
					Type:    "response",
					ID:      m.ID,
					Payload: json.RawMessage(`{"subscribed": true}`),
				}
				_ = conn.WriteJSON(resp)

				// Send an update
				update := Message{
					Type:    "response",
					ID:      m.ID,
					Payload: json.RawMessage(`{"value": 42}`),
				}
				_ = conn.WriteJSON(update)
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
	defer client.Close()

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
