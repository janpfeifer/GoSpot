package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/janpfeifer/GoSpot/internal/game"
)

func TestTableWebsocket(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s := NewServerState()
	server := httptest.NewServer(http.HandlerFunc(s.HandleWS))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Helper to connect and join
	connectAndJoin := func(playerID, playerName string, symbol int, tableID string) (*websocket.Conn, error) {
		conn, _, err := websocket.Dial(ctx, wsURL, nil)
		if err != nil {
			return nil, err
		}

		joinMsg, err := game.NewWsMessage(game.MsgTypeJoin, game.JoinMessage{
			TableID: tableID,
			Player: game.Player{
				ID:     playerID,
				Name:   playerName,
				Symbol: symbol,
			},
		})
		if err != nil {
			conn.CloseNow()
			return nil, err
		}
		if err := wsjson.Write(ctx, conn, joinMsg); err != nil {
			conn.CloseNow()
			return nil, err
		}
		return conn, nil
	}

	tableID := "test-table"

	// Player 1 Joins
	conn1, err := connectAndJoin("p1", "Alice", 1, tableID)
	if err != nil {
		t.Fatalf("Player 1 failed to join: %v", err)
	}
	defer conn1.CloseNow()

	// Wait for state from P1 joining
	var msg1 game.WsMessage
	if err := wsjson.Read(ctx, conn1, &msg1); err != nil {
		t.Fatalf("Player 1 failed to read first state: %v", err)
	}
	if msg1.Type != game.MsgTypeState {
		t.Fatalf("Expected State message, got %s", msg1.Type)
	}

	// Player 2 Joins
	conn2, err := connectAndJoin("p2", "Bob", 2, tableID)
	if err != nil {
		t.Fatalf("Player 2 failed to join: %v", err)
	}
	defer conn2.CloseNow()

	// Both should receive updated state with 2 players
	checkState := func(conn *websocket.Conn, name string) {
		var msg game.WsMessage
		// We might need to read until we get the state with 2 players
		// since P1 might get its own state first.
		for {
			if err := wsjson.Read(ctx, conn, &msg); err != nil {
				t.Fatalf("%s failed to read state: %v", name, err)
			}
			if msg.Type != game.MsgTypeState {
				continue
			}

			p, err := msg.Parse()
			if err != nil {
				t.Fatalf("%s: Failed to parse payload: %v", name, err)
			}
			stateMsg, ok := p.(*game.StateMessage)
			if !ok {
				t.Fatalf("%s: Expected StateMessage, got: %T", name, p)
			}

			if len(stateMsg.Table.Players) == 2 {
				// Success!
				return
			}
			if len(stateMsg.Table.Players) > 2 {
				t.Fatalf("%s: Expected 2 players, got %d", name, len(stateMsg.Table.Players))
			}
			// If 1 player, keep reading
		}
	}

	checkState(conn1, "Player 1")
	checkState(conn2, "Player 2")
}
