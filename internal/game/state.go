package game

import (
	"fmt"
	"strings"
	"time"
)

// Player represents a user in the lobby or game.
type Player struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Symbol    int           `json:"symbol"`     // Symbol ID chosen by the player
	Score     int           `json:"score"`      // Number of cards
	InPenalty bool          `json:"in_penalty"` // True if player clicked wrong symbol
	Latency   time.Duration `json:"latency"`    // Measured round-trip time / 2 (one-way estimate)
	Hand      [][]int       `json:"-"`          // Cards in player's hand (not sent in full state)
}

// PendingClick represents a client click that is currently delayed waiting to be processed.
type PendingClick struct {
	PlayerID    string
	ProcessTime time.Time
	Symbol      int
	Round       int // The round in which this click was made
}

// Table represents a game room.
type Table struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Players      []*Player     `json:"players"`     // Players currently at the table
	Started      bool          `json:"started"`     // True if game has started
	TargetCard   []int         `json:"target_card"` // Current card on the table
	Round        int           `json:"round"`       // Current round number
	PendingClick *PendingClick `json:"-"`           // Server tracking of pending click
	ClickTimer   *time.Timer   `json:"-"`           // Server timer to process the click
}

func (t *Table) String() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Table %s: name=%s, started=%t, round=%d, targetCard=%v, players: ", t.ID, t.Name, t.Started, t.Round, t.TargetCard)
	for _, p := range t.Players {
		fmt.Fprintf(&sb, "%s (%d, %d cards), ", p.Name, p.Score, len(p.Hand))
	}
	return sb.String()
}
