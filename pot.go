package main

import "math"

//===============POT AND BET============
//======================================
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

//Resolves partial pots from previous round
// increments potNumber to a previously unused number
func (pot *Pot) newRound() {
	pot.condenseBets()
	pot.makeSidePots()
	pot.minRaise = 0
	pot.totalToCall = 0
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

func (pot *Pot) receiveBet(guid guid, bet money) {
	newBet := Bet{potNumber: pot.potNumber, player: guid, value: bet}
	pot.bets = append(pot.bets, newBet)
	totalBet := pot.totalPlayerBetThisRound(guid)
	if totalBet > pot.totalToCall {
		pot.totalToCall = totalBet
	}
	raise := totalBet - pot.totalToCall
	if raise > pot.minRaise {
		pot.minRaise = 2 * raise
	}
}

func (pot *Pot) totalInPot() money {
	var sum money = 0
	for _, m := range pot.bets {
		sum += m.value
	}
	return sum
}

func (pot *Pot) totalPlayerBetThisRound(p guid) money {
	sum := money(0)
	for _, bet := range pot.bets {
		if bet.player == p && bet.potNumber == pot.potNumber {
			sum += bet.value
		}
	}
	return sum
}

// stakeholders returns a map of sidepots to players who have bet in each sidepot.
func (pot *Pot) stakeholders() map[uint][]guid {
	stakeholders := make(map[uint][]guid)
	for _, bet := range pot.bets {
		if _, ok := stakeholders[bet.potNumber]; !ok {
			stakeholders[bet.potNumber] = make([]guid, 0)
		}
		stakeholders[bet.potNumber] = append(stakeholders[bet.potNumber], bet.player)
	}
	return stakeholders
}

// amounts returns a map of sidepots to amounts.
func (pot *Pot) amounts() map[uint]money {
	amounts := make(map[uint]money)
	for _, bet := range pot.bets {
		amounts[bet.potNumber] += bet.value
	}
	return amounts
}
