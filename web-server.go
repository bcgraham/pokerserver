package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync"
	"time"

	mux "github.com/gorilla/mux"
)

const COST = 10

func parseAuthHeader(header string) (credentials []string, err error) {
	authinfo := strings.Fields(header)
	if len(authinfo) != 2 {
		return credentials, errors.New("You must provide properly-formatted credentials. Expecting HTTP Basic Authentication scheme and base64-encoded username:password pair.")
	}
	scheme := authinfo[0]
	if scheme != "Basic" {
		return credentials, errors.New("You must provide properly-formatted credentials. You either did not use the HTTP Basic Authentication scheme or did not provide a username/password field. We have not yet checked whether it is a valid base64 encoding; we have only checked whether the field was present at all.")
	}
	raw, err := base64.StdEncoding.DecodeString(authinfo[1])
	if err != nil {
		return credentials, errors.New("You must provide properly-formatted credentials. We could not decode your base64-encoded username/password combination.")
	}
	credentials = strings.SplitN(string(raw), ":", 2)
	if len(credentials) != 2 {
		return credentials, errors.New("You must provide properly-formatted credentials. Your plaintext username and password must be separated by a colon.")
	}
	return credentials, nil
}

func protector(um *UserMap, restricted func(http.ResponseWriter, *http.Request, guid)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		credentials, err := parseAuthHeader(r.Header.Get("Authorization"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		username := credentials[0]
		submitted := credentials[1]
		playerID := um.handles[username]
		password, ok := um.passwords[playerID]
		if !ok {
			http.Error(w, "Invalid credentials [1].", http.StatusUnauthorized)
			return
		}
		if submitted != password {
			http.Error(w, "Invalid credentials [2].", http.StatusUnauthorized)
			return
		}
		// we have successfully authenticated the user; she is who she says she is.
		restricted(w, r, playerID)
	}
}

func (re RestExposer) getGames(w http.ResponseWriter, r *http.Request) {
	enc := json.NewEncoder(w)
	pgs := make([]PublicGame, 0)
	for _, g := range re.gc.Games {
		pg := re.gc.getGame(g.gameID)
		pgs = append(pgs, pg)
	}
	err := enc.Encode(&pgs)
	if err != nil {
		fmt.Println(err)
	}
}

func (re RestExposer) makeGame(w http.ResponseWriter, r *http.Request, verifiedPlayerID guid) {
	pg := re.gc.makeGame()
	g := re.gc.Games[guid(pg.GameID)]
	joinGame(w, g, verifiedPlayerID)
}

func (re RestExposer) getGame(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if _, ok := re.gc.Games[guid(vars["GameID"])]; !ok {
		http.Error(w, "Game not found.", http.StatusNotFound)
		return
	}
	pg := re.gc.getGame(guid(vars["GameID"]))
	enc := json.NewEncoder(w)
	enc.Encode(pg)
}

func (re RestExposer) getGameAuthenticated(w http.ResponseWriter, r *http.Request, verifiedPlayerID guid) {
	vars := mux.Vars(r)
	g, ok := re.gc.Games[guid(vars["GameID"])]
	if !ok {
		log.Fatalf("no such game: %v", vars["GameID"])
	}
	if !g.table.contains(verifiedPlayerID) {
		http.Error(w, "The authenticated player has not joined this game.", http.StatusForbidden)
		return
	}
	pg := re.gc.getGamePrivate(guid(vars["GameID"]), verifiedPlayerID)
	enc := json.NewEncoder(w)
	enc.Encode(pg)
}

func (re RestExposer) getPlayers(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	g, ok := re.gc.Games[guid(vars["GameID"])]
	if !ok {
		// TODO: handle errors
		log.Fatalf("no such game: %v", vars["GameID"])
	}
	enc := json.NewEncoder(w)
	enc.Encode(g.table)
}

func (re RestExposer) playerJoinGame(w http.ResponseWriter, r *http.Request, verifiedPlayerID guid) {
	vars := mux.Vars(r)
	gameID := guid(vars["GameID"])
	g, ok := re.gc.Games[gameID]
	if !ok {
		http.Error(w, fmt.Sprintf("Could not find game: \"%v\"", gameID), http.StatusNotFound)
		return
	}
	joinGame(w, g, verifiedPlayerID)
}

func (re RestExposer) quitPlayer(w http.ResponseWriter, r *http.Request, verifiedPlayerID guid) {
	vars := mux.Vars(r)
	err := r.ParseForm()
	if err != nil {
		log.Fatalf("couldn't parse form")
	}
	g, ok := re.gc.Games[guid(vars["GameID"])]
	if !ok {
		// TODO: handle errors
		log.Fatalf("no such game: %v", vars["GameID"])
	}
	playerID := guid(vars["PlayerID"])
	if !g.table.contains(playerID) {
		// TODO: handle errors
		log.Fatalf("no such player: %v", playerID)
	}
	g.table.remove(playerID) // does not exist
	//send confirmation of kill
}

func (re RestExposer) makeAct(w http.ResponseWriter, r *http.Request, verifiedPlayerID guid) {
	vars := mux.Vars(r)
	gameID := guid(vars["GameID"])
	playerID := guid(vars["PlayerID"])
	limited := &io.LimitedReader{R: r.Body, N: 1048576} // meg of json ought to be enough
	data, err := ioutil.ReadAll(limited)
	if err != nil {
		log.Fatalf("Problem reading makeAct body: %v", err)
	}
	act := &Act{}
	err = json.Unmarshal(data, act)
	if err != nil {
		log.Fatalf("couldn't decode makeAct json: %v", err)
	}
	act.Player = playerID
	g := re.gc.Games[gameID]

	if !g.table.contains(playerID) {
		http.Error(w, "This player isn't seated at this game. Join game before trying to make a turn.", http.StatusUnauthorized)
	}
	err = g.controller.registerPlayerAct(*act)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (re RestExposer) makeUser(um *UserMap) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		credentials, err := parseAuthHeader(r.Header.Get("Authorization"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		username := credentials[0]
		if len(username) == 0 {
			http.Error(w, "Username cannot be zero-length; choose another username.", http.StatusBadRequest)
		}
		submitted := credentials[1]
		um.Lock()
		defer um.Unlock()
		playerID, ok := um.handles[username]
		if ok {
			if um.passwords[playerID] != submitted {
				http.Error(w, "Username already exists; choose another username.", http.StatusForbidden)
				return
			}
		} else {
			playerID = guid(createGuid())
			um.handles[username] = playerID
			um.passwords[playerID] = submitted
			if err != nil {
				log.Printf("Problem generating password: %v", err)
				http.Error(w, "There's been a server error. It's probably programming-related. We're sorry. WS-210.", http.StatusInternalServerError)
			}
		}
		w.WriteHeader(http.StatusCreated)
		enc := json.NewEncoder(w)
		err = enc.Encode(struct{ PlayerID string }{PlayerID: string(playerID)})
		if err != nil {
			log.Printf("Problem when encoding from makeUser: %v\n", err)
		}
		return
	}

}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	f, err := os.Create("profile1.prof")
	if err != nil {
		log.Fatal(err)
	}
	pprof.StartCPUProfile(f)
	time.AfterFunc(time.Duration(1*time.Minute), func() {
		pprof.StopCPUProfile()
		f.Close()
		fmt.Println("stopped profile...")
	})

	UserMap := NewUserMap()
	gc := NewGameController()
	re := ExposeByREST(gc)
	r := mux.NewRouter().StrictSlash(true)

	r.HandleFunc("/users/", re.makeUser(UserMap)).Methods("POST")

	//user := users.PathPrefix("/{UserID:[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}}").Subrouter()
	//user.HandleFunc("/", protector(UserMap, re.updateUser)).Methods("PUT")
	//user.HandleFunc("/", protector(UserMap, re.removeUser)).Methods("DELETE")

	r.HandleFunc("/games/", re.getGames).Methods("GET")
	r.HandleFunc("/games/", protector(UserMap, re.makeGame)).Methods("POST") // consider not allowing users to make games

	games := r.PathPrefix("/games").Subrouter()
	games.HandleFunc("/{GameID:[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}}/", protector(UserMap, re.getGameAuthenticated)).Methods("GET").Headers("Authorization", "")
	games.HandleFunc("/{GameID:[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}}/", re.getGame).Methods("GET")

	game := games.PathPrefix("/{GameID:[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}}").Subrouter()
	game.HandleFunc("/players/", re.getPlayers).Methods("GET")
	game.HandleFunc("/players/", protector(UserMap, re.playerJoinGame)).Methods("POST")

	players := game.PathPrefix("/players").Subrouter()
	players.HandleFunc("/{PlayerID:[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}}/", protector(UserMap, re.quitPlayer)).Methods("DELETE")

	player := players.PathPrefix("/{PlayerID:[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}}").Subrouter()
	player.HandleFunc("/acts/", protector(UserMap, re.makeAct)).Methods("POST")

	/*	r.HandleFunc("/blah/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hi")
	}) */

	http.Handle("/", r)
	http.ListenAndServe(":8080", nil)
	//http.ListenAndServeTLS(":10443", "cert.pem", "key.pem", nil)

}

type RestExposer struct {
	gc *GameController
}

func ExposeByREST(gc *GameController) (re RestExposer) {
	re = RestExposer{}
	re.gc = gc
	return re
}

type UserMap struct {
	handles   map[string]guid
	passwords map[guid]string
	sync.RWMutex
}

func NewUserMap() (um *UserMap) {
	um = new(UserMap)
	um.handles = make(map[string]guid)
	um.passwords = make(map[guid]string)
	return um
}

func joinGame(w http.ResponseWriter, g *Game, verifiedPlayerID guid) {
	p := NewPlayer(verifiedPlayerID)
	err := g.controller.enqueuePlayer(g, p)
	if err != nil {
		// TODO: make error type to marshal errors into for sending to clients
		http.Error(w, "This player has already joined this game.", http.StatusConflict)
		return
	}
	enc := json.NewEncoder(w)
	w.WriteHeader(http.StatusAccepted)
	pg := MakePublicGame(g)
	err = enc.Encode(pg)
	if err != nil {
		log.Printf("Error in joinGame when encoding public game: %v\n", err)
		return
	}
}
