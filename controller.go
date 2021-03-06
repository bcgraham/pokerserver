package main

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

type GameController struct {
	Games map[guid]*Game
	auth  authenticator
}

type PublicPlayer struct {
	GUID       guid   `json:"playerID"`
	Handle     string `json:"handle"`
	State      string `json:"state"`
	Wealth     money  `json:"wealth"`
	InFor      money  `json:"bet_so_far"`
	SmallBlind bool   `json:"small_blind"`
}

type PublicTable []*PublicPlayer

type Turn struct {
	Player      guid   `json:"playerID"`
	PlayerBet   money  `json:"bet_so_far"`
	BetToPlayer money  `json:"bet_to_player"`
	MinRaise    money  `json:"minimum_raise"`
	Expiry      string `json:"expiry"`
}

type PublicCards struct {
	Hole  []string `json:"hole"`
	Flop  []string `json:"flop"`
	Turn  []string `json:"turn"`
	River []string `json:"river"`
}

type PublicPots []*PublicPot

type PublicPot struct {
	Size    money  `json:"size"`
	Players []guid `json:"players"`
}

type PublicGame struct {
	GameID          string       `json:"gameID"`
	Table           PublicTable  `json:"table"`
	Turn            *Turn        `json:"turn"`
	Cards           *PublicCards `json:"cards"`
	Pots            *PublicPots  `json:"pots"`
	LastHandWinners []Playerhand `json:"last_winners"`
}

type authenticator map[guid]guid

const TIMEOUT = 100

func (gc *GameController) getGames() []*Game {
	gs := make([]*Game, 0)
	for _, g := range gc.Games {
		gs = append(gs, g)
	}
	return gs
}

func (gc *GameController) getGame(game guid) PublicGame {
	return *gc.Games[game].controller.public
}

func (gc *GameController) makeGame() *PublicGame {
	g := NewGame(gc)
	gc.Games[g.gameID] = g
	go g.run()
	pg := MakePublicGame(g)
	return pg
}

func MakeTurn() *Turn {
	turn := new(Turn)
	turn.Player = "" // ??
	turn.Expiry = "" // ??
	return turn
}

func MakePublicGame(g *Game) *PublicGame {
	pg := new(PublicGame)
	pg.GameID = string(g.gameID)
	pg.Table = make(PublicTable, 0)
	for _, player := range g.table {
		pg.Table = append(pg.Table, MakePublicPlayer(g, player))
	}
	pg.Turn = MakeTurn()
	pg.Turn.BetToPlayer = g.pot.totalToCall
	pg.Turn.MinRaise = g.pot.minRaise
	pg.Turn.Player = "" // ??
	pg.Turn.Expiry = "" // ??
	pg.Cards = MakePublicCards(g)
	pg.Pots = MakePublicPots(g)
	return pg
}

func (gc *GameController) getGamePrivate(game, player guid) (pg PublicGame) {
	g := gc.Games[game]
	pg = gc.getGame(game)
	pg.Cards = MakePublicCards(g)
	pg.Cards.Hole = g.deck.Get(string(player))
	return pg
}

func MakePublicCards(g *Game) (pc *PublicCards) {
	pc = new(PublicCards)
	switch g.round {
	case 3: // river
		pc.River = g.deck.Get("RIVER")
		fallthrough
	case 2: // turn
		pc.Turn = g.deck.Get("TURN")
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
	pp.InFor = g.pot.totalPlayerBetThisRound(p.guid)
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
	if g.table[0] == p {
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
	toGame  chan Act
	public  *PublicGame
	waiting []*Player
	sync.Mutex
}

type Playerhand struct {
	PlayerID guid
	Hand     Hand
}

func (c *controller) recordWinners(winners []*Player) {
	playerhands := make([]Playerhand, 0)
	for _, player := range winners {
		playerhands = append(playerhands, Playerhand{PlayerID: player.guid, Hand: player.bestHand})
	}
	c.public.LastHandWinners = playerhands
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
	for _, player := range g.table {
		if player.guid == p.guid {
			return fmt.Errorf("controller: player %v is already sitting at the table", p.guid)
		}
	}
	c.waiting = append(c.waiting, p)
	return nil
}

type Act struct {
	Player    guid
	Action    int
	BetAmount money
}

func (c *controller) getPlayerBet(g *Game, wanted guid) (int, money, error) {
	pg := MakePublicGame(g)
	pg.LastHandWinners = c.public.LastHandWinners
	c.public = pg
	c.public.Turn.Player = wanted
	c.public.Turn.PlayerBet = g.pot.totalPlayerBetThisRound(wanted)
	c.public.Turn.Expiry = time.Now().Add(TIMEOUT * time.Second).String()
	timeout := time.After(TIMEOUT * time.Second)
	for {
		select {
		case <-timeout:
			return 0, 0, fmt.Errorf("controller: timed out waiting for bet from player %v", wanted)
		case a := <-c.toGame:
			return a.Action, a.BetAmount, nil
		}
	}

	panic("Unreachable")
}

func (c *controller) registerPlayerAct(a Act) error {
	if a.Player != c.public.Turn.Player {
		return fmt.Errorf("controller: not this player's turn: %v", a.Player)
	}
	c.toGame <- a
	return nil
}

func NewPlayer(id guid) (p *Player) {
	p = new(Player)
	p.guid = id
	p.wealth = BUY_IN
	return p
}

func NewController(g *Game) *controller {
	c := new(controller)
	c.toGame = make(chan Act)
	c.public = MakePublicGame(g)
	c.waiting = make([]*Player, 0)
	return c
}

func (c *controller) registerInvalidBet(g *Game, player guid, bet money) { return }
func (c *controller) removePlayerFromGame(g *Game, player guid) {
	replacementPlayers := make([]*Player, 0)
	for i := 0; i < len(g.table); i++ {
		if g.table[i].guid != player {
			replacementPlayers = append(replacementPlayers, g.table[i])
		}
	}
	g.table = replacementPlayers
	return
}
