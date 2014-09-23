package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

var host = flag.String("host", "localhost:8080", "host:port location of server")
var user = flag.String("u", "user", "username to be used when authenticating")
var pass = flag.String("p", "password", "password to be used when authenticating")

func init() {
	flag.Parse()
	u := url.URL{Scheme: "http", Host: *host, Path: "users/"}
	fmt.Println(*host)
	req, err := http.NewRequest("POST", u.String(), nil)
	if err != nil {
		log.Fatalf("Could not start game: %v", err)
	}
	req.SetBasicAuth(*user, *pass)
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		log.Fatalf("Could not start game: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		log.Fatalf("Could not create user: %v", resp.Status)
	}
}

func main() {
	fmt.Println(*host)
	game, err := joinAnyGame(*host)
	if err != nil {
		log.Fatalf("Could not join game: %v", err)
	}
	fmt.Printf("Joined game %v.\n", game.gameID)
	for {
		buf := make([]byte, 1024)
		n, err := os.Stdin.Read(buf)
		if err != nil && err != io.EOF {
			panic(err)
		}
		input := strings.Fields(sanitize(buf[:n]))
		act, args := input[0], input[1:]
		switch act {
		case "bet":
			if len(args) == 0 {
				fmt.Println("Invalid input; need an amount to bet.")
			}
			bet, err := strconv.Atoi(args[0])
			if err != nil {
				fmt.Printf("Invalid input; could not parse %v as a number.\n", args[0])
			}
			if err := game.bet(bet); err != nil {
				fmt.Printf("Could not make bet: got error: %v\n", err)
			}
		case "call":
			if err := game.call(); err != nil {
				fmt.Printf("Could not make bet: got error: %v\n", err)
			}
		case "raise by":
			if len(args) == 0 {
				fmt.Println("Invalid input; need an amount to bet.")
			}
			bet, err := strconv.Atoi(args[0])
			if err != nil {
				fmt.Printf("Invalid input; could not parse %v as a number.\n", args[0])
			}
			if err := game.raiseBy(bet); err != nil {
				fmt.Printf("Could not make bet: got error: %v\n", err)
			}
		case "fold":
			if err := game.fold(); err != nil {
				fmt.Printf("Could not make bet: got error: %v\n", err)
			}
		}
	}
}

type Turn struct {
	Player      string `json:"player"`
	PlayerBet   int    `json:"player bet so far"`
	BetToPlayer int    `json:"bet to the player"`
	MinRaise    int    `json:"minimum raise"`
	Expiry      string `json:"expiry"`
}

type Act struct {
	Player    string
	Action    int
	BetAmount int
}

type Game struct {
	turn     *Turn
	user     string
	pass     string
	gameID   string
	playerID string
	betSoFar int
	url      url.URL
}

func joinAnyGame(host string) (*Game, error) {
	u := url.URL{Scheme: "http", Host: host}
	u.Path = "games/"
	game := &Game{user: *user, pass: *pass, url: u}
	retryTicker := time.NewTicker(500 * time.Millisecond)
	timeout := time.NewTimer(30 * time.Second)
	for {
		select {
		case <-retryTicker.C:
			gameList, err := getGameList(u.String())
			if err != nil {
				return &Game{}, err
			}
			if len(gameList) > 0 && len(gameList[0]) > 0 {
				fmt.Println(gameList)
				var gameID string
				for key, _ := range gameList[0] {
					if key == "GameID" {
						gameID = gameList[0][key].(string)
						break
					}
				}
				u.Path = ""
				turn := &Turn{}
				fmt.Println(gameList[0]["Turn"].(map[string]interface{}))
				serverTurn := gameList[0]["Turn"].(map[string]interface{})
				turn.Player = serverTurn["player"].(string)
				fmt.Println(serverTurn["player"].(string))
				turn.PlayerBet = int(serverTurn["player bet so far"].(float64))
				turn.BetToPlayer = int(serverTurn["bet to the player"].(float64))
				turn.MinRaise = int(serverTurn["minimum raise"].(float64))
				turn.Expiry = serverTurn["expiry"].(string)
				fmt.Println("turn in turn set = ", turn)
				game.turn = turn
				game.gameID = gameID
				err := game.join()
				if err != nil {
					return &Game{}, err
				}
				return game, nil
			} else {
				fmt.Println("Server has no active games; making game...")
				err = game.make()
				if err != nil {
					return &Game{}, err
				}
				game, err = joinAnyGame(host)
				return game, err
			}
		case <-timeout.C:
			return &Game{}, fmt.Errorf("did not join game; timed out trying to get game list")
		}
	}
	panic("unreachable")
}

