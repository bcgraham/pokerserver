package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"

	"code.google.com/p/go.crypto/bcrypt"

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

func protector(um *UserMap, restricted http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		credentials, err := parseAuthHeader(r.Header.Get("Authorization"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		vars := mux.Vars(r)
		playerIDstr, ok := vars["PlayerID"]
		playerID := guid(playerIDstr)
		username := credentials[0]
		if ok {
			if um.handles[username] != playerID {
				http.Error(w, "You cannot take an action for a player without authenticating as that player.", http.StatusUnauthorized)
			}
		}
		password := []byte(credentials[1])
		if playerID == "" {
			playerID = um.handles[username]
		}
		hash, ok := um.passwords[playerID]
		if !ok {
			http.Error(w, "Invalid credentials.", http.StatusUnauthorized)
			return
		}
		err = bcrypt.CompareHashAndPassword(hash, password)
		if err != nil {
			http.Error(w, "Invalid credentials.", http.StatusUnauthorized)
			return
		}
		// we have successfully authenticated the user; she is who she says she is.
		restricted(w, r)
	}
}

func (re RestExposer) getGames(w http.ResponseWriter, r *http.Request) {
	enc := json.NewEncoder(w)
	gs := make([]*Game, 0)
	for _, g := range re.gc.Games {
		gs = append(gs, g)
	}
	enc.Encode(gs)
}

func (re RestExposer) makeGame(w http.ResponseWriter, r *http.Request) {
	gameID := guid(createGuid())
	g := NewGame(re.gc)
	re.gc.Games[gameID] = g
	g.gameID = gameID
	go g.run()
	enc := json.NewEncoder(w)
	w.WriteHeader(http.StatusCreated)
	enc.Encode(gameID)
}

func (re RestExposer) getGame(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if _, ok := vars["GameID"]; !ok {
		log.Fatalf("no such game: %v", vars["GameID"])
	}
	enc := json.NewEncoder(w)
	enc.Encode(re.gc.Games[guid(vars["GameID"])])
}

func (re RestExposer) getPlayers(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	g, ok := re.gc.Games[guid(vars["GameID"])]
	if !ok {
		// TODO: handle errors
		log.Fatalf("no such game: %v", vars["GameID"])
	}
	enc := json.NewEncoder(w)
	enc.Encode(g.table.players)
}

func (re RestExposer) playerJoinGame(um *UserMap) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		g, ok := re.gc.Games[guid(vars["GameID"])]
		credentials, err := parseAuthHeader(r.Header.Get("Authorization"))
		username := credentials[0]
		if err != nil {
			log.Fatalf("playerJoinGame: problem parsing header to get player ID: %v", err)
		}
		if !ok {
			// TODO: handle errors
			log.Fatalf("playerJoinGame: no such game: %v", vars["GameID"])
		}
		p := NewPlayer(um.handles[username])
		err = g.controller.enqueuePlayer(g, p)
		if err != nil {
			// TODO: make error type to marshal errors into for sending to clients
			log.Fatalf("player can't join: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
		// need to write actual data or header sufficient?
	}
}

func (re RestExposer) quitPlayer(w http.ResponseWriter, r *http.Request) {
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

func (re RestExposer) makeAct(w http.ResponseWriter, r *http.Request) {
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
	act.player = playerID
	g := re.gc.Games[gameID]

	if !g.table.contains(playerID) {
		http.Error(w, "This player isn't seated at this game. Join game before trying to make a turn.", http.StatusUnauthorized)
	}
	err = g.controller.registerPlayerAct(*act)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (re RestExposer) makeUser(um *UserMap) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		credentials, err := parseAuthHeader(r.Header.Get("Authorization"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		username := credentials[0]
		password := []byte(credentials[1])
		um.Lock()
		defer um.Unlock()
		if playerID, ok := um.handles[username]; ok {
			if err := bcrypt.CompareHashAndPassword(um.passwords[playerID], password); err != nil {
				http.Error(w, "Username already exists; choose another username.", http.StatusForbidden)
				return
			}
			w.WriteHeader(http.StatusCreated)
			// consider using http.StatusNoContent to differentiate between just-made users and already-existing users
			return
		}
		playerID := guid(createGuid())
		um.handles[username] = playerID
		um.passwords[playerID], err = bcrypt.GenerateFromPassword(password, COST)
		if err != nil {
			log.Fatalf("Problem generating password: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
		return
	}

}

func main() {
	UserMap := NewUserMap()
	gc := NewGameController()
	re := ExposeByREST(gc)
	r := mux.NewRouter()

	users := r.PathPrefix("/users").Subrouter()
	users.HandleFunc("/", re.makeUser(UserMap)).Methods("POST")

	//user := users.PathPrefix("/{UserID:[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}}").Subrouter()
	//user.HandleFunc("/", protector(UserMap, re.updateUser)).Methods("PUT")
	//user.HandleFunc("/", protector(UserMap, re.removeUser)).Methods("DELETE")

	games := r.PathPrefix("/games").Subrouter()
	games.HandleFunc("/", re.getGames).Methods("GET")
	games.HandleFunc("/", protector(UserMap, re.makeGame)).Methods("POST") // consider not allowing users to make games

	game := games.PathPrefix("/{GameID:[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}}").Subrouter()
	game.HandleFunc("/", re.getGame).Methods("GET")

	players := game.PathPrefix("/players").Subrouter()
	players.HandleFunc("/", re.getPlayers).Methods("GET")
	players.HandleFunc("/", protector(UserMap, re.playerJoinGame(UserMap))).Methods("POST")

	player := players.PathPrefix("/{PlayerID:[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}}").Subrouter()
	player.HandleFunc("/", protector(UserMap, re.quitPlayer)).Methods("DELETE")

	acts := player.PathPrefix("/acts").Subrouter()
	acts.HandleFunc("/", protector(UserMap, re.makeAct)).Methods("POST")

	http.Handle("/", r)
	http.ListenAndServe(":8080", nil)

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
	passwords map[guid][]byte
	sync.RWMutex
}

func NewUserMap() (um *UserMap) {
	um = new(UserMap)
	um.handles = make(map[string]guid)
	um.passwords = make(map[guid][]byte)
	return um
}
