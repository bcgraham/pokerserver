package main

import (
	"fmt"
	"sort"
	"sync"
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
	Player      guid   `json:"player"`
	BetToPlayer money  `json:"bet to the player"`
	MinRaise    money  `json:"minimum raise"`
	Expiry      string `json:"expiry"`
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
	Turn  *Turn
	Cards *PublicCards
	Pots  *PublicPots
}

type authenticator map[guid]guid

func (gc *GameController) getGames() []*Game {
	gs := make([]*Game, 0)
	for _, g := range gc.Games {
		gs = append(gs, g)
	}
	return gs
}

func (gc *GameController) getGame(game guid) (pg *PublicGame) {
	g := gc.Games[game]
	pg = new(PublicGame)
	pg.Table = make(PublicTable, 0)
	for _, player := range g.table.players {
		pg.Table = append(pg.Table, MakePublicPlayer(g, player))
	}
	pg.Turn = new(Turn)
	pg.Turn.BetToPlayer = g.pot.totalToCall
	pg.Turn.MinRaise = g.pot.minRaise
	pg.Turn.Player = "" // ??
	pg.Turn.Expiry = "" // ??
	pg.Cards = MakePublicCards(g)
	pg.Pots = MakePublicPots(g)
	return pg
}

func (gc *GameController) getGamePrivate(game, player guid) (pg *PublicGame) {
	pg = gc.getGame(game)
	pg.Cards.Hole = gc.Games[game].deck.Get(string(player))
	return pg
}

func MakePublicCards(g *Game) (pc *PublicCards) {
	pc = new(PublicCards)
	switch g.pot.potNumber {
	case 3: // river
		pc.River = g.deck.Get("RIVER")[0]
		fallthrough
	case 2: // turn
		pc.Turn = g.deck.Get("TURN")[0]
		fallthrough
	case 1: // flop
		pc.Flop = g.deck.Get("FLOP")
		fallthrough
	default: // pre-flop
	}
	return pc
}

func (deck Deck) Get(wanted string) (cards []string) {
	cards = make([]string, 0)
	for card, location := range deck {
		if location == wanted {
			cards = append(cards, card)
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
	if g.table.players[0] == p {
		pp.SmallBlind = true
	}
	return pp
}

func MakePublicPots(g *Game) (pp *PublicPots) {
	pp = new(PublicPots)
	for _, bet := range g.pot.bets {
		for int(bet.potNumber) >= len(*pp) {
			*pp = append(*pp, new(PublicPot))
		}
		pot := (*pp)[bet.potNumber]
		if !contains(pot.Players, bet.player) {
			pot.Players = append(pot.Players, bet.player)
		}
		pot.Size += bet.value
		(*pp)[bet.potNumber] = pot
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
	gc.Games = make(map[guid]*Game)
	return gc
}

type controller struct {
	toGame   chan Act
	fromGame chan int // ???
	buffer   Acts
	waiting  []*Player
	sync.RWMutex
}

func (c *controller) getNewPlayers(g *Game, openSeats int) (players []*Player) {
	c.Lock()
	defer c.Unlock()
	x := openSeats
	if len(c.waiting) == 0 {
		return players
	}
	if len(c.waiting) < openSeats {
		x = len(c.waiting)
	}
	players = c.waiting[:x]
	c.waiting = c.waiting[x:len(c.waiting)]
	return players
}

func (c *controller) enqueuePlayer(g *Game, p *Player) error {
	c.Lock()
	defer c.Unlock()
	for _, player := range c.waiting {
		if player.guid == p.guid {
			return fmt.Errorf("controller: player %v is already queued to join table")
		}
	}
	for _, player := range g.table.players {
		if player.guid == p.guid {
			return fmt.Errorf("controller: player %v is already sitting at the table", p.guid)
		}
	}
	c.waiting = append(c.waiting, p)
	return nil
}

type Acts []Act

type Act struct {
	player    guid
	action    int
	betAmount money
}

func (as *Acts) getPlayerAct(wanted guid) (a Act, err error) {
	for i := 0; i < len(*as); i++ {
		if (*as)[i].player == wanted {
			return as.remove(i), nil
		}
	}
	return Act{}, fmt.Errorf("controller: player %v does not have an action in the queue", wanted)
}

func (as *Acts) remove(i int) (a Act) {
	a = (*as)[i]
	*as = append((*as)[:i], (*as)[i+1:]...)
	*as = (*as)[:len(*as)-1]
	return a
}

func NewPlayer(id guid) (p *Player) {
	p = new(Player)
	p.guid = id
	p.wealth = BUY_IN
	return p
}

func (c *controller) registerInvalidBet(g *Game, player guid, bet money)    { return }
func (c *controller) removePlayerFromGame(g *Game, player guid)             { return }
func (c *controller) getPlayerBet(g *Game, player guid) (int, money, error) { return 2, money(2), nil }
