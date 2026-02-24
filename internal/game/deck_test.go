package game

import (
	"fmt"
	"testing"
)

func TestDeckMatchingSymbol(t *testing.T) {
	orders := []int{2, 3, 5, 7, 11} // Prime orders

	for _, order := range orders {
		t.Run(fmt.Sprintf("%d", order), func(t *testing.T) {
			deck := GenerateDeck(order)

			// Check that each card has exactly one matching symbol with each other card
			for i := 0; i < len(deck); i++ {
				for j := i + 1; j < len(deck); j++ {
					matches := countMatches(deck[i], deck[j])
					if matches != 1 {
						t.Errorf("Order %d: Expected exactly 1 matching symbol between card %d %v and card %d %v, got %d",
							order, i, deck[i], j, deck[j], matches)
					}
				}
			}
		})
	}
}

func countMatches(card1, card2 []int) int {
	matches := 0
	for _, s1 := range card1 {
		for _, s2 := range card2 {
			if s1 == s2 {
				matches++
			}
		}
	}
	return matches
}
