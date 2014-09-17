package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"
)

var host = flag.String("host", "localhost:8080", "host:port location of server")
var user = flag.String("u", "user", "username to be used when authenticating")
var pass = flag.String("p", "password", "password to be used when authenticating")

func init() {
	u := url.URL{Scheme: "http", Host: *host, Path: "users/"}
	flag.Parse()
	req, err := http.NewRequest("POST", u.String(), nil)
	if err != nil {
		log.Fatalf("Could not start game: %v")
	}
	req.SetBasicAuth(*user, *pass)
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		log.Fatalf("Could not start game: %v")
	}
	if resp.StatusCode != http.StatusCreated {
		log.Fatalf("Could not create user: %v", resp.Status)
	}
}

func main() {
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
		case "call":
			cmd := exec.Command("goecho", strings.Join(args, " "))
			output, err := cmd.CombinedOutput()
			if err != nil {
				panic(err)
			}
			fmt.Println(string(output[:len(output)-1]))
		case "raise":
			fmt.Println("calling raise...")
			cmd := exec.Command("goecho", "raise")
			err := cmd.Run()
			if err != nil {
				panic(err)
			}
		case "fold":
			fmt.Println("calling fold...")
			cmd := exec.Command("goecho", "fold")
			err := cmd.Run()
			if err != nil {
				panic(err)
			}
		}
	}
}

type Turn struct {
	Player      string `json:"player"`
	BetToPlayer int    `json:"bet to the player"`
	MinRaise    int    `json:"minimum raise"`
	Expiry      string `json:"expiry"`
}

type Act struct {
	player    string
	action    int
	betAmount int
}

type Game struct {
	user     string
	pass     string
	gameID   string
	playerID string
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
				var gameID string
				for key, _ := range gameList[0] {
					if key == "GameID" {
						gameID = gameList[0][key].(string)
						break
					}
				}
				u.Path = ""
				turn := Turn{}
				serverTurn := gameList[0]["Turn"].(map[string]interface{})
				turn.Player = serverTurn["player"].(string)
				turn.BetToPlayer = int(serverTurn["bet to the player"].(float64))
				turn.MinRaise = int(serverTurn["minimum raise"].(float64))
				turn.Expiry = serverTurn["expiry"].(string)
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
				return game, nil
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

func sanitize(raw []byte) string {
	str := string(raw)
	crlf := "\r\n"
	return strings.Replace(str, crlf, "", -1)
}
