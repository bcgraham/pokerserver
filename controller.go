package controller

import (
	"sort"
	"time"
)

type GameController struct {
	Games map[guid]*Game
	auth  authenticator
}

type PublicPlayer struct {
	GUID       guid   `json:"guid"`
	Handle     string `json:"handle"`
	State      string `json:"state"`
	Wealth     money  `json:"wealth"`
	InFor      money  `json:"in for"`
	SmallBlind bool   `json:"small blind"`
}

type PublicTable []*PublicPlayer

type Turn struct {
	Player      guid      `json:"player"`
	BetToPlayer money     `json:"bet to the player"`
	MinRaise    money     `json:"minimum raise"`
	Expiry      time.Time `json:"expiry"`
}

type PublicCards struct {
	Hole  []string
	Flop  []string
	Turn  string
	River string
}

type PublicPots []*PublicPot

type PublicPot struct {
	Size    money
	Players []guid
}

type PublicGame struct {
	Table PublicTable
	Turn  Turn
	Cards PublicCards
	Pots  PublicPots
}

type authenticator map[guid]*Player

func (gc *GameController) getGames() {
	if player == "" {
		// send public-only data
	}

}

func (gc *GameController) getGame(game guid) (pg *PublicGame) {
	g := gc.Games[game]
	pg = new(PublicGame)
	pg.Table = make(PublicTable, 0)
	for _, player := range g.Table {
		pg.Table = append(pg.Table, MakePublicPlayer(g, player))
	}
	pg.Turn = new(Turn)
	pg.Turn.BetToPlayer = g.Pot.totalToCall
	pg.Turn.MinRaise = g.Pot.minRaise
	pg.Turn.Player = "" // ??
	pg.Turn.Expiry = "" // ??
	pg.PublicCards = MakePublicCards(g)
	pg.PublicPots = MakePublicPots(g)
	return pg
}

func (gc *GameController) getGamePrivate(game, player guid) (pg *PublicGame) {
	pg = getGame(game)
	pg.Cards.Hole = gc.Games[game].deck.Get(player)
	return pg
}

func MakePublicCards(g *Game) (pc *PublicCards) {
	pc = new(PublicCards)
	switch g.round {
	case 3: // river
		pc.River = g.Deck.Get("RIVER")[0]
		fallthrough
	case 2: // turn
		pc.Turn = g.Deck.Get("TURN")[0]
		fallthrough
	case 1: // flop
		pc.Flop = g.Deck.Get("FLOP")
		fallthrough
	default: // pre-flop
	}
	return pc
}

func (deck Deck) Get(wanted string) (cards []string) {
	cards = make([]string, 0)
	for card, location := range deck {
		if location == wanted {
			cards = append(locations, card)
		}
	}
	sort.Strings(cards)
	return cards
}

func MakePublicPlayer(g *Game, p *Player) (pp *PublicPlayer) {
	pp = new(PublicPlayer)
	pp.Wealth = p.wealth
	pp.GUID = p.guid
	switch p.state {
	case 0:
		pp.State = "active"
	case 1:
		pp.State = "folded"
	case 2:
		pp.State = "called"
	default:
		panic("unknown state")
	}
	if g.Table[0] == p {
		pp.SmallBlind = true
	}
}

func MakePublicPots(g *Game) (pp *PublicPots) {
	pp = new(PublicPots)
	for _, bet := range g.Pot.bets {
		for bet.potNumber >= len(pp) {
			pp = append(pp, new(PublicPot))
		}
		pot = pp[bet.potNumber]
		if !contains(pot.players, bet.player) {
			pot.players = append(pot.players, bet.player)
		}
		pot.Size += bet.value
		pp[bet.potNumber] = pot
	}
	return pp
}

func contains(haystack []guid, needle guid) bool {
	for _, g := range haystack {
		if g == needle {
			return true
		}
	}
	return false
}

func NewGameController() (gc *GameController) {
	gc = new(GameController)
	gc.Games = make([]*Game, 0)
	return gc
}
