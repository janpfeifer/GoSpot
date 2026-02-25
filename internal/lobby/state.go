package lobby

import (
	"context"
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
	Error  string
	Conn   *websocket.Conn

	// Login State (persistent across re-renders)
	PendingName string
	SymbolID    int
	ShowSymbols bool

	// Listeners for state updates
	Listeners map[string]func()
}

var State *GlobalClientState

func (s *GlobalClientState) Notify() {
	klog.Infof("GlobalClientState: Notifying %d listeners", len(s.Listeners))
	for _, l := range s.Listeners {
		if l != nil {
			l()
		}
	}
}

func InitState() {
	if State == nil {
		klog.V(1).Infof("InitState: creating new state (was nil)")
		State = &GlobalClientState{
			Player:    &game.Player{},
			Listeners: make(map[string]func()),
		}
		// rand.Seed is deprecated in Go 1.20+, but we can still use it or use rand.New(rand.NewSource(...))
		// For now keeping it simple as this is Wasm.
		State.SymbolID = rand.Intn(57) // 0 to 56
	} else {
		klog.V(1).Infof("InitState: state already exists")
	}
}

// ConnectWS connects to the server and sends a join message.
func (s *GlobalClientState) ConnectWS(tableID string) error {
	if s.Conn != nil {
		klog.Infof("ConnectWS: Closing existing connection")
		s.Conn.CloseNow()
	}

	wsURL := fmt.Sprintf("ws://%s/ws", app.Window().URL().Host)
	klog.Infof("ConnectWS: Connecting to %s (Table: %s)", wsURL, tableID)

	// We use a context that lasts for the duration of the connection setup.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		klog.Errorf("ConnectWS: Dial failed: %v", err)
		return fmt.Errorf("dial failed: %w", err)
	}

	s.Conn = conn
	klog.Infof("ConnectWS: Connected, sending Join message...")

	// Send join message
	joinMsg, err := game.NewWsMessage(game.MsgTypeJoin, game.JoinMessage{
		TableID: tableID,
		Player:  *s.Player,
	})
	if err != nil {
		klog.Errorf("ConnectWS: Failed to create join message: %v", err)
		return fmt.Errorf("failed to create join message: %w", err)
	}

	if err := wsjson.Write(ctx, conn, joinMsg); err != nil {
		klog.Errorf("ConnectWS: Failed to send join: %v", err)
		return fmt.Errorf("failed to send join: %w", err)
	}

	klog.Infof("ConnectWS: Join message sent. Starting read loop.")
	// Start reading loop in background
	go s.readLoop(conn)

	return nil
}

func (s *GlobalClientState) readLoop(conn *websocket.Conn) {
	ctx := context.Background()
	klog.Infof("readLoop: started")
	for {
		var msg game.WsMessage
		err := wsjson.Read(ctx, conn, &msg)
		if err != nil {
			klog.Errorf("readLoop: WS read error: %v", err)
			break
		}

		klog.Infof("readLoop: received message type: %s", msg.Type)
		s.handleMessage(msg)
	}
}

func (s *GlobalClientState) handleMessage(msg game.WsMessage) {
	switch msg.Type {
	case game.MsgTypeState:
		p, err := msg.Parse()
		if err != nil {
			klog.Errorf("handleMessage: Failed to parse state message: %v", err)
			return
		}
		stateMsg, ok := p.(*game.StateMessage)
		if !ok {
			klog.Errorf("handleMessage: Expected StateMessage, got: %T", p)
			return
		}

		klog.Infof("handleMessage: State updated. Players: %d", len(stateMsg.Table.Players))
		State.Table = &stateMsg.Table
		State.Error = ""
		s.Notify()

	case game.MsgTypeError:
		p, err := msg.Parse()
		if err != nil {
			klog.Errorf("handleMessage: Failed to parse error message: %v", err)
			return
		}
		errMsg, ok := p.(*game.ErrorMessage)
		if !ok {
			klog.Errorf("handleMessage: Expected ErrorMessage, got: %T", p)
			return
		}

		klog.Infof("handleMessage: Error received: %s", errMsg.Message)
		State.Error = errMsg.Message
		State.Table = nil
		s.Notify()

	case game.MsgTypePing:
		p, err := msg.Parse()
		if err != nil {
			klog.Errorf("handleMessage: Failed to parse ping message: %v", err)
			return
		}
		ping, ok := p.(*game.PingMessage)
		if !ok {
			return
		}
		pongMsg, _ := game.NewWsMessage(game.MsgTypePong, game.PongMessage{
			ServerTime: ping.ServerTime,
			ClientTime: time.Now().UnixNano(),
		})
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
		defer cancel()
		wsjson.Write(ctx, s.Conn, pongMsg)
	}
}

// SendStart sends a start message to the server
func (s *GlobalClientState) SendStart() {
	if s.Conn == nil {
		return
	}
	msg, err := game.NewWsMessage(game.MsgTypeStart, nil)
	if err != nil {
		klog.Errorf("SendStart: Failed to create start message: %v", err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	wsjson.Write(ctx, s.Conn, msg)
}

// SendCancel sends a cancel message to the server
func (s *GlobalClientState) SendCancel() {
	if s.Conn == nil {
		return
	}
	msg, err := game.NewWsMessage(game.MsgTypeCancel, nil)
	if err != nil {
		klog.Errorf("SendCancel: Failed to create cancel message: %v", err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	wsjson.Write(ctx, s.Conn, msg)
}
