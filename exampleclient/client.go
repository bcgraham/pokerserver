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
	fmt.Println("User: ", *user, "; Pass: ", *pass)
	req.SetBasicAuth(*user, *pass)
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		log.Fatalf("Could not start game: %v", err)
	}
	fmt.Println("From making user: ", resp.Status)
	if resp.StatusCode != http.StatusCreated {
		log.Fatalf("Could not create user: %v", resp.Status)
	}
}

func main() {
	flag.Parse()
	fmt.Println(*host)
	game, err := joinAnyGame(*host, *user, *pass)
	if err != nil {
		log.Fatalf("Could not join game: %v", err)
	}
	fmt.Printf("Joined game %v.\n", game.GameID)
	for {
		buf := make([]byte, 1024)
		n, err := os.Stdin.Read(buf)
		if err != nil && err != io.EOF {
			panic(err)
		}
		input := strings.Fields(sanitize(buf[:n]))
		act, args := input[0], input[1:]
		switch act {
		case "list":
			gameList, err := getGameList(*host)
			if err != nil {
				log.Printf("Could not list games: %v\n", err)
				continue
			}
			for _, g := range gameList {
				fmt.Println(g.GameID)
				continue
			}
		case "bet":
			if len(args) == 0 {
				fmt.Println("Invalid input; need an amount to bet.")
				continue
			}
			bet, err := strconv.Atoi(args[0])
			if err != nil {
				fmt.Printf("Invalid input; could not parse %v as a number.\n", args[0])
				continue
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
				continue
			}
			bet, err := strconv.Atoi(args[0])
			if err != nil {
				fmt.Printf("Invalid input; could not parse %v as a number.\n", args[0])
				continue
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

type Act struct {
	Player    string
	Action    int
	BetAmount int
}

type Game struct {
	Table []struct {
		GUID       string
		Handle     string
		State      string
		Wealth     int
		InFor      int
		SmallBlind bool
	}
	Turn struct {
		Player      string `json:"player"`
		PlayerBet   int    `json:"player bet so far"`
		BetToPlayer int    `json:"bet to the player"`
		MinRaise    int    `json:"minimum raise"`
		Expiry      string `json:"expiry"`
	}
	Cards struct {
		Hole  []string
		Flop  []string
		Turn  string
		River string
	}
	Pots []struct {
		Size    int
		Players []string
	}
	User     string
	Pass     string
	GameID   string
	PlayerID string
	BetSoFar int
	URL      url.URL
}

func joinAnyGame(host, user, pass string) (*Game, error) {
	retryTicker := time.NewTicker(500 * time.Millisecond)
	timeout := time.NewTimer(30 * time.Second)
	for {
		select {
		case <-retryTicker.C:
			gameList, err := getGameList(host)
			if err != nil {
				return &Game{}, err
			}
			if len(gameList) > 0 {
				fmt.Println(gameList)
				game := gameList[0]
				game.User = user
				game.Pass = pass
				game.URL = url.URL{Scheme: "http", Host: host}
				err := game.join()
				if err != nil {
					return &Game{}, err
				}
				return game, nil
			} else {
				fmt.Println("Server has no active games; making game...")
				game := &Game{User: user, Pass: pass, URL: url.URL{Scheme: "http", Host: host}}
				err = game.make()
				if err != nil {
					return &Game{}, err
				}
				fmt.Println("About to try to join...")
				err = game.join()
				if err != nil {
					fmt.Println("Could not join...")
					return &Game{}, err
				}
				fmt.Println("Seemingly joined okay...")
				return game, err
			}
		case <-timeout.C:
			return &Game{}, fmt.Errorf("did not join game; timed out trying to get game list")
		}
	}
	panic("unreachable")
}

func getGameList(host string) (gameList []*Game, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	u := url.URL{Scheme: "http", Host: host}
	u.Path = "games/"
	resp, err := http.Get(u.String())
	if err != nil {
		return gameList, err
	}
	contents := make([]byte, 131072) // should be enough, but will fail poorly if game list is longer
	n, err := resp.Body.Read(contents)
	if err != nil && err != io.EOF {
		return gameList, err
	}
	gameList = make([]*Game, 0)
	err = json.Unmarshal(contents[:n], &gameList)
	if err != nil {
		return make([]*Game, 0), err
	}
	return gameList, err
}

func (g *Game) join() error {
	u := g.URL
	u.Path = "games/" + g.GameID + "/players/"
	req, err := http.NewRequest("POST", u.String(), nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(g.User, g.Pass)
	resp, err := (&http.Client{}).Do(req)
	if resp.StatusCode != http.StatusAccepted {
		raw := make([]byte, 1024)
		n, _ := resp.Body.Read(raw)
		body := sanitize(raw[:n])
		return fmt.Errorf(body)
	}
	contents := make([]byte, 10240)
	n, err := resp.Body.Read(contents)
	if err != nil && err != io.EOF {
		return err
	}
	err = json.Unmarshal(contents[:n], g)
	if err != nil {
		return err
	}
	return nil
}

func (g *Game) make() error {
	u := g.URL
	u.Path = "games/"
	req, err := http.NewRequest("POST", u.String(), nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(g.User, g.Pass)
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return err
	}
	contents := make([]byte, 10240)
	n, err := resp.Body.Read(contents)
	if err != nil && err != io.EOF {
		return err
	}
	err = json.Unmarshal(contents[:n], g)
	if err != nil {
		return err
	}
	return nil
}

func (g *Game) bet(betAmount int) error {
	g.BetSoFar += betAmount
	return g.act(1, betAmount)
}

func (g *Game) fold() error {
	g.BetSoFar = 0
	return g.act(0, 0)
}

func (g *Game) call() error {
	fmt.Println("bet to player: ", g.Turn.BetToPlayer, "; player bet : ", g.Turn.PlayerBet)
	g.BetSoFar = g.Turn.BetToPlayer - g.Turn.PlayerBet
	fmt.Println("about to bet ", g.BetSoFar, "...")
	return g.act(1, g.BetSoFar)
}

func (g *Game) raiseBy(betAmount int) error {
	if betAmount < g.Turn.MinRaise {
		return fmt.Errorf("%v is less than the minimum raise of %v; action not submitted to server. Please submit another bet.", betAmount, g.Turn.MinRaise)
	}
	g.BetSoFar = g.Turn.BetToPlayer + betAmount - g.BetSoFar
	return g.act(1, g.Turn.BetToPlayer+betAmount)
}

func (g *Game) act(action int, betAmount int) error {
	u := g.URL
	u.Path = "games/" + g.GameID + "/players/" + g.PlayerID + "/acts/"
	act := &Act{Action: action, BetAmount: betAmount, Player: g.PlayerID}
	actJSON, err := json.Marshal(act)
	if err != nil {
		return err
	}
	bodyReader := bytes.NewReader(actJSON)
	fmt.Println(u.String())
	req, err := http.NewRequest("POST", u.String(), bodyReader)
	req.SetBasicAuth(g.User, g.Pass)
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
