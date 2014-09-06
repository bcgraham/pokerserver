package main

type Controller struct {

}
 
func (c *Controller) registerInvalidBet(g *Game, player guid, bet money)
func (c *Controller) removePlayerFromGame(g *Game,player guid)
func (c *Controller) getPlayerBet(g *Game, player guid) (int, money, error)
func (c *Controller) getNewPlayers(g *Game, numNeeded uint) (players []Player)
