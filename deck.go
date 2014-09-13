package main

import "fmt"

type Deck map[string]string

//generateAllHands returns all possible five-card hands from the
//five table cards and two hole cards.
func generateAllHands(deck Deck, p Player) []Hand {
	allCards := make([]string, 0)
	for card, location := range deck {
		if location == string(p.guid) || location == "FLOP" || location == "TURN" || location == "RIVER" {
			allCards = append(allCards, card)
		}
	}
	if len(allCards) != 7 {
		panicMsg := fmt.Sprintf("Should have 7 cards. Have %v", allCards)
		panic(panicMsg)
	}
	return nChooseK(allCards, 5)
}

//nChooseK returns all k-length combinations of a slice of strings.
func nChooseK(allCards []string, k int) []Hand {
	allHands := make([]Hand, 0)
	if k == 0 {
		return make([]Hand, 1)
	}

	for i := 0; i < len(allCards)-k+1; i++ {
		combinations := nChooseK(allCards[i+1:], k-1)
		for _, single_combination := range combinations {
			single_combination = append(single_combination, allCards[i])
			allHands = append(allHands, single_combination)
		}
	}

	return allHands
}
