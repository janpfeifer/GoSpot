package server

import (
	"context"
	"net"
	"net/http"
	"testing"
	"testing/synctest"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/janpfeifer/GoSpot/internal/game"
)

// pipeListener serves HTTP connections over net.Pipe
type pipeListener struct {
	ch   chan net.Conn
	done chan struct{}
}

func (l *pipeListener) Accept() (net.Conn, error) {
	select {
	case c := <-l.ch:
		return c, nil
	case <-l.done:
		return nil, net.ErrClosed
	}
}

func (l *pipeListener) Close() error {
	select {
	case <-l.done:
	default:
		close(l.done)
	}
	return nil
}

func (l *pipeListener) Addr() net.Addr { return &net.TCPAddr{} }

func TestLatencyCompensationClick(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		s := NewServerState()
		srv := &http.Server{Handler: http.HandlerFunc(s.HandleWS)}
		listener := &pipeListener{ch: make(chan net.Conn, 10), done: make(chan struct{})}
		defer listener.Close()
		go srv.Serve(listener)
		defer srv.Close()

		connectAndJoin := func(playerID, playerName string) *websocket.Conn {
			opts := &websocket.DialOptions{
				HTTPClient: &http.Client{
					Transport: &http.Transport{
						DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
							cli, srv := net.Pipe()
							listener.ch <- srv
							return cli, nil
						},
					},
				},
			}

			conn, _, err := websocket.Dial(ctx, "http://localhost/ws", opts)
			if err != nil {
				t.Fatalf("Dial error: %v", err)
			}

			joinMsg, _ := game.NewWsMessage(game.MsgTypeJoin, game.JoinMessage{
				TableID: "table1",
				Player: game.Player{
					ID:   playerID,
					Name: playerName,
				},
			})
			_ = wsjson.Write(ctx, conn, joinMsg)

			// Read and respond to the initial ping from the server.
			// This unblocks the server's HandleWS goroutine so it enters its read loop.
			var pingMsg game.WsMessage
			if err := wsjson.Read(ctx, conn, &pingMsg); err != nil {
				t.Fatalf("Failed to read initial ping: %v", err)
			}
			if pingMsg.Type != game.MsgTypePing {
				t.Fatalf("Expected ping message, got %s", pingMsg.Type)
			}
			p, err := pingMsg.Parse()
			if err != nil {
				t.Fatalf("Failed to parse ping: %v", err)
			}
			ping := p.(*game.PingMessage)
			pongMsg, _ := game.NewWsMessage(game.MsgTypePong, game.PongMessage{
				ServerTime: ping.ServerTime,
				ClientTime: time.Now().UnixNano(),
			})
			_ = wsjson.Write(ctx, conn, pongMsg)

			return conn
		}

		conn1 := connectAndJoin("p1", "FastPlayer")
		conn2 := connectAndJoin("p2", "SlowPlayer")

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

		synctest.Wait()

		startMsg, _ := game.NewWsMessage(game.MsgTypeStart, nil)
		_ = wsjson.Write(ctx, conn1, startMsg)

		synctest.Wait()

		s.mu.Lock()
		table := s.Tables["table1"]
		if table == nil {
			t.Fatalf("Table not found")
		}

		var fastPlayer, slowPlayer *game.Player
		for _, p := range table.Players {
			if p.ID == "p1" {
				p.Latency = 10 * time.Millisecond
				fastPlayer = p
			} else {
				p.Latency = 500 * time.Millisecond
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
		s.mu.Unlock()

		if fastMatch == -1 || slowMatch == -1 {
			t.Fatalf("Could not find matching cards")
		}

		fastClickMsg, _ := game.NewWsMessage(game.MsgTypeClick, game.ClickMessage{Symbol: fastMatch})
		slowClickMsg, _ := game.NewWsMessage(game.MsgTypeClick, game.ClickMessage{Symbol: slowMatch})

		// Fast player clicks first
		_ = wsjson.Write(ctx, conn1, fastClickMsg)

		// Advance time by 50ms
		time.Sleep(50 * time.Millisecond)

		// Slow player clicks
		_ = wsjson.Write(ctx, conn2, slowClickMsg)

		// Wait for both to be recorded as PendingClick
		synctest.Wait()

		// Enough time for timer to fire
		time.Sleep(1 * time.Second)
		synctest.Wait()

		s.mu.Lock()
		defer s.mu.Unlock()

		if table.Round != 2 {
			t.Fatalf("Expected round to increment to 2, got %d", table.Round)
		}

		if len(slowPlayer.Hand) >= len(fastPlayer.Hand) {
			t.Fatalf("Slow player should have won, but fast player won. Slow hand len: %d, Fast hand len: %d", len(slowPlayer.Hand), len(fastPlayer.Hand))
		}
	})
}