func getGameList(listURL string) (gameList []map[string]interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	resp, err := http.Get(listURL)
	if err != nil {
		return gameList, err
	}
	dec := json.NewDecoder(resp.Body)
	dec.Decode(&gameList)
	return gameList, err
}

func (g *Game) join() error {
	u := g.url
	u.Path = "games/" + g.gameID + "/players/"
	req, err := http.NewRequest("POST", u.String(), nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(g.user, g.pass)
	resp, err := (&http.Client{}).Do(req)
	if resp.StatusCode != http.StatusCreated {
		raw := make([]byte, 1024)
		n, _ := resp.Body.Read(raw)
		body := sanitize(raw[:n])
		return fmt.Errorf(body)
	}
	dec := json.NewDecoder(resp.Body)
	var body map[string]interface{}
	err = dec.Decode(&body)
	if err != nil {
		return err
	}
	g.playerID = body["PlayerID"].(string)
	return nil
}

func (g *Game) make() error {
	fmt.Println("")
	g.url.Path = "games/"
	req, err := http.NewRequest("POST", g.url.String(), nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(g.user, g.pass)
	resp, err := (&http.Client{}).Do(req)
	var madeGame interface{}
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&madeGame)
	if err != nil {
		return err
	}
	fmt.Println()
	for id := range madeGame.(map[string]interface{}) {
		if id == "GameID" {
			g.gameID = madeGame.(map[string]interface{})[id].(string)
		}
		break
	}
	return nil
}

func (g *Game) bet(betAmount int) error {
	g.betSoFar += betAmount
	return g.act(1, betAmount)
}

func (g *Game) fold() error {
	g.betSoFar = 0
	return g.act(0, 0)
}

func (g *Game) call() error {
	fmt.Println("bet to player: ", g.turn.BetToPlayer, "; player bet : ", g.turn.PlayerBet)
	g.betSoFar = g.turn.BetToPlayer - g.turn.PlayerBet
	fmt.Println("about to bet ", g.betSoFar, "...")
	return g.act(1, g.betSoFar)
}

func (g *Game) raiseBy(betAmount int) error {
	if betAmount < g.turn.MinRaise {
		return fmt.Errorf("%v is less than the minimum raise of %v; action not submitted to server. Please submit another bet.", betAmount, g.turn.MinRaise)
	}
	g.betSoFar = g.turn.BetToPlayer + betAmount - g.betSoFar
	return g.act(1, g.turn.BetToPlayer+betAmount)
}

func (g *Game) act(action int, betAmount int) error {
	u := g.url
	u.Path = "games/" + g.gameID + "/players/" + g.playerID + "/acts/"
	act := &Act{Action: action, BetAmount: betAmount, Player: g.playerID}
	actJSON, err := json.Marshal(act)
	if err != nil {
		return err
	}
	bodyReader := bytes.NewReader(actJSON)
	fmt.Println(u.String())
	req, err := http.NewRequest("POST", u.String(), bodyReader)
	req.SetBasicAuth(g.user, g.pass)
	resp, err := (&http.Client{}).Do(req)
	raw := make([]byte, 1024)
	n, err := resp.Body.Read(raw)
	if err != nil && err != io.EOF {
		return err
	}
	body := sanitize(raw[:n])
	fmt.Println("bet status code:", resp.StatusCode)
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf(body)
	}
	return nil
}

func sanitize(raw []byte) string {
	str := string(raw)
	crlf := "\r\n"
	return strings.Replace(str, crlf, "", -1)
}
