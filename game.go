//Utility Functions
package main 
import (
    "math/rand"
    "math"
    "strconv"
    "fmt"
)

//==========================================================================
//===============TYPES AND CONSTS===========================================
//==========================================================================
var SEED int64 = 0 // seed for deal 
type roundName int 
type state int 
type money uint64
type Deck map[string]string 
type guid string
const (
    fold int = iota
    bet
    call
)
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

func (g *Game) deal(seed int64) {
    unshuffled := newDeck()
    numPlayers := len(g.table.players)
    r := rand.New(rand.NewSource(seed))
    shuffled := rand.Perm(52)
    for i := 0; i < numPlayers; i = i + 2 {
        card1, card2 := unshuffled[i], unshuffled[i+1]
        g.deck[card1] = string(g.table.players[i].guid)
        g.deck[card2] = string(g.table.players[i].guid)
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
    smallBlind money
    controller *Controller 
}

func (g *Game) run() {
    for {
        g.addWaitingPlayersToGame()
        g.removeBrokePlayers()
        g.betBlinds()
        g.deal(SEED)  
        for i := 0 ; g.notSettled() && i < 4; i++{
            g.Bets()
            g.table.ResetRound()
            g.pot.newRound()
        }
        g.resolveBets()
        g.table.AdvanceButton()
        g.table.ResetHand()
    }
}

//notSettled returns true if >1 player is not folded
func (g *Game) notSettled() bool {
    notFolded := 0
    for _,p := range g.table.players {
        if p.state != folded {
            notFolded++
        }
    }
    return notFolded > 1
}

func (g *Game) addWaitingPlayersToGame() {
    numPlayersNeeded := uint(10 - len(g.table.players))
    newPlayers := g.controller.getNewPlayers(g, numPlayersNeeded)
    for _,p := range newPlayers {
        err := g.table.addPlayer(p.guid)
        if err != nil {
            panic(err)
        }
    }
}

func (g *Game) removeBrokePlayers() {
    for _,p := range g.table.players {
        if p.wealth == 0 {
            p.state = folded
            g.controller.removePlayerFromGame(g, p.guid)
        } else if p.wealth < 0 {
            panic("player has < 0 wealth!")
        }
    }
}

func (g *Game) betBlinds(){
    //Bet small blind
    player := g.table.Next()
    if player.wealth >= g.smallBlind {
        g.commitBet(player, g.smallBlind)
    } else {
        g.commitBet(player, player.wealth)
    }

    //Bet big blind
    player = g.table.Next()
    if player.wealth >= 2*g.smallBlind {
        g.commitBet(player, 2*g.smallBlind)
    } else {
        g.commitBet(player, player.wealth)
    }
}

//setBlinds sets the money amount for the blinds
// and rotates the "button"
func (g *Game) setBlinds() {
    g.smallBlind = 25
}

func (g *Game) betsNeeded() bool {
    numActives := 0
    numCalled := 0
    for _, p := range g.table.players {
        if p.state == active{
            numActives++
        } else if p.state == called {
            numCalled++
        }
    }
    return (numActives >= 1) && (numCalled >= 1)
}

//Bets gets the bet from each player
func (g *Game) Bets() {
    table := g.table 

    for player := g.table.Next(); g.betsNeeded(); player = g.table.Next() {
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
        if g.betInvalid(player, betAmount) {
            g.controller.registerInvalidBet(g, player.guid, betAmount)
            player.state = folded
            continue
        }

        //Legit bets
        isRaising := (g.pot.totalPlayerBetThisRound(player.guid) + betAmount) > g.pot.totalToCall
        if isRaising { 
            g.table.ResetRoundPlayerState()
        }
        g.commitBet(player, betAmount)
        player.state = called
    }

}

func (g *Game) betInvalid(player *Player, bet money) (bool) {
    return (bet > player.wealth) || 
           (bet < g.pot.minRaise) || 
           (bet < player.wealth && (g.pot.totalPlayerBetThisRound(player.guid) + bet) < g.pot.totalToCall)
}

func (g *Game) commitBet(player *Player, amount money) {
    if amount <= 0 {
        panic("trying to bet <= 0")
    }
    g.pot.receiveBet(player.guid, amount)
    player.wealth -= amount
}

//==========================================================================
//===============POT AND BET================================================
//==========================================================================
type Pot struct{
    minRaise money
    totalToCall money
    potNumber uint
    bets []Bet
}

type Bet struct {
    potNumber uint 
    player guid
    value money
}

//Resolves partial pots from previous round
// increments potNumber to a previously unused number
func (pot *Pot) newRound() {
    pot.condenseBets()
    pot.makeSidePots()
    pot.minRaise = 0
    pot.totalToCall = 0
}

func (pot *Pot) condenseBets(){
    playerBets := make(map[guid]money)
    betsCopy := make([]Bet,0)
    for _,bet := range pot.bets {
        if bet.potNumber == pot.potNumber {
            playerBets[bet.player] += bet.value
        } else {
            betsCopy = append(betsCopy, bet)
        }
    }
    for k,v := range playerBets {
        betsCopy = append(betsCopy, Bet{potNumber:pot.potNumber, player: k, value:v})
    }

    pot.bets = betsCopy
}

func (pot *Pot) allBetsEqual(potNumber uint) bool {
    var prevBet money
    for _,bet := range pot.bets {
        if bet.potNumber == potNumber{
            prevBet = bet.value
            break
        }
    }
    for _,bet := range pot.bets {
        if bet.potNumber != potNumber{
            continue
        }
        if prevBet != bet.value{
            return false
        }
    }
    return true
}

func (pot *Pot) makeSidePots() {    
    if pot.allBetsEqual(pot.potNumber) {
        return
    }
    pot.potNumber++ //Make a new pot

    minimum := money(math.MaxUint64)
    for _,b := range pot.bets {
        if b.value < minimum && b.potNumber == (pot.potNumber-1){
            minimum = b.value
        }
    }
    for _,b := range pot.bets {
        if b.value > minimum && b.potNumber == (pot.potNumber-1){
            excess := b.value - minimum
            b.value = minimum
            pot.bets = append(pot.bets, Bet{potNumber: pot.potNumber, player:b.player , value:excess})
        }
    }
    pot.makeSidePots() //Call again to split new side pot into more side pots if necessary
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

func (pot *Pot) totalInPot() (money) {
    var sum money = 0
    for _,m := range pot.bets {
        sum += m.value
    }
    return sum
}

func (pot *Pot) totalPlayerBetThisRound(g guid) (money) {
    sum := money(0)
    for _,m := range pot.bets {
        if m.player == g && m.potNumber == pot.potNumber {
            sum += m.value
        }
    }
    return sum
}

//==========================================================================
//===============TABLE======================================================
//==========================================================================
type Table struct {
    players []*Player
    index int 
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
    for i:=0; i < (len(t.players)-1);i++{
        t.players[i] = t.players[i+1]
    }
    t.players[n - 1] = last
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

//==========================================================================
//===============MAIN=======================================================
//==========================================================================
func main() {
    fmt.Println("Hello, world!")

}

//==========================================================================
//===============HELPERS====================================================
//==========================================================================
func createGuid() string {
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
