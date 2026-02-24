package game

// Deck represents a full set of GoSpot cards.
// Each card is a slice of integers representing symbol IDs.
type Deck [][]int

// GenerateDeck creates a mathematically perfect deck based on a prime order.
// For standard gameplay, use order = 7 (yields 57 cards with 8 symbols each).
func GenerateDeck(order int) Deck {
	// A standard deck requires the order to be a prime number (2, 3, 5, 7, 11...)
	// Total cards and symbols will equal: order^2 + order + 1
	numCards := order*order + order + 1
	deck := make([][]int, 0, numCards)

	// ---------------------------------------------------------
	// STEP 1: Generate the "Infinity Line" (Card 0)
	// ---------------------------------------------------------
	card := make([]int, 0, order+1)
	card = append(card, 0)
	for i := 1; i <= order; i++ {
		card = append(card, i)
	}
	deck = append(deck, card)

	// ---------------------------------------------------------
	// STEP 2: Generate the "Vertical Lines" (Next 'order' Cards)
	// ---------------------------------------------------------
	for j := range order {
		card = make([]int, 0, order+1)
		card = append(card, 0)
		for k := range order {
			// Formula: (n + 1) + (n * j) + k
			symbol := (order + 1) + (order * j) + k
			card = append(card, symbol)
		}
		deck = append(deck, card)
	}

	// ---------------------------------------------------------
	// STEP 3: Generate the "Diagonal Lines" (Final 'order^2' Cards)
	// ---------------------------------------------------------
	for i := range order {
		for j := range order {
			card = make([]int, 0, order+1)
			card = append(card, i+1)
			for k := range order {
				// Formula: (n + 1) + (n * k) + ((i * k + j) % n)
				symbol := (order + 1) + (order * k) + ((i*k + j) % order)
				card = append(card, symbol)
			}
			deck = append(deck, card)
		}
	}

	return deck
}
