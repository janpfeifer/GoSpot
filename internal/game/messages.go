package game

import (
	"encoding/json"
	"fmt"
)

// Message type for WebSocket communication between client and server.
type MessageType string

const (
	MsgTypeJoin   MessageType = "join"   // Client wants to join a table
	MsgTypeState  MessageType = "state"  // Server sends full table state
	MsgTypeStart  MessageType = "start"  // Client wants to start the game
	MsgTypeCancel MessageType = "cancel" // Client (creator) wants to cancel/destroy the table
	MsgTypePing   MessageType = "ping"   // Server pings client to measure RTT
	MsgTypePong   MessageType = "pong"   // Client responds to ping
	MsgTypeUpdate MessageType = "update" // Server sends game update (top card, target card)
	MsgTypeClick  MessageType = "click"  // Client clicks a symbol
	MsgTypeError  MessageType = "error"  // Server sends an error message
	MsgTypeChat   MessageType = "chat"   // (Optional) simple chat
)

// WsMessage represents a WebSocket message.
type WsMessage struct {
	Type    MessageType     `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// NewWsMessage creates a new WsMessage with a marshaled payload.
func NewWsMessage(msgType MessageType, payload interface{}) (WsMessage, error) {
	if payload == nil {
		return WsMessage{Type: msgType}, nil
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return WsMessage{}, fmt.Errorf("failed to marshal payload: %w", err)
	}
	return WsMessage{
		Type:    msgType,
		Payload: payloadBytes,
	}, nil
}

// Parse unmarshals the message payload into one of the message types (JoinMessage, StateMessage, etc.)
func (m *WsMessage) Parse() (any, error) {
	var target any
	switch m.Type {
	case MsgTypeJoin:
		target = &JoinMessage{}
	case MsgTypeState:
		target = &StateMessage{}
	case MsgTypeStart:
		target = &StartMessage{}
	case MsgTypeCancel:
		target = &CancelMessage{}
	case MsgTypePing:
		target = &PingMessage{}
	case MsgTypePong:
		target = &PongMessage{}
	case MsgTypeUpdate:
		target = &UpdateMessage{}
	case MsgTypeClick:
		target = &ClickMessage{}
	case MsgTypeError:
		target = &ErrorMessage{}
	default:
		return nil, fmt.Errorf("unknown message type: %s", m.Type)
	}

	if len(m.Payload) == 0 {
		return target, nil
	}

	err := json.Unmarshal(m.Payload, target)
	return target, err
}

// JoinMessage is the payload for MsgTypeJoin
type JoinMessage struct {
	TableID string `json:"table_id"`
	Player  Player `json:"player"`
}

// StartMessage: empty.
type StartMessage struct{}

// CancelMessage: empty.
type CancelMessage struct{}

// StateMessage is the payload for MsgTypeState
type StateMessage struct {
	Table Table `json:"table"`
}

// UpdateMessage is the payload for MsgTypeUpdate
type UpdateMessage struct {
	TargetCard []int  `json:"target_card"` // Current card on the table
	TopCard    []int  `json:"top_card"`    // Player's top card
	Round      int    `json:"round"`       // Current round number
	WinnerID   string `json:"winner_id"`   // Player ID who won the previous round
}

// ClickMessage is the payload for MsgTypeClick
type ClickMessage struct {
	Symbol int `json:"symbol"` // The symbol ID that was clicked
}

// PingMessage is the payload for MsgTypePing
type PingMessage struct {
	ServerTime int64 `json:"server_time"` // Nanoseconds since Unix epoch
}

// PongMessage is the payload for MsgTypePong
type PongMessage struct {
	ServerTime int64 `json:"server_time"` // Same value from Ping
	ClientTime int64 `json:"client_time"` // Client's own timestamp
}

// ErrorMessage is the payload for MsgTypeError
type ErrorMessage struct {
	Message string `json:"message"`
}
