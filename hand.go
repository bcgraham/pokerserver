package main

import (
	"strings"
	// "fmt"
)

type Hand []string

var RANKS string = "--23456789TJQKA"

//Returns nil if hands is empty
func findWinningHands(hands []Hand) (winners []Hand) {
	if len(hands) == 0 {
		return nil
	} else if len(hands) == 1 {
		return hands
	}

	winners = append(make([]Hand, 0), hands[0])
	max_hand, max_count := winners[0].handRank()
	for _, h := range hands[1:] {
		test_hand, test_count := h.handRank()
		if test_hand == max_hand && isEq(test_count, max_count) {
			//The hands are equal
			winners = append(winners, h)
		} else if (test_hand > max_hand) ||
			((test_hand == max_hand) && gt(test_count, max_count)) {
			//test_hand > max_hand
			winners = append(make([]Hand, 0), h)
			max_hand, max_count = h.handRank()
		} else if (test_hand < max_hand) ||
			((test_hand == max_hand) && less(test_count, max_count)) {
			//test_hand < max_hand
			continue
		}
	}
	return winners
}

// less returns true if a is less than b
func less(a, b []int) bool {
	if len(a) != len(b) {
		panic("len(a) != len(b) in less comparison")
	}
	for i := range a {
		if a[i] < b[i] {
			return true
		} else if a[i] > b[i] {
			return false
		}
	}
	return false
}

// gt returns true if a is greater than b
func gt(a, b []int) bool {
	if len(a) != len(b) {
		panic("len(a) != len(b) in gt comparison")
	}
	for i := range a {
		if a[i] > b[i] {
			return true
		} else if a[i] < b[i] {
			return false
		}
	}
	return false
}

func isEq(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (h Hand) rawRanksAndSuits() (ranks [5]int, suits [5]rune) {
	for i, card := range h {
		numeral := string(card[0])
		ranks[i] = strings.Index(RANKS, numeral)
		if ranks[i] == -1 {
			panic("can't find card")
		}
		suits[i] = rune(card[1])
	}
	return ranks, suits
}

func countAndCombine(rawRanks [5]int) (counts []int, countRanks []int) {
	var hash [15]int
	for _, n := range rawRanks {
		if n < 2 || n > 14 {
			panic("ranking out of expected range")
		}
		hash[n]++
	}

	for q := 4; q >= 1; q-- {
		for i := len(hash) - 1; i >= 2; i-- {
			if hash[i] == q {
				counts = append(counts, q)
				countRanks = append(countRanks, i)
			}
		}
	}

	return counts, countRanks
}

func isStraight(countRanks []int) bool {
	if len(countRanks) != 5 {
		return false
	}
	if (countRanks[0] - countRanks[4]) != 4 {
		return false
	}
	return true
}

func isFlush(suits [5]rune) bool {
	suit := suits[0]
	for _, v := range suits[1:] {
		if suit != v {
			return false
		}
	}
	return true
}

//Returns the ranking for the hand and the card rankings
//'7 T 7 9 7' => handRank = 3 and cardRanks = [7, 10, 9]
// cardRanks is ordered by count first, then by card rank
func (h Hand) handRank() (handRank uint, countRanks []int) {
	rawRanks, suits := h.rawRanksAndSuits()
	counts, countRanks := countAndCombine(rawRanks)
	if isEq(countRanks, []int{14, 5, 4, 3, 2}) {
		//Ace low straight
		countRanks = []int{5, 4, 3, 2, 1}
	}
	straight := isStraight(countRanks)
	flush := isFlush(suits)

	switch {
	case straight && flush:
		handRank = 8
	case isEq(counts, []int{4, 1}):
		handRank = 7
	case isEq(counts, []int{3, 2}):
		handRank = 6
	case flush:
		handRank = 5
	case straight:
		handRank = 4
	case isEq(counts, []int{3, 1, 1}):
		handRank = 3
	case isEq(counts, []int{2, 2, 1}):
		handRank = 2
	case isEq(counts, []int{2, 1, 1, 1}):
		handRank = 1
	default:
		if !isEq(counts, []int{1, 1, 1, 1, 1}) {
			panic("Either have 5 of a kind, or missed a straight and/or a flush")
		}
		handRank = 0
	}

	return handRank, countRanks

}

func areHandsEq(a, b Hand) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
