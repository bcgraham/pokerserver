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
	PlayerBet   money  `json:"player bet so far"`
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
	GameID string
	Table  PublicTable
	Turn   *Turn
	Cards  *PublicCards
	Pots   *PublicPots
}

type authenticator map[guid]guid

func (gc *GameController) getGames() []*Game {
	gs := make([]*Game, 0)
	for _, g := range gc.Games {
		gs = append(gs, g)
	}
	return gs
}

func (gc *GameController) getGame(game guid) *PublicGame {
	return gc.Games[game].controller.public
}

func (gc *GameController) makeGame() *PublicGame {
	g := NewGame(gc)
	fmt.Println(g)
	gc.Games[g.gameID] = g
	go g.run()
	pg := new(PublicGame)
	pg = MakePublicGame(g)
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
	toGame     chan Act
	fromGame   chan int // ???
	queuedActs Acts
	public     *PublicGame
	waiting    []*Player
	sync.Mutex
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

type Acts struct {
	acts []Act
	sync.Mutex
}

type Act struct {
	Player    guid
	Action    int
	BetAmount money
}

func NewActs() Acts {
	as := new(Acts)
	as.acts = make([]Act, 0)
	return *as
}

func (c *controller) getPlayerBet(g *Game, wanted guid) (int, money, error) {
	ticker := time.NewTicker(500 * time.Millisecond)

	//---Testing---
	fmt.Printf("Asking **%s** for bet. Player has bet **%v**. Total to call is **%v**\n\n",
		wanted, g.pot.totalPlayerBetThisRound(wanted), g.pot.totalToCall)
	//-------------
	c.public = MakePublicGame(g)
	c.public.Turn.Player = wanted
	c.public.Turn.PlayerBet = g.pot.totalPlayerBetThisRound(wanted)
	fmt.Println(time.Now().Add(30 * time.Second).String())
	c.public.Turn.Expiry = time.Now().Add(30 * time.Second).String()
	timeout := time.After(30 * time.Second)
	for {
		select {
		case <-timeout:
			return 0, 0, fmt.Errorf("controller: timed out waiting for bet from player %v", wanted)
		case <-ticker.C:
			/*
				input := make([]byte, 1024)
				fmt.Println(">>place your bet...")
				n, err := os.Stdin.Read(input)
				if err != nil && err != io.EOF {
					panicMsg := "Error reading from standard in: " + fmt.Sprint(err)
					panic(panicMsg)
				}
				fmt.Println(input[:n-2])
				inputstr := string(input[:n-2])
				i, err := strconv.Atoi(inputstr)
				if err != nil {
					continue
				}
				if i < 0 {
					return fold, 0, nil
				}
				return call, money(i), nil */
			c.queuedActs.Lock()
			for i := 0; i < len(c.queuedActs.acts); i++ {
				if c.queuedActs.acts[i].Player == wanted {
					a := c.queuedActs.remove(i)
					c.queuedActs.Unlock()
					return a.Action, a.BetAmount, nil
				}
			}
			c.queuedActs.Unlock()
		}
	}

	panic("Unreachable")
}

func (c *controller) registerPlayerAct(a Act) error {
	c.queuedActs.Lock()
	defer c.queuedActs.Unlock()
	for i := 0; i < len(c.queuedActs.acts); i++ {
		if c.queuedActs.acts[i].Player == a.Player {
			return fmt.Errorf("controller: player %v already has an action in the queue", a.Player)
		}
	}
	c.queuedActs.acts = append(c.queuedActs.acts, a)
	return nil
}

func (as *Acts) remove(i int) (a Act) {
	a = as.acts[i]
	as.acts = append(as.acts[:i], as.acts[i+1:]...)
	return a
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
	c.fromGame = make(chan int)
	c.queuedActs = NewActs()
	c.waiting = make([]*Player, 0)
	return c
}

func (c *controller) registerInvalidBet(g *Game, player guid, bet money) { return }
func (c *controller) removePlayerFromGame(g *Game, player guid) {
	as := c.queuedActs
	replacementActs := Acts{}
	replacementActs.acts = make([]Act, 0)
	as.Lock()
	defer as.Unlock()
	for i := 0; i < len(as.acts); i++ {
		if as.acts[i].Player != player {
			replacementActs.acts = append(replacementActs.acts, as.acts[i])
		}
	}
	c.queuedActs = replacementActs
	replacementPlayers := make([]*Player, 0)
	for i := 0; i < len(g.table); i++ {
		if g.table[i].guid != player {
			replacementPlayers = append(replacementPlayers, g.table[i])
		}
	}
	g.table = replacementPlayers
	return
}
