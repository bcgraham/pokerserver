package main

import (
	"fmt"
	"math/rand"
)

type Table struct {
	players []*Player
	index   int
}

func (t *Table) addPlayer(p guid) (err error) {
	var newPlayer *Player = &Player{state: active, guid: p, wealth: 1000}
	if len(t.players) >= 10 {
		err = fmt.Errorf("Table full!")
		return err
	}

	t.players = append(t.players, newPlayer)
	return err

}

func (t *Table) AdvanceButton() {
	n := len(t.players)
	last := t.players[0]
	for i := 0; i < (len(t.players) - 1); i++ {
		t.players[i] = t.players[i+1]
	}
	t.players[n-1] = last
}

func (t *Table) ResetRoundPlayerState() {
	for _, p := range t.players {
		if p.state == called {
			p.state = active
		}
	}
}

func (t *Table) ResetHandPlayerState() {
	for _, p := range t.players {
		p.state = active
	}
}

func (t *Table) ResetRound() {
	t.index = 0
	t.ResetRoundPlayerState()
}

func (t *Table) ResetHand() {
	t.index = 0
	t.ResetHandPlayerState()
}

func (t *Table) Next() (p *Player) {
	p = t.players[t.index]
	t.index = (t.index + 1) % len(t.players)
	return p
}

// TODO: need map[guid]player at table level to avoid
// messy n^2 lookup time
func (t *Table) getPlayers(guids []guid) []*Player {
	players := make([]*Player, 0)
	for _, p := range t.players {
		for _, g := range guids {
			if p.guid == g {
				players = append(players, p)
			}
		}
	}
	if len(players) != len(guids) {
		panic("table.getPlayers is returning a list of players of different length than the length of its input guids")
	}
	return players
}

// assignBestHands assigns to each player
// her best hand from the current deal.
func (t *Table) assignBestHands(deck Deck) {
	for _, p := range t.players {
		allHands := generateAllHands(deck, p)
		p.bestHand = bestHand(allHands)
	}
}

// bestHand returns the best Hand from a slice of Hands.
func bestHand(hands []Hand) Hand {
	return findWinningHands(allHands)[0]
}

func NewGame(gc *GameController) (g *Game) {
	g = new(Game)
	g.table = new(Table)
	g.table.players = make([]*Player, 0)
	g.pot = new(Pot)
	g.pot.bets = make([]Bet, 0)
	g.controller = NewController()
	g.smallBlind = 10
	g.random = rand.New(rand.NewSource(SEED))
	return g
}
