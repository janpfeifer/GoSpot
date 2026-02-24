package lobby

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/janpfeifer/GoSpot/internal/game"
	"github.com/maxence-charriere/go-app/v10/pkg/app"
	"k8s.io/klog/v2"
)

// GlobalClientState manages connection, current player info, and table state
type GlobalClientState struct {
	Player *game.Player
	Table  *game.Table
	Conn   *websocket.Conn

	// Login State (persistent across re-renders)
	PendingName string
	SymbolID    int
	ShowSymbols bool
}

var State *GlobalClientState

func InitState() {
	if State == nil {
		klog.V(1).Infof("InitState: creating new state (was nil)")
		State = &GlobalClientState{
			Player: &game.Player{},
		}
		rand.Seed(time.Now().UnixNano())
		State.SymbolID = rand.Intn(57) + 1
	} else {
		klog.V(1).Infof("InitState: state already exists")
	}
}

// ConnectWS connects to the server and sends a join message.
func (s *GlobalClientState) ConnectWS(tableID string) error {
	if s.Conn != nil {
		s.Conn.CloseNow()
	}

	wsURL := fmt.Sprintf("ws://%s/ws", app.Window().URL().Host)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		return fmt.Errorf("dial failed: %w", err)
	}

	s.Conn = conn

	// Send join message
	joinMsg := game.WsMessage{
		Type: game.MsgTypeJoin,
		Payload: game.JoinPayload{
			TableID: tableID,
			Player:  *s.Player,
		},
	}
	if err := wsjson.Write(ctx, conn, joinMsg); err != nil {
		return fmt.Errorf("failed to send join: %w", err)
	}

	// Start reading loop in background
	go s.readLoop(conn)

	return nil
}

func (s *GlobalClientState) readLoop(conn *websocket.Conn) {
	ctx := context.Background()
	for {
		var msg game.WsMessage
		err := wsjson.Read(ctx, conn, &msg)
		if err != nil {
			klog.Errorf("WS read error: %v", err)
			break
		}

		s.handleMessage(msg)
	}
}

func (s *GlobalClientState) handleMessage(msg game.WsMessage) {
	switch msg.Type {
	case game.MsgTypeState:
		payloadBytes, err := json.Marshal(msg.Payload)
		if err != nil {
			return
		}
		var statePayload game.StatePayload
		if err := json.Unmarshal(payloadBytes, &statePayload); err != nil {
			return
		}

		State.Table = &statePayload.Table
		app.Window().Get("document").Call("dispatchEvent", app.Window().Get("Event").New("table_update"))
	}
}

// SendStart sends a start message to the server
func (s *GlobalClientState) SendStart() {
	if s.Conn == nil {
		return
	}
	msg := game.WsMessage{
		Type: game.MsgTypeStart,
	}
	wsjson.Write(context.Background(), s.Conn, msg)
}
