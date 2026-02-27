package game

// Version of the game.
// Bumping this number will eventually make clients reload the WASM.
//
// If you set this to an empty string, a random version number will be
// used, and force the reload of the WASM on every restart (the reload
// still only happens after the first page is loaded, so there is a delay).
// This is useful during development.
var Version = "v0.1.0"

// BonusDiscards is the number of cards to discard when a player matches
// a symbol in the card that also matches their own symbol.
var BonusDiscards = 3
