package main

type Controller struct {

}
 
func (c *Controller) registerInvalidBet(g *Game, player guid, bet money){return}
func (c *Controller) removePlayerFromGame(g *Game,player guid){return}
func (c *Controller) getPlayerBet(g *Game, player guid) (int, money, error){return 2,money(2),nil}
func (c *Controller) getNewPlayers(g *Game, numNeeded uint) (players []Player){return []Player{}}
