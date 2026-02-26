package server

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/janpfeifer/GoSpot/internal/game"
)

func TestTableWebsocket(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	started := make(chan *ServerState, 1)
	go Run(ctx, "", started)
	serverState := <-started
	wsURL := "ws://" + serverState.Address + "/ws"

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

		// Handle initial Ping from server
		var pingMsg game.WsMessage
		if err := wsjson.Read(ctx, conn, &pingMsg); err != nil {
			conn.CloseNow()
			return nil, err
		}
		if pingMsg.Type != game.MsgTypePing {
			conn.CloseNow()
			return nil, err
		}

		p, err := pingMsg.Parse()
		if err != nil {
			conn.CloseNow()
			return nil, err
		}
		ping, ok := p.(*game.PingMessage)
		if !ok {
			conn.CloseNow()
			return nil, err
		}
		pongMsg, _ := game.NewWsMessage(game.MsgTypePong, game.PongMessage{
			ServerTime: ping.ServerTime,
			ClientTime: time.Now().UnixNano(),
		})
		if err := wsjson.Write(ctx, conn, pongMsg); err != nil {
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

	// Player 1 starts the game
	startMsg, _ := game.NewWsMessage(game.MsgTypeStart, nil)
	if err := wsjson.Write(ctx, conn1, startMsg); err != nil {
		t.Fatalf("Player 1 failed to send start message: %v", err)
	}

	checkCards := func(conn *websocket.Conn, name string) []int {
		var msg game.WsMessage
		var targetCard []int
		for {
			if err := wsjson.Read(ctx, conn, &msg); err != nil {
				t.Fatalf("%s failed to read message: %v", name, err)
			}
			if msg.Type == game.MsgTypeUpdate {
				p, err := msg.Parse()
				if err != nil {
					t.Fatalf("%s: Failed to parse update payload: %v", name, err)
				}
				updateMsg, ok := p.(*game.UpdateMessage)
				if !ok {
					t.Fatalf("%s: Expected UpdateMessage, got: %T", name, p)
				}

				if len(updateMsg.TopCard) == 0 {
					t.Fatalf("%s received empty top card", name)
				}
				if len(updateMsg.TargetCard) == 0 {
					t.Fatalf("%s received empty target card", name)
				}
				targetCard = updateMsg.TargetCard
				return targetCard
			}
		}
	}

	target1 := checkCards(conn1, "Player 1")
	target2 := checkCards(conn2, "Player 2")

	if len(target1) != len(target2) {
		t.Fatalf("Target cards have different lengths: P1=%d, P2=%d", len(target1), len(target2))
	}
	for i := range target1 {
		if target1[i] != target2[i] {
			t.Fatalf("Target cards differ: P1=%v, P2=%v", target1, target2)
		}
	}
}

func TestHandleTestGame(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	started := make(chan *ServerState, 1)
	go func() {
		_ = Run(ctx, "", started)
	}()
	s := <-started

	req, err := http.NewRequest("GET", "http://"+s.Address+"/test/game", nil)
	if err != nil {
		t.Fatal(err)
	}

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	// Check the status code is what we expect (Redirect to table).
	if status := resp.StatusCode; status != http.StatusSeeOther {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusSeeOther)
	}

	// Check if the redirect location is correct
	location := resp.Header.Get("Location")
	if location != "/table/ThreeStooges" {
		t.Errorf("handler returned wrong redirect location: got %v want %v",
			location, "/table/ThreeStooges")
	}

	// Verify cookie is set
	cookies := resp.Cookies()
	var playerCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "gospot_player" {
			playerCookie = c
			break
		}
	}
	if playerCookie == nil {
		t.Fatalf("Expected gospot_player cookie to be set")
	}

	// Verify server state
	s.mu.RLock()
	defer s.mu.RUnlock()

	table, exists := s.Tables["ThreeStooges"]
	if !exists {
		t.Fatalf("Table ThreeStooges was not created in server state")
	}

	if table.Name != "ThreeStooges" {
		t.Errorf("Expected table name 'ThreeStooges', got '%s'", table.Name)
	}

	if len(table.Players) != 3 {
		t.Errorf("Expected 3 players, got %d", len(table.Players))
	}

	if !table.Started {
		t.Errorf("Expected game to be started")
	}

	if len(table.TargetCard) == 0 {
		t.Errorf("Expected target card to be dealt")
	}

	playerNames := map[string]bool{"Moe": true, "Larry": true, "Curly": true}
	for _, p := range table.Players {
		if !playerNames[p.Name] {
			t.Errorf("Unexpected player name: %s", p.Name)
		}
		if len(p.Hand) == 0 {
			t.Errorf("Expected player %s to have cards dealt", p.Name)
		}
	}
}
