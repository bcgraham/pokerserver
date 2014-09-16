package main

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
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
	fmt.Fprint(f, "\n\n")
}

func createGuid() string {
	return s4() + s4() + "-" + s4() + "-" + s4() + "-" + s4() + "-" + s4() + s4() + s4()
}

func s4() string {
	s := ""
	for i := 0; i < 4; i++ {
		n := rand.Int63n(16)
		s += strconv.FormatInt(n, 16)
	}
	return s
}

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

func (t *Table) contains(id guid) bool {
	for _, player := range t.players {
		if player.guid == id {
			return true
		}
	}
	return false
}

func (t *Table) remove(id guid) {
	var index int
	for i, player := range t.players {
		if player.guid == id {
			index = i
		}
	}
	if index == len(t.players)-1 {
		t.players = t.players[:index]
	} else {
		t.players = append(t.players[:index], t.players[index+1:]...)
	}
}
