//Utility Functions
var guid = (function() {
  function s4() {
    return Math.floor((1 + Math.random()) * 0x10000)
               .toString(16)
               .substring(1);
  }
  return function() {
    return s4() + s4() + '-' + s4() + '-' + s4() + '-' +
           s4() + '-' + s4() + s4() + s4();
  };
})();

var Player = function()){
    
}

var Game = function(){
    //Member State
    table : [undefined,undefined,undefined,undefined,undefined,
             undefined,undefined,undefined,undefined,undefined], //INVARIANT: Small blind = table[0]
    rounds : [], //List of [int roundNumber, player guid, int bet] tuples
    gameID : guid(),
    handID : undefined, // Current hand
    deck : this.newDeck(), //Cards can be "DECK", "FLOP", "TURN", "RIVER", "Player GUID"
}

Game.prototype = {
    newDeck : function(){
       suits = ['S', 'C', 'D', 'H']
        ranks = ['2','3','4','5','6','7','8','9','T', 'J','Q','K','A']
        cards = {}

        for (var i = suits.length - 1; i >= 0; i--) {
            for (var j = ranks.length - 1; j >= 0; j--) {
                label = ranks[j] + suits[i];
                cards[label] = 'DECK';
            };
        }; 

        return cards;
    },
    next : function(){

    },
    dealTo: function(recipient){
        //Assign "recipient" to a random card in deck that currently has value "DECK"
        //Returns true on success, false othewise
    },
    dealInitialCards : function(){

    },
    dealFlop : function(){

    },
    dealTurn : function(){

    },
    dealRiver : function(){

    },
    bet : function(player,amount){
        //Returns true if bet was placed successfully
    },
}
