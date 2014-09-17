package main

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
)

// gamePrinter prints the state of a Game to Std out. Used for testing.
func gamePrinter(g *Game) {
	f := os.Stdout

	//Print Basics
	fmt.Fprintf(f, "####GAME %v####\n", g.gameID)
	fmt.Fprintf(f, "round: %v\n", g.round)
	fmt.Fprintf(f, "small blind: %v\n", g.smallBlind)
	fmt.Fprintf(f, "deck : %s\n", g.deck)

	//Print players
	fmt.Fprintf(f, "--Players--\n")
	for _, p := range g.table {
		fmt.Fprintf(f, "  %v %v $%v\n", p.state, p.guid, p.wealth)
	}

	//Print the pot
	fmt.Fprintf(f, "--Pot--\n")
	fmt.Fprintf(f, "  min raise: %v\n", g.pot.minRaise)
	fmt.Fprintf(f, "  total to call: %v\n", g.pot.totalToCall)
	fmt.Fprintf(f, "  pot number: %v\n", g.pot.potNumber)
	for _, bet := range g.pot.bets {
		fmt.Fprintf(f, "  [%v | %v | %v]\n", bet.potNumber, bet.player, bet.value)
	}
	fmt.Fprint(f, "\n\n")
}

// createGuid returns a string representing a globally unique ID.
func createGuid() string {
	return s4() + s4() + "-" + s4() + "-" + s4() + "-" + s4() + "-" + s4() + s4() + s4()
}

// s4 is a helper function for createGuid. Should not be called directly.
func s4() string {
	s := ""
	for i := 0; i < 4; i++ {
		n := rand.Int63n(16)
		s += strconv.FormatInt(n, 16)
	}
	return s
}

// generateCardNames returns a slice of strings, each 2 characters in length representing
//  the 52 cards in a standard playing card deck
func generateCardNames() (deck [52]string) {
	suits := []string{"S", "C", "D", "H"}
	ranks := []string{"2", "3", "4", "5", "6", "7", "8", "9", "T", "J", "Q", "K", "A"}
	i := 0
	for _, suit := range suits {
		for _, rank := range ranks {
			deck[i] = rank + suit
			i++
		}
	}
	return deck
}
