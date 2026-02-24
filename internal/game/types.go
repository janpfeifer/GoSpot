package game

// Player represents a user in the lobby or game.
type Player struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Symbol    int    `json:"symbol"`     // Symbol ID chosen by the player
	Score     int    `json:"score"`      // Number of cards
	InPenalty bool   `json:"in_penalty"` // True if player clicked wrong symbol
}

// Table represents a game room.
type Table struct {
	ID      string             `json:"id"`
	Name    string             `json:"name"`
	Players map[string]*Player `json:"players"` // Players currently at the table
	Started bool               `json:"started"` // True if game has started
}

// Message type for WebSocket communication between client and server.
type MessageType string

const (
	MsgTypeJoin  MessageType = "join"  // Client wants to join a table
	MsgTypeState MessageType = "state" // Server sends full table state
	MsgTypeStart MessageType = "start" // Client wants to start the game
	MsgTypeError MessageType = "error" // Server sends an error message
	MsgTypeChat  MessageType = "chat"  // (Optional) simple chat
)

// WsMessage represents a WebSocket message.
type WsMessage struct {
	Type    MessageType `json:"type"`
	Payload interface{} `json:"payload,omitempty"`
}

// JoinPayload is the payload for MsgTypeJoin
type JoinPayload struct {
	TableID string `json:"table_id"`
	Player  Player `json:"player"`
}

// StatePayload is the payload for MsgTypeState
type StatePayload struct {
	Table Table `json:"table"`
}

// ErrorPayload is the payload for MsgTypeError
type ErrorPayload struct {
	Message string `json:"message"`
}
