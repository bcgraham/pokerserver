package main

import (
	"fmt"
	"math/rand"
)

type Table []*Player

func (t Table) addPlayer(p guid) (err error) {
	var newPlayer *Player = &Player{state: active, guid: p, wealth: 1000}
	if len(t) >= 10 {
		err = fmt.Errorf("Table full!")
		return err
	}

	t = append(t, newPlayer)
	return err

}

func (t Table) AdvanceButton() {
	tmp := t[len(t)-1]
	for i := len(t) - 1; i > 0; i-- {
		t[i] = t[i-1]
	}
	t[0] = tmp
}

func (t Table) makeCalledPlayersActive() {
	for _, p := range t {
		if p.state == called {
			p.state = active
		}
	}
}

func (t Table) makeAllPlayersActive() {
	for _, p := range t {
		p.state = active
	}
}

// TODO: need map[guid]player at table level to avoid
// messy n^2 lookup time
func (t Table) getPlayers(guids []guid) []*Player {
	players := make([]*Player, 0)
	for _, p := range t {
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
func (t Table) assignBestHands(deck Deck) {
	for _, p := range t {
		allHands := generateAllHands(deck, p.guid)
		p.bestHand = bestHand(allHands)
	}
}

// bestHand returns the best Hand from a slice of Hands.
func bestHand(hands []Hand) Hand {
	return findWinningHands(hands)[0]
}

func NewGame(gc *GameController) (g *Game) {
	g = new(Game)
	g.gameID = guid(createGuid())
	fmt.Println(g.gameID)
	g.table = make(Table, 0)
	g.pot = new(Pot)
	g.pot.bets = make([]Bet, 0)
	g.controller = NewController()
	g.smallBlind = 10
	g.random = rand.New(rand.NewSource(SEED))
	return g
}

func (t Table) contains(id guid) bool {
	for _, player := range t {
		if player.guid == id {
			return true
		}
	}
	return false
}

func (t Table) remove(id guid) {
	var index int
	for i, player := range t {
		if player.guid == id {
			index = i
		}
	}
	if index == len(t)-1 {
		t = t[:index]
	} else {
		t = append(t[:index], t[index+1:]...)
	}
}
