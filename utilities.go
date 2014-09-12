package main

import (
	"fmt"
	"os"
)

func gamePrinter(g *Game) {
	// f, err := os.OpenFile("test.log", os.O_APPEND|os.O_WRONLY, 0777)
	f := os.Stdout
	// defer f.Close()
	// if err != nil {
	// 	panic(err)
	// }

	//Print Basics
	fmt.Fprintf(f, "####GAME %v####\n", g.gameID)
	fmt.Fprintf(f, "round: %v\n", g.round)
	fmt.Fprintf(f, "small blind: %v\n", g.smallBlind)
	fmt.Fprintf(f, "table index: %v\n", g.table.index)
	fmt.Fprintf(f, "deck : %s\n", g.deck)

	//Print players
	fmt.Fprintf(f, "--Players--\n")
	for _, p := range g.table.players {
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
	fmt.Fprint(f, "\n\n\n")
}
