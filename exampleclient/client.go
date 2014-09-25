package main

import (
	"bytes"
	"crypto/tls"
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
var useTLS = flag.Bool("tls", false, "if enabled, will try to communicate over HTTPS")

func main() {
	flag.Parse()
	scheme := "http"
	if *useTLS {
		scheme = "https"
	}
	u := url.URL{Scheme: scheme, Host: *host, Path: "users/"}
	req, err := http.NewRequest("POST", u.String(), nil)
	if err != nil {
		log.Fatalf("Could not start game: %v", err)
	}
	req.SetBasicAuth(*user, *pass)
	resp, err := client(*useTLS).Do(req)
	if err != nil {
		log.Fatalf("Could not connect to host %v: %v", *host, err)
	}
	if resp.StatusCode != http.StatusCreated {
		log.Fatalf("Could not create user: %v", resp.Status)
	}
	contents := make([]byte, 512)
	n, err := resp.Body.Read(contents)
	if err != nil && err != io.EOF {
		log.Fatalf("Couldn't make user: %v\n", err)
	}
	resp.Body.Close()
	playerID := new(string)
	err = json.Unmarshal(contents[:n], &struct {
		PlayerID *string `json:"playerID"`
	}{PlayerID: playerID})
	if err != nil {
		log.Fatalf("Could not start game: %v", err)
	}
	game, err := joinAnyGame(scheme, *host, *user, *pass)
	game.PlayerID = *playerID
	for {
		if game.GameID != "" {
			err := game.poll()
			if err != nil {
				fmt.Println("poll returned err: ", err)
			}
			// fmt.Printf("\nYour turn! \n%v", cardPrinter(game.Cards))
			// if game.Turn.BetToPlayer == 0 {
			// 	fmt.Printf("There is no current bet.\n")
			// } else {
			// 	fmt.Printf("The current bet is %v. You have %v in play.\nIt would cost you %v to call. The minimum raise is %v.\n", game.Turn.BetToPlayer, game.Turn.PlayerBet, game.Turn.BetToPlayer-game.Turn.PlayerBet, game.Turn.MinRaise)
			// }
			game.call()
			continue
		}
		buf := make([]byte, 1024)
		n, err := os.Stdin.Read(buf)
		if err != nil && err != io.EOF {
			panic(err)
		}
		input := strings.Fields(sanitize(buf[:n]))
		act, args := input[0], input[1:]
		switch act {
		case "list":
			gameList, err := getGameList(scheme, *host)
			if err != nil {
				log.Printf("Could not list games: %v\n", err)
				continue
			}
			for _, g := range gameList {
				fmt.Println(g.GameID)
			}
			continue
		case "joinany":
			game, err = joinAnyGame(scheme, *host, *user, *pass)
			game.PlayerID = *playerID
			if err != nil {
				log.Fatalf("Could not join game: %v", err)
			}
			continue
		case "join":
			if len(args) == 0 {
				fmt.Println("Invalid input; need a game to join.")
			}
			continue
		case "make":
			game = &Game{User: *user, Pass: *pass, PlayerID: *playerID, URL: url.URL{Scheme: scheme, Host: *host}}
			err = game.make()
			if err != nil {
				fmt.Printf("Could not make game; received error: %v\n", err)
			}
			continue
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
			continue
		case "call":
			if err := game.call(); err != nil {
				fmt.Printf("Could not make bet: got error: %v\n", err)
			}
			continue
		case "check":
			if game.Turn.BetToPlayer > game.Turn.PlayerBet {
				fmt.Print("You can't check. You have to call.\n")
				continue
			}
			if err := game.call(); err != nil {
				fmt.Printf("Could not make bet: got error: %v\n", err)
			}
			continue
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
		Player      string `json:"playerID"`
		PlayerBet   int    `json:"bet_so_far"`
		BetToPlayer int    `json:"bet_to_player"`
		MinRaise    int    `json:"minimum_raise"`
		Expiry      string `json:"expiry"`
	} `json:"turn"`
	Cards struct {
		Hole  []string
		Flop  []string
		Turn  []string
		River []string
	}
	Pots []struct {
		Size    int
		Players []string
	}
	User     string
	Pass     string
	GameID   string `json:"gameID"`
	PlayerID string
	BetSoFar int
	URL      url.URL
}

func joinAnyGame(scheme, host, user, pass string) (*Game, error) {
	retryTicker := time.NewTicker(500 * time.Millisecond)
	timeout := time.NewTimer(30 * time.Second)
	for {
		select {
		case <-retryTicker.C:
			gameList, err := getGameList(scheme, host)
			if err != nil {
				return &Game{}, err
			}
			if len(gameList) > 0 {
				game := gameList[0]
				game.User, game.Pass, game.URL = user, pass, url.URL{Scheme: scheme, Host: host}
				err := game.join()
				if err != nil {
					return &Game{}, err
				}
				return game, nil
			} else {
				game := &Game{User: user, Pass: pass, URL: url.URL{Scheme: scheme, Host: host}}
				err = game.make()
				if err != nil {
					return &Game{}, err
				}
				return game, err
			}
		case <-timeout.C:
			return &Game{}, fmt.Errorf("did not join game; timed out trying to get game list")
		}
	}
	panic("unreachable")
}

func getGameList(scheme, host string) (gameList []*Game, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	u := url.URL{Scheme: scheme, Host: host}
	u.Path = "games/"
	resp, err := client(*useTLS).Get(u.String())
	if err != nil {
		return gameList, err
	}
	defer resp.Body.Close()
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
	resp, err := client(*useTLS).Do(req)
	defer resp.Body.Close()
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
	resp, err := client(*useTLS).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
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
	g.BetSoFar = g.Turn.BetToPlayer
	return g.act(1, g.BetSoFar-g.Turn.PlayerBet)
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
	act := &Act{Action: action, BetAmount: betAmount}
	actJSON, err := json.Marshal(act)
	if err != nil {
		return err
	}
	bodyReader := bytes.NewReader(actJSON)
	req, err := http.NewRequest("POST", u.String(), bodyReader)
	req.SetBasicAuth(g.User, g.Pass)
	resp, err := client(*useTLS).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw := make([]byte, 1024)
	n, err := resp.Body.Read(raw)
	if err != nil && err != io.EOF {
		return err
	}
	body := sanitize(raw[:n])
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf(body)
	}
	return nil
}

func (g *Game) poll() error {
	u := g.URL
	u.Path = "games/" + g.GameID + "/"
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(g.User, g.Pass)
	tempGame := new(Game)
	for {
		resp, err := client(*useTLS).Do(req)
		if err != nil {
			return fmt.Errorf("Error doing request: %v", err)
		}
		if resp.StatusCode == http.StatusForbidden {
			time.Sleep(10 * time.Millisecond)
			resp.Body.Close()
			continue
		}
		contents := make([]byte, 10240)
		n, err := resp.Body.Read(contents)
		if err != nil && err != io.EOF {
			resp.Body.Close()
			return fmt.Errorf("Error reading contents of response body: %v", err)
		}
		err = json.Unmarshal(contents[:n], tempGame)
		if err != nil {
			resp.Body.Close()
			return fmt.Errorf("Error unmarshaling json: %v\n.\n%v\n", err, string(contents[:n]))
		}
		if tempGame.Turn.Player == g.PlayerID {
			tempGame.User = g.User
			tempGame.Pass = g.Pass
			tempGame.GameID = g.GameID
			tempGame.PlayerID = g.PlayerID
			tempGame.BetSoFar = g.BetSoFar
			tempGame.URL = g.URL
			*g = *tempGame
			return nil
		}
		resp.Body.Close()
	}
}

func sanitize(raw []byte) string {
	str := string(raw)
	crlf := "\r\n"
	return strings.Replace(str, crlf, "", -1)
}

func cardPrinter(cards struct {
	Hole  []string
	Flop  []string
	Turn  []string
	River []string
}) string {
	s := fmt.Sprintf("You're holding %v.\n", cards.Hole)
	c := "["
	for _, card := range cards.Flop {
		c += card
		c += ","
	}
	for _, card := range cards.Turn {
		c += card
		c += ","
	}
	for _, card := range cards.River {
		c += card
		c += ","
	}
	c = string(c[:len(c)-1])
	if len(c) == 0 {
		s += fmt.Sprintf("There are no table cards showing.\n")
	} else {
		c += "]"
		s += fmt.Sprintf("The table is showing %v.\n", c)
	}
	return s
}

func client(useTLS bool) *http.Client {
	if useTLS {
		return &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
	}
	return &http.Client{}
}
