# Hacker School Poker Engine

## Overview
The Hacker School Poker Engine (HSPE) is a persistently running game server that coordinates and executes games of poker. Remote users can connect to HSPE and participate in games of poker against other remotely-connected users. The HSPE API provides a [Restful](http://en.wikipedia.org/wiki/Representational_State_Transfer) interface with lightweight [JSON](http://en.wikipedia.org/wiki/JSON)-formatted responses to interact with the HSPE.

This documentation provides an overview of how the Hacker School Poker Engine (HSPE) operates and how you can use the API to build AI bots to play games of poker.

## Game Rules
### General rules
Games of poker in HSPE are played using the [rules](http://www.wsop.com/poker-games/texas-holdem/rules/) of No Limit Texas Holdem. We also enforce the following rules:

- Each game has a maximum of 10 players allowed.

- Each game has a minimum of 2 players required to play the game. If you are the only player in a game the game will not start until another player joins.

- All players start with $10,000

- Big blinds are $20, Small blinds are $10

### Betting
Betting is allowed in increments no smaller than $1 (no floating point bets). All bets sent to HSPE must be valid. Invalid bets will lead to you immediately folding your hand. Invalid bets are:

- Bets with negative values

- Bets greater than your current wealth

- Bets less than the minimum required to call and less than your current wealth (you are allowed to bet less than the amount required to call if this bet would put you “all in”)

- Bets that raise on the previous bet, but do so less than the minimum raise amount as defined by the general rules of poker (you must raise at least as much as the last raise amount)

### Understanding the state of the game
It is the responsibility of a player to query HSPE at regular intervals and get the state of the game they are playing. There is currently no rate limiting enforced but we suggest players to query HSPE no more than once every 50ms

### Ordering and turns
A player can join a game at any time. If a hand is in progress, the player will start playing at the beginning of the next hand. If the game is full, the player will be put in queue and will start playing once there is a seat open at the table.

A player can fold and leave a game at any time.

HSPE will only accept bet/fold orders from a player when it is that player's turn. Orders sent when it is not that player's turn will be ignored. When it is a player's turn, they will have a 15 second window to send a bet/fold instruction before HSPE times out and removes the player from the game.

## Running the server
Fork the github repo, run ```go build```, then run ```pokerserver```. This will start up a server running on your machine on port 8080.

## Accessing the API
HSPE will be hosted on port 8080. Say you spin up a instance of the server running on ```127.0.0.1```. A request to retrieve all active games would look like:
    
    GET https://127.0.0.1:8080/games/

## Authentication
Some requests to HSPE need to be authenticated.  HSPE uses [HTTP basic authentication](http://en.wikipedia.org/wiki/Basic_access_authentication) for this purpose.

## Output formats
The bodies of successful responses will be sent in JSON format. Unsuccessful responses, those with error codes, maybe be sent as plain text not formatted as JSON.

## API Resources
The following documentation assume the server is running on  ```127.0.0.1:8080```. Replace this address with the appropriate location of the server.

### Make a new user
  
|                     | Details                      |
---------------------:|-------------------------------
**URI**               | https://127.0.0.1:8080/users/
**Synopsis**          | Create a new user
**HTTP Method**       | POST
**Parameters**        | --
**Success code**      | 201 Created
**Success body**      | string(GUID)
**Error response**    | 400 Bad Request if can’t parse request <br> 403 Forbidden if username already exists
**Error body**        | Error details (if applicable)
**Requires Auth**     | Y
**Notes**           | There is no second registration step. Authentication user and password are used to create new user.

### Get information about all active games

|                     |       Details                 |
---------------------:|-------------------------------|
**URI**               | https://127.0.0.1:8080/games/
**Synopsis**          | Get detailed information on all active games
**HTTP Method**       | GET
**Parameters**        | --
**Success code**      | 200 OK
**Success body**      | array(Game)
**Error response**    | --
**Error body**        | --
**Requires Auth**     | N
**Notes**             | If authenticated request is made from a user who is playing in the game, that player’s “hole” cards will be included in the response

### Get information about a specific game

|                     |       Details                 |
---------------------:|-------------------------------|
**URI**               |  https://127.0.0.1:8080/games/:gameID
**Synopsis**          |  Get detailed information on game with :gameID
**HTTP Method**       |  GET
**Parameters**        |  --
**Success code**      |  200 OK
**Success body**      |  Game
**Error response**    |  404 Not Found if can’t find :gameID <br> 401 Unauthorized if auth credentials are invalid
**Error body**        |  Error details (if applicable)
**Requires Auth**     |  N
**Notes**             |  If authenticated request is made from a user who is playing in the game, that player’s “hole” cards will be included in the response

### Join a game
|                     |       Details                 |
---------------------:|-------------------------------|
**URI**               | https://127.0.0.1:8080/games/:gameID/players/
**Synopsis**          | Join game :gameID
**HTTP Method**       | POST
**Parameters**        | --
**Success code**      | 201 Created if able to join game <br> 202 Accepted if table is full but game does exist
**Success body**      | string(GUID)
**Error response**    | 404 Not Found if can’t find :gameID <br> 401 Unauthorized if auth credentials are invalid
**Error body**        | Error details (if applicable)
**Requires Auth**     | Y
**Notes**             | --

### Bet or Fold
|                     |       Details                 |
---------------------:|-------------------------------|
**URI**               | https://127.0.0.1:8080/games/:gameID/players/:playerID/acts/
**Synopsis**          | Place a bet or fold
**HTTP Method**       | POST
**Parameters**        | Act
**Success code**      | 201 Created
**Success body**      | --
**Error response**    | 403 Forbidden if trying to act when it’s not your turn <br> 404 Not Found if can’t find :gameID or :playerID <br> 401 Unauthorized if auth credentials are invalid
**Error body**        | Error details (if applicable)
**Requires Auth**     | Y
**Notes**             | --

### Leave game
|                     |       Details                 |
---------------------:|-------------------------------|
**URI**              | https://127.0.0.1:8080/games/:gameID/players/:playerID
**Synopsis**         | Leave the game, forfeit winnings
**HTTP Method**      | DELETE
**Parameters**       | --
**Success code**     | 200 OK
**Success body**     | --
**Error response**   | 404 Not Found if can’t find :gameID or :playerID <br> 401 Unauthorized if auth credentials are invalid
**Error body**       | Error details (if applicable)
**Requires Auth**    | Y
**Notes**            | -- 

## Types
### Act
**Fields**

| Name                     | Type                 | Description     |
---------------------------|----------------------|-----------------|
action                   | int   | 0 for "fold", 1 for "bet"
betAmount                   | uint   | How much to bet

### Game
**Fields**

| Name                     | Type                 | Description     |
---------------------------|----------------------|-----------------|
gameID           | string           | GUID for this game
table           | array(Player)           | Who is actively playing
turn           | Turn           | Whose turn it is
cards           | dict[string:array(string)]           | Dealt cards up to this point
pots           | array(Pot)           | Money bet so far.

**Notes** <br>
Each round will have at least one Pot. Some rounds might have multiple pots if side pots are needed. Pots are ordered chronologically earliest to latest

### Player
**Fields**

| Name                     | Type                 | Description     |
---------------------------|----------------------|-----------------|
playerID           | string           | GUID identifying player
handle           | string           | Player username
state           | string           | “active” - Player needs to bet           <br> “folded” - Player has folded their hand           <br> “called” - Player has called or raised
wealth           | int           | Total money player has
bet_so_far           | int           | Amount bet so far in round
small_blind           | boolean           | Is this player small blind?

### Turn
**Fields**

| Name                     | Type                 | Description     |
---------------------------|----------------------|-----------------|
playerID    | string    | GUID identifying player
bet_so_far    | int    | Amount bet so far in round
bet_to_player    | int    | Total amount that must be matched to call
minimum_raise    | int    | Minimum amount you can raise
expiry    | datetime    | Time you need to respond by in order to stay in the game

### Cards Dictionary
**Fields**

| Name                     | Type                 | Description     |
---------------------------|----------------------|-----------------|
hole    | array(string)    | Player’s hole cards. Only populated if request is authenticated
flop    | array(string)    | Three cards dealt at the “Flop”
turn    | array(string)    | Card dealt at the “Turn”
river    | array(string)    | Card dealt at the “River”

**Notes** <br>
flop, turn, and river are populated when the appropriate round is reached, and are otherwise null values. Cards are represented as strings of two characters. For example, the ace of spades is “AS”, the two of hearts is “2H”, and the ten of clubs is “TC”.

### Pot
**Fields**

| Name                     | Type                 | Description     |
---------------------------|----------------------|-----------------|
size     | int     | Amount of money in this pot
players     | array(GUID)     | Players who have bet into this pot


### Example JSON response 
Request:

```GET http://127.0.0.1/games/fbbbbb44-bc4f-c3f6-6519-d81bd1a66d8a```

Response:
```.json
[
   {
      "gameID":"fbbbbb44-bc4f-c3f6-6519-d81bd1a66d8a",
      "table":[
         {
            "playerID":"2fd318e4-0947-64b9-8b63-14f5a5824b8b",
            "handle":"",
            "state":"active",
            "wealth":15002,
            "bet_so_far":0,
            "small blind":true
         },
         {
            "playerID":"0dd7d693-06f9-b5ba-694d-f8779579fa8a",
            "handle":"",
            "state":"active",
            "wealth":4006,
            "bet_so_far":0,
            "small blind":false
         }
      ],
      "turn":{
         "playerID":"2fd318e4-0947-64b9-8b63-14f5a5824b8b",
         "bet_so_far":0,
         "bet_to_player":0,
         "minimum_raise":0,
         "expiry":"2014-09-24 15:14:53.021303798 -0400 EDT"
      },
      "cards":{
         "hole":[
            "AC",
            "TD"
         ],
         "flop":[
            "2S",
            "3S",
            "JS"
         ],
         "turn":[
            "4D"
         ],
         "river":[
            "AH"
         ]
      },
      "pots":[
         {
            "size":60,
            "players":[
               "8c479795-adcb-a3c1-bebd-4bdf54016f4a",
               "2fd318e4-0947-64b9-8b63-14f5a5824b8b",
               "0dd7d693-06f9-b5ba-694d-f8779579fa8a"
            ]
         },
         {
            "size":0,
            "players":[
               "2fd318e4-0947-64b9-8b63-14f5a5824b8b",
               "0dd7d693-06f9-b5ba-694d-f8779579fa8a"
            ]
         },
         {
            "size":0,
            "players":[
               "2fd318e4-0947-64b9-8b63-14f5a5824b8b",
               "0dd7d693-06f9-b5ba-694d-f8779579fa8a"
            ]
         }
      ]
   }
]
```


