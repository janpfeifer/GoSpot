package server

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/janpfeifer/GoSpot/internal/game"
)

func TestLatencyCompensationClick(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		startAddrChan := make(chan *ServerState, 1)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go Run(ctx, NetPipeAddr, startAddrChan)
		serverState := <-startAddrChan
		startAddr := serverState.Address

		const tableName = "test_table"

		wsURL := "ws://" + startAddr + "/ws"

		var conn1, conn2 *websocket.Conn
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			conn, err := testConnectAndJoin(ctx, serverState, wsURL, tableName, "p1", "FastPlayer", 0, 0)
			if err != nil {
				t.Errorf("FastPlayer failed to join: %v", err)
			}
			conn1 = conn
			fmt.Println("FastPlayer connected")
		}()
		go func() {
			defer wg.Done()
			conn, err := testConnectAndJoin(ctx, serverState, wsURL, tableName, "p2", "SlowPlayer", 0, 10*time.Millisecond)
			if err != nil {
				t.Errorf("SlowPlayer failed to join: %v", err)
			}
			conn2 = conn
			fmt.Println("SlowPlayer connected")
		}()
		wg.Wait()
		fmt.Println("Connected")

		// Drain incoming messages from both connections in background goroutines.
		// The server sends state broadcasts and pings that must be consumed to
		// prevent blocking server-side writes on the unbuffered net.Pipe.
		drainConn := func(conn *websocket.Conn) {
			for {
				var msg game.WsMessage
				if err := wsjson.Read(ctx, conn, &msg); err != nil {
					return
				}
			}
		}
		go drainConn(conn1)
		go drainConn(conn2)

		fmt.Println("Sending start message")
		startMsg, _ := game.NewWsMessage(game.MsgTypeStart, nil)
		_ = wsjson.Write(ctx, conn1, startMsg)

		// Wait for table to start
		for {
			serverState.mu.Lock()
			table := serverState.Tables[tableName]
			if table != nil && table.Started {
				serverState.mu.Unlock()
				break
			}
			serverState.mu.Unlock()
			time.Sleep(1 * time.Millisecond)
		}

		serverState.mu.Lock()
		table := serverState.Tables[tableName]
		if table == nil {
			t.Fatalf("Table not found")
		}
		synctest.Wait()
		fmt.Printf("- Table: %s\n", table)

		var fastPlayer, slowPlayer *game.Player
		for _, p := range table.Players {
			if p.ID == "p1" {
				p.Latency = 2 * time.Millisecond
				fastPlayer = p
			} else {
				p.Latency = 10 * time.Millisecond
				slowPlayer = p
			}
		}

		if !table.Started {
			t.Fatalf("Game should have started")
		}

		targetCard := table.TargetCard
		findMatch := func(hand []int) int {
			if len(hand) == 0 {
				return -1
			}
			for _, sym := range targetCard {
				for _, hSym := range hand {
					if sym == hSym {
						return sym
					}
				}
			}
			return -1
		}

		fastMatch := findMatch(fastPlayer.Hand[0])
		slowMatch := findMatch(slowPlayer.Hand[0])
		serverState.mu.Unlock()

		if fastMatch == -1 || slowMatch == -1 {
			t.Fatalf("Could not find matching cards")
		}

		fastClickMsg, _ := game.NewWsMessage(game.MsgTypeClick, game.ClickMessage{Symbol: fastMatch})
		slowClickMsg, _ := game.NewWsMessage(game.MsgTypeClick, game.ClickMessage{Symbol: slowMatch})

		// Fast player clicks first
		_ = wsjson.Write(ctx, conn1, fastClickMsg)

		// Advance time by 5ms
		time.Sleep(5 * time.Millisecond)

		// Slow player clicks
		_ = wsjson.Write(ctx, conn2, slowClickMsg)

		// Enough time for timer to fire
		time.Sleep(100 * time.Millisecond)

		serverState.mu.Lock()
		defer serverState.mu.Unlock()

		if table.Round != 2 {
			t.Fatalf("Expected round to increment to 2, got %d", table.Round)
		}

		if len(slowPlayer.Hand) >= len(fastPlayer.Hand) {
			t.Fatalf("Slow player should have won, but fast player won. Slow hand len: %d, Fast hand len: %d", len(slowPlayer.Hand), len(fastPlayer.Hand))
		}
		synctest.Wait()
	})
}
