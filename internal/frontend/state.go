package frontend

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

	// Music state
	SoundEnabled bool
	Music        app.Value
	musicStop    chan struct{}
	musicSrc     string

	// Game state (individual)
	TopCard    []int
	TargetCard []int
	Round      int
	ScoringIDs []string

	// Listeners for state updates
	Listeners map[string]func()
}

var State *GlobalClientState

func (s *GlobalClientState) ToggleSound() {
	s.SoundEnabled = !s.SoundEnabled
	klog.Infof("ToggleSound: SoundEnabled is now %v", s.SoundEnabled)
	s.SyncMusic()
	s.Notify()
}

func (s *GlobalClientState) PlaySound(url string) {
	// SoundEnabled is only for the music, not for sound effects
	// if !s.SoundEnabled {
	// 	return
	// }

	// Create a new Audio element for the sound effect
	audio := app.Window().Get("document").Call("createElement", "audio")
	audio.Set("src", url)

	// Play the sound (fire and forget)
	promise := audio.Call("play")
	if promise.Truthy() {
		promise.Call("then", app.FuncOf(func(this app.Value, args []app.Value) any {
			audio.Set("volume", 1.0)
			return nil
		}))
		promise.Call("catch", app.FuncOf(func(this app.Value, args []app.Value) any {
			klog.Errorf("PlaySound: Failed to play %s: %v", url, args[0])
			return nil
		}))
	}
}

func (s *GlobalClientState) SyncMusic() {
	if app.IsServer {
		return
	}

	if !s.SoundEnabled {
		if s.musicStop != nil {
			klog.Infof("SyncMusic: Stopping music loop (SoundEnabled=false)")
			close(s.musicStop)
			s.musicStop = nil
			if s.Music != nil && s.Music.Truthy() {
				s.Music.Call("pause")
				s.Music.Call("remove")
			}
		}
		return
	}

	var targetSrc string
	if s.Table != nil && s.Table.Started {
		targetSrc = "/web/sounds/Glimmering_Gauntlet.mp3"
	} else {
		targetSrc = "/web/sounds/Xylophonic_Cascade.mp3"
	}

	if s.musicStop == nil || s.musicSrc != targetSrc {
		if s.musicStop != nil {
			close(s.musicStop)
			if s.Music != nil && s.Music.Truthy() {
				s.Music.Call("pause")
				s.Music.Call("remove")
			}
		}
		s.musicSrc = targetSrc
		s.musicStop = make(chan struct{})
		go s.musicLoop(s.musicStop, targetSrc)
	}
}

func (s *GlobalClientState) musicLoop(stop chan struct{}, src string) {
	klog.Infof("musicLoop: Started")
	for {
		if s.Music == nil || !s.Music.Truthy() {
			klog.Infof("musicLoop: Creating audio element")
			s.Music = app.Window().Get("document").Call("createElement", "audio")
			s.Music.Get("style").Set("display", "none")
			app.Window().Get("document").Get("body").Call("appendChild", s.Music)
		}
		s.Music.Set("src", src)

		klog.Infof("musicLoop: Attempting to play...")
		promise := s.Music.Call("play")

		started := make(chan bool, 1)
		if promise.Truthy() {
			var onSuccess, onFailure app.Func
			onSuccess = app.FuncOf(func(this app.Value, args []app.Value) any {
				klog.Infof("musicLoop: Play started successfully")
				s.Music.Set("volume", 0.04)
				select {
				case started <- true:
				default:
				}
				onSuccess.Release()
				onFailure.Release()
				return nil
			})
			onFailure = app.FuncOf(func(this app.Value, args []app.Value) any {
				klog.Errorf("musicLoop: Play failed (likely autoplay block): %v", args[0])
				select {
				case started <- false:
				default:
				}
				onSuccess.Release()
				onFailure.Release()
				return nil
			})
			promise.Call("then", onSuccess)
			promise.Call("catch", onFailure)
		} else {
			klog.Warning("musicLoop: Play did not return a promise")
			started <- true
		}

		var ok bool
		select {
		case <-stop:
			return
		case ok = <-started:
		case <-time.After(5 * time.Second):
			klog.Warning("musicLoop: Play promise timed out")
			ok = false
		}

		if ok {
			// Wait for the song to finish or stop signal
			ended := make(chan struct{})
			onEnded := app.FuncOf(func(this app.Value, args []app.Value) any {
				select {
				case ended <- struct{}{}:
				default:
				}
				return nil
			})
			s.Music.Call("addEventListener", "ended", onEnded)

			select {
			case <-stop:
				s.Music.Call("removeEventListener", "ended", onEnded)
				onEnded.Release()
				return
			case <-ended:
				s.Music.Call("removeEventListener", "ended", onEnded)
				onEnded.Release()
				klog.Infof("musicLoop: Finished playing")
			}

			// 3 second pause
			klog.Infof("musicLoop: Pausing for 3 seconds")
			select {
			case <-stop:
				return
			case <-time.After(3 * time.Second):
			}
		} else {
			// If failed, wait a bit before retrying (avoid busy loop)
			klog.Infof("musicLoop: Retrying in 5 seconds...")
			select {
			case <-stop:
				return
			case <-time.After(5 * time.Second):
			}
		}
	}
}

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
			Player:       &game.Player{},
			Listeners:    make(map[string]func()),
			SoundEnabled: true,
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
		s.SyncMusic()
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

		State.Error = errMsg.Message
		State.Table = nil
		s.SyncMusic()
		s.Notify()

	case game.MsgTypeUpdate:
		p, err := msg.Parse()
		if err != nil {
			klog.Errorf("handleMessage: Failed to parse update message: %v", err)
			return
		}
		updateMsg, ok := p.(*game.UpdateMessage)
		if !ok {
			return
		}

		klog.Infof("handleMessage: Game update received. TopCard: %v, TargetCard: %v, Round: %d, ScoringIDs: %v",
			updateMsg.TopCard, updateMsg.TargetCard, updateMsg.Round, updateMsg.ScoringIDs)
		State.TopCard = updateMsg.TopCard
		State.TargetCard = updateMsg.TargetCard
		State.Round = updateMsg.Round
		State.ScoringIDs = updateMsg.ScoringIDs
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

// SendClick sends a click message to the server
func (s *GlobalClientState) SendClick(symbol int) {
	if s.Conn == nil {
		return
	}
	msg, err := game.NewWsMessage(game.MsgTypeClick, game.ClickMessage{Symbol: symbol})
	if err != nil {
		klog.Errorf("SendClick: Failed to create click message: %v", err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	wsjson.Write(ctx, s.Conn, msg)
}
