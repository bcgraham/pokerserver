package main

import "math/rand"

const SEED int64 = 0 // seed for deal
var UNSHUFFLED = generateCardNames()

const (
	fold int = iota
	bet
	call
)
const (
	active state = iota
	folded
	called
)
const BUY_IN money = 500

type state int
type money uint64
type guid string
type Player struct {
	state    state
	guid     guid
	wealth   money
	bestHand Hand
}
type Game struct {
	table      Table
	pot        *Pot
	gameID     guid
	deck       Deck
	round      uint
	smallBlind money
	controller *controller
	random     *rand.Rand
}

//run executes the game of poker forever while there are at least 2 players
func (g *Game) run() {
	for {
		g.removeBrokePlayers()
		g.addWaitingPlayers()
		if len(g.table) < 2 {
			continue //Need 2 players to start a hand
		}
		gamePrinter(g)
		g.table.AdvanceButton()
		g.pot = newPot()
		g.betBlinds()
		g.deal()
		for g.round = 0; !g.allFolded() && g.round < 4; g.round++ {
			g.placeBets()
			g.table.makeCalledPlayersActive()
			g.pot.newRound()
		}
		g.resolveBets()
		g.table.makeAllPlayersActive()
	}
}

// removeBrokePlayers folds and removes any players with wealth == 0
func (g *Game) removeBrokePlayers() {
	for _, p := range g.table {
		if p.wealth == 0 {
			p.state = folded
			g.controller.removePlayerFromGame(g, p.guid)
		} else if p.wealth < 0 {
			panic("player has < 0 wealth!")
		}
	}
}

// addWaitingPlayers asks controller for waiting players
//  and adds them to the table
func (g *Game) addWaitingPlayers() {
	numPlayersNeeded := (10 - len(g.table))
	newPlayers := g.controller.getNewPlayers(g, numPlayersNeeded)
	for _, p := range newPlayers {
		err := g.table.addPlayer(p.guid)
		if err != nil {
			panic(err)
		}
	}
}

// betBlinds places the small blind and big blind bet for the first two players.
//  If those players do not have enough money to meet blinds, will put player all in
func (g *Game) betBlinds() {
	//Bet small blind
	player := g.table[0]
	if player.wealth >= g.smallBlind {
		g.pot.commitBet(player, g.smallBlind)
	} else {
		g.pot.commitBet(player, player.wealth)
	}

	//Bet big blind
	player = g.table[1]
	if player.wealth >= 2*g.smallBlind {
		g.pot.commitBet(player, 2*g.smallBlind)
	} else {
		g.pot.commitBet(player, player.wealth)
	}
}

//deal assigns 2 unique cards to each player and 5 unique cards to the table
func (g *Game) deal() {
	g.deck = make(Deck, 52)
	numPlayers := len(g.table)
	rand_ints := g.random.Perm(52)
	for i := 0; i < numPlayers; i++ {
		card1, card2 := UNSHUFFLED[rand_ints[i*2]], UNSHUFFLED[rand_ints[i*2+1]]
		g.deck[card1] = string(g.table[i].guid)
		g.deck[card2] = string(g.table[i].guid)
	}
	n := numPlayers * 2
	g.deck[UNSHUFFLED[rand_ints[n+0]]] = "FLOP"
	g.deck[UNSHUFFLED[rand_ints[n+1]]] = "FLOP"
	g.deck[UNSHUFFLED[rand_ints[n+2]]] = "FLOP"
	g.deck[UNSHUFFLED[rand_ints[n+3]]] = "TURN"
	g.deck[UNSHUFFLED[rand_ints[n+4]]] = "RIVER"

	g.table.assignBestHands(g.deck)
}

//allFolded returns true if all players have folded.
func (g *Game) allFolded() bool {
	numFolded := 0
	for _, p := range g.table {
		if p.state == folded {
			numFolded++
		}
	}
	return numFolded == len(g.table)
}

//placeBets gets bet from controller, checks bet validity, and places bet
func (g *Game) placeBets() {
	for i := 0; g.betsNeeded(); i = (i + 1) % len(g.table) {
		player := g.table[i]

		if player.state != active {
			continue
		}
		action, betAmount, err := g.controller.getPlayerBet(g, player.guid)

		//Illegit bets
		if err != nil {
			//Err occurs on connection timeout
			player.state = folded
			g.controller.removePlayerFromGame(g, player.guid)
			continue
		}
		if action == fold {
			player.state = folded
			continue
		}
		if g.pot.betInvalid(player, betAmount) {
			g.controller.registerInvalidBet(g, player.guid, betAmount)
			player.state = folded
			continue
		}

		//Legit bets
		if g.pot.raiseAmount(player.guid, betAmount) > 0 {
			g.table.makeCalledPlayersActive()
		}
		g.pot.commitBet(player, betAmount)
		player.state = called
	}
}

// betsNeeded returns true if there are players that still need to bet
func (g *Game) betsNeeded() bool {
	numActives := 0
	numFolded := 0
	for _, p := range g.table {
		if p.state == active {
			numActives++
		} else if p.state == folded {
			numFolded++
		}
	}
	return (numActives >= 1) && ((len(g.table) - numFolded) > 1)
}

// resolveBets loops through all sidepots. For each sidepot,
// among the stakeholders, the pot is distributed to the winner(s).
func (g *Game) resolveBets() {
	moneyInPots := g.pot.amounts()

	for potNumber, guids := range g.pot.stakeholders() {
		sidepot := moneyInPots[potNumber]
		players := g.table.getPlayers(guids)
		winners := findWinners(players)
		numWinners := money(len(winners))
		for _, p := range winners {
			p.wealth += sidepot / numWinners
			if sidepot%numWinners > 0 {
				p.wealth++
				moneyInPots[potNumber]--
			}
		}
	}
}
