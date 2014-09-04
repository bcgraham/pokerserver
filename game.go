//Utility Functions
package main 
import (
    "math/rand"
    "strconv"
    "fmt"
)

//==========================================================================
//===============TYPES AND CONSTS===========================================
//==========================================================================
var SEED = 0 // seed for deal 
type roundName int 
type state int 
type money uint64
type Deck map[string]string 
type Pot []Bets
type guid string
const (
    deck int = iota 
    _ 
    flop 
    turn 
    river 
)
const (
    active state = iota 
    folded 
    called
)
type Player struct {
    state state
    guid guid
    wealth money
}
type Bet struct {
    potNumber int 
    player guid
    value uint
}

func (g *Game) Deal(seed int64) {
    unshuffled := newDeck()
    numPlayers := len(g.Players)
    r := rand.New(rand.NewSource)
    shuffled := rand.Perm(52)
    for i := 0; i < numPlayers; i = i + 2 {
        card1, card2 = unshuffled[i], unshuffled[i+1]
        g.deck[card1] = g.Players[i].guid
        g.deck[card2] = g.Players[i].guid
    }
    n := numPlayers*2
    g.deck[unshuffled[n+0]] = "FLOP"
    g.deck[unshuffled[n+1]] = "FLOP"
    g.deck[unshuffled[n+2]] = "FLOP"
    g.deck[unshuffled[n+3]] = "TURN"
    g.deck[unshuffled[n+4]] = "RIVER"
}


//==========================================================================
//===============GAME CLASS=================================================
//==========================================================================
type Game struct {
    table Table 
    pot Pot
    gameID guid
    handID guid
    deck Deck
    round int  
    smallBlind money
    controller *Controller 
    minBet money
}

func (g *Game) run() {

    // LOOP: // this loop represents one complete hand 
        // ->Bring in new players
        // -> is players array > 0 ?
        // -> make sure everyone playing has money to play (enought o make blinds, etc…). if not end game
        // -> kick out players w/o enough money
                    // *somehow revoke their token and let them know
        // -> Check for new players that want into the game
        // -> Queue up new players to join game, notify player they are in queue to join
            // *queued players should get game state with player list that d/n include them
            // before: no bets placed
        
        g.betBlinds()
        g.Deal(SEED)  
        for i := 0 ; g.notSettled() && i < 4; i++{
            g.round = i
            g.Bets()
            g.Table.Reset()
        }
        g.resolveBets()
        g.AdvanceButton()

       



        // -> send the game state to the controller(server) with a list of players to send info to
        //     --->(controllre will then filter and format the data to send to each player so they
        //      get just the info they need in JSON format)
        // ENDLOOP


    // -> add queued players to game, if they are still responding
    //     *Give them some default amount of money
    //     *give them a token to send back requests
    // ENDLOOP

}

//setBlinds sets the money amount for the blinds
// and rotates the "button"
func (g *Game) setBlinds() {
    g.smallBlind = 25
}

func (g *Game) betsNeeded() bool {
    numActives := 0
    numCalled := 0
    for _, p := range g.Players {
        if p.state == active{
            numActives++
        }
        else if p.state == called {
            numCalled++
        }
    }
    return (numActives >= 1) && (numCalled >= 1)
}

//Bets gets the bet from each player
func (g *Game) Bets() bool {
    table = g.table 
    g.topBet := 0 
    g.minBet := g.smallBlind

    for player := table.Next(); g.betsNeeded(); player = table.Next() {
        controller.sendGameState(g)

        if player.state != active {
            continue
        }

        action, value, err := controller.getPlayerBet(g, player.guid)

        //Illegit bets
        if err != nil {
            //Err occurs on connection timeout
            player.state = folded
            controller.removePlayerFromGame(g, player.guid)
            continue
        }
        if action == folded {
            player.state = folded 
            continue 
        }
        if g.betInvalid(player, bet) {
            controller.registerInvalidBet(g, player.guid, bet)
            player.state = folded
            continue
        }

        //Legit bets
        g.CommitBet(player, bet)
        minBet = bet
        if bet > topBet { // raising 
            topBet = bet
            g.ResetPlayerState()
        }
        player.state = called
    }

}

func (g *Game) betInvalid(p player, bet money) {
    return (b > p.wealth) || (b < g.minBet) || (bet < player.wealth && bet < g.topBet)
}

func (g *Game) CommitBet(p player, bet money) {
    g.Pot.receiveBet(player.guid, bet)
    player.wealth -= bet
}

//==========================================================================
//===============TABLE======================================================
//==========================================================================
type Table struct {
    players [10]*Player
    index int 
}

func (t *Table) AdvanceButton() {
    temp := t.players[0]
    t.players[0:] = t.players[1:]
    t.players[9] = temp 
    if t.players[0] == nil {
        t.AdvanceButton()
    }
}

func (t *Table) ResetPlayerState() {
    for _, p := range t {
        if p.state == called {
            p.state = active
        }
    }
}

func (t *Table) Reset() {
    t.index = 0
    t.ResetPlayerState()
}

func (t *Table) Next() (p player) {
    for {
        p = t.players[t.index]
        t.index = (t.index + 1) % 10
        if p != nil {
            return p
        }
    }
}

func (t *Table) First() int {
    for i := 0; i<10; i++ {
        if t.players[i]==p {
            return i 
        }
    }
    panic("unreachable")
}

//==========================================================================
//===============MAIN=======================================================
//==========================================================================
func main() {
    fmt.Println("Hello, world!")

}

//==========================================================================
//===============HELPERS====================================================
//==========================================================================
func guid() string {
    return s4() + s4() + "-" + s4() + "-" + s4() + "-" + s4() + "-" + s4() + s4() + s4()
}

func s4() string {
    s := ""
    for i := 0; i<4; i++ {
        n := rand.Int63n(16)
        s += strconv.FormatInt(n, 16)
    }
    return s           
}

func newDeck() (deck [52]string) {
    suits := []string{"S", "C", "D", "H"}
    ranks := []string{"2","3","4","5","6","7","8","9","T","J","Q","K","A"}
    i := 0 
    for _, suit := range suits {
        for _, rank := range ranks {
            deck[i]=rank+suit
            i++
        }
    }
    return deck 
}
