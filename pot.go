package main

import (
	"fmt"
	"math"
)

type Pot struct {
	minRaise    money
	totalToCall money
	potNumber   uint
	bets        []Bet
}
type Bet struct {
	potNumber uint
	player    guid
	value     money
}

//newPot is a constructor for a new Pot struct
func newPot() *Pot {
	pot := new(Pot)
	pot.bets = make([]Bet, 0)
	return pot
}

//Resolves partial pots from previous round
// increments potNumber to a previously unused number
func (pot *Pot) newRound() {
	pot.condenseBets()
	pot.makeSidePots()
	pot.minRaise = 0
	pot.totalToCall = 0
	pot.potNumber++
}

func (pot *Pot) condenseBets() {
	playerBets := make(map[guid]money)
	betsCopy := make([]Bet, 0)
	for _, bet := range pot.bets {
		if bet.potNumber == pot.potNumber {
			playerBets[bet.player] += bet.value
		} else {
			betsCopy = append(betsCopy, bet)
		}
	}
	for k, v := range playerBets {
		betsCopy = append(betsCopy, Bet{potNumber: pot.potNumber, player: k, value: v})
	}

	pot.bets = betsCopy
}

func (pot *Pot) allBetsEqual(potNumber uint) bool {
	var prevBet money
	for _, bet := range pot.bets {
		if bet.potNumber == potNumber {
			prevBet = bet.value
			break
		}
	}
	for _, bet := range pot.bets {
		if bet.potNumber != potNumber {
			continue
		}
		if prevBet != bet.value {
			return false
		}
	}
	return true
}

func (pot *Pot) makeSidePots() {
	if pot.allBetsEqual(pot.potNumber) {
		return
	}
	pot.potNumber++

	minimum := money(math.MaxUint64)
	for _, b := range pot.bets {
		if b.value < minimum && b.potNumber == (pot.potNumber-1) {
			minimum = b.value
		}
	}
	for i := 0; i < len(pot.bets); i++ {
		b := pot.bets[i]
		if b.value > minimum && b.potNumber == (pot.potNumber-1) {
			excess := b.value - minimum
			b.value = minimum
			pot.bets = append(pot.bets, Bet{potNumber: pot.potNumber, player: b.player, value: excess})
		}
		pot.bets[i] = b
	}
	pot.makeSidePots()
}

func (p *Pot) receiveBet(id guid, bet money) {
	betSoFar := p.totalPlayerBetThisRound(id)
	raise := p.raiseAmount(id, bet)
	if betSoFar+bet > p.totalToCall {
		p.totalToCall = betSoFar + bet
	}
	fmt.Printf("Receiving bet: player %v has bet amount %v, bringing her total bet to %v (raise amount %v)\n", id, bet, betSoFar, raise)
	if raise > p.minRaise {
		p.minRaise = raise
	}
	newBet := Bet{potNumber: p.potNumber, player: id, value: bet}
	p.bets = append(p.bets, newBet)
}

func (p *Pot) totalInPot() money {
	var sum money = 0
	for _, m := range p.bets {
		sum += m.value
	}
	return sum
}

func (p *Pot) totalPlayerBetThisRound(id guid) money {
	sum := money(0)
	for _, bet := range p.bets {
		if bet.player == id && bet.potNumber == p.potNumber {
			sum += bet.value
		}
	}
	return sum
}

// stakeholders returns a map of sidepots to players who have bet in each sidepot.
func (p *Pot) stakeholders() map[uint][]guid {
	stakeholders := make(map[uint][]guid)
	for _, bet := range p.bets {
		if _, ok := stakeholders[bet.potNumber]; !ok {
			stakeholders[bet.potNumber] = make([]guid, 0)
		}
		stakeholders[bet.potNumber] = append(stakeholders[bet.potNumber], bet.player)
	}
	return stakeholders
}

// amounts returns a map of sidepots to amounts.
func (p *Pot) amounts() map[uint]money {
	amounts := make(map[uint]money)
	for _, bet := range p.bets {
		amounts[bet.potNumber] += bet.value
	}
	return amounts
}

// commitBet decrements player wealth by bet and adds bet to the current pot.
func (p *Pot) commitBet(player *Player, bet money) {
	if bet < 0 {
		panic("trying to bet < 0")
	}
	p.receiveBet(player.guid, bet)
	player.wealth -= bet
}

// betInvalid returns true if the bet is a valid bet, and false if the bet is not valid.
func (p *Pot) betInvalid(player *Player, bet money) bool {
	raise := p.raiseAmount(player.guid, bet)
	return (bet > player.wealth) ||
		(raise > 0 && raise < p.minRaise) ||
		(bet < player.wealth && (p.totalPlayerBetThisRound(player.guid)+bet) < p.totalToCall)
}

// raiseAmount returns the amount the current bet is raising (possibly 0).
func (p *Pot) raiseAmount(id guid, betAmount money) money {
	return p.totalPlayerBetThisRound(id) + betAmount - p.totalToCall
}
