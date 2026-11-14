package webostv

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestInputControlRejectsInvalidMouseCommands(t *testing.T) {
	input := InputControl{mouseConn: &websocket.Conn{}}
	tests := []struct {
		name string
		call func() error
	}{
		{
			name: "odd parameter count",
			call: func() error {
				return input.sendMouseCommand("move", "dx")
			},
		},
		{
			name: "newline in value",
			call: func() error {
				return input.Button("ENTER\ntype:click")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.call(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestInputControlReconnectsAfterServerClose(t *testing.T) {
	var (
		mu       sync.Mutex
		conns    int
		received []string
		firstEnd = make(chan struct{})
	)

	var pointerSrv *httptest.Server
	pointerSrv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		mu.Lock()
		conns++
		connNum := conns
		mu.Unlock()

		if connNum == 1 {
			// Wait for one message, then cleanly close the pointer socket.
			_, msg, err := conn.ReadMessage()
			if err == nil {
				mu.Lock()
				received = append(received, string(msg))
				mu.Unlock()
			}
			_ = conn.WriteControl(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, "idle"),
				time.Now().Add(time.Second),
			)
			_ = conn.Close()
			close(firstEnd)
			return
		}

		defer func() { _ = conn.Close() }()
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			mu.Lock()
			received = append(received, string(msg))
			mu.Unlock()
		}
	}))
	defer pointerSrv.Close()

	pointerURL := "ws://" + strings.TrimPrefix(pointerSrv.URL, "http://") + "/pointer"
	mainSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()

		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var req Message
			if err := json.Unmarshal(data, &req); err != nil {
				return
			}
			_ = conn.WriteJSON(Message{
				Type:    "response",
				ID:      req.ID,
				Payload: json.RawMessage(`{"socketPath":"` + pointerURL + `"}`),
			})
		}
	}))
	defer mainSrv.Close()

	client := NewClient(strings.TrimPrefix(mainSrv.URL, "http://"))
	if err := client.Connect(); err != nil {
		t.Fatalf("Connect() err = %v", err)
	}
	defer closeClient(t, client)

	input := &InputControl{Control: Control{Client: client}}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := input.ConnectInput(ctx); err != nil {
		t.Fatalf("ConnectInput #1 err = %v", err)
	}
	if err := input.Up(); err != nil {
		t.Fatalf("Up #1 err = %v", err)
	}

	<-firstEnd
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		input.mouseConnMu.Lock()
		cleared := input.mouseConn == nil
		input.mouseConnMu.Unlock()
		if cleared {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	input.mouseConnMu.Lock()
	cleared := input.mouseConn == nil
	input.mouseConnMu.Unlock()
	if !cleared {
		t.Fatal("expected readMouseLoop to clear mouseConn after server close")
	}

	if err := input.ConnectInput(ctx); err != nil {
		t.Fatalf("ConnectInput #2 err = %v", err)
	}
	if err := input.Down(); err != nil {
		t.Fatalf("Down #2 err = %v", err)
	}

	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		n := len(received)
		mu.Unlock()
		if n >= 2 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 2 {
		t.Fatalf("received %d messages, want 2: %v", len(received), received)
	}
	if !strings.Contains(received[0], "name:UP") || !strings.Contains(received[1], "name:DOWN") {
		t.Fatalf("unexpected commands received: %v", received)
	}
}

func TestInputControlClosesMouseConnWhenClientCloses(t *testing.T) {
	pointerSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	defer pointerSrv.Close()

	pointerURL := "ws://" + strings.TrimPrefix(pointerSrv.URL, "http://") + "/pointer"
	mainSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var req Message
			if err := json.Unmarshal(data, &req); err != nil {
				return
			}
			_ = conn.WriteJSON(Message{
				Type:    "response",
				ID:      req.ID,
				Payload: json.RawMessage(`{"socketPath":"` + pointerURL + `"}`),
			})
		}
	}))
	defer mainSrv.Close()

	client := NewClient(strings.TrimPrefix(mainSrv.URL, "http://"))
	if err := client.Connect(); err != nil {
		t.Fatalf("Connect() err = %v", err)
	}

	input := &InputControl{Control: Control{Client: client}}
	if err := input.ConnectInput(context.Background()); err != nil {
		t.Fatalf("ConnectInput() err = %v", err)
	}

	if err := client.Close(); err != nil {
		t.Fatalf("Close() err = %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		input.mouseConnMu.Lock()
		cleared := input.mouseConn == nil
		input.mouseConnMu.Unlock()
		if cleared {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("expected mouseConn to be cleared when client closed")
}
