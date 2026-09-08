package webostv

import (
	"testing"

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
