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

	"code.google.com/p/go.crypto/bcrypt"

	mux "github.com/gorilla/mux"
)

const COST = 10

func parseAuthHeader(header string) (credentials []string, err error) {
	authinfo := strings.Fields(header)
	if len(authinfo) != 2 || authinfo[0] != "Basic" {
		return credentials, errors.New("You must provide properly-formatted credentials. You either did not use the HTTP Basic Authentication scheme or did not provide a username/password field. We have not yet checked whether it is a valid base64 encoding; we have only checked whether the field was present at all.")
	}
	scheme := authinfo[0]
	raw, err := base64.StdEncoding.DecodeString(authinfo[1])
	if err != nil {
		return credentials, errors.New("You must provide properly-formatted credentials. We could not decode your base64-encoded username/password combination.")
	}
	credentials = strings.SplitN(raw, ":", 2)
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
		playerID, ok := vars["PlayerID"]
		if ok {
			if um.handles[username] != playerID {
				http.Error(w, "You cannot take an action for a player without authenticating as that player.", http.StatusUnauthorized)
			}
		}
		username := credentials[0]
		password := []byte(credentials[1])
		hashed, ok := um[username]
		if !ok {
			http.Error(w, "Invalid credentials.", http.StatusUnauthorized)
			return
		}
		err = bcrypt.CompareHashAndPassword(hashedPassword, password)
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
	go g.run()
	// actually spin up new game here
	enc := json.NewEncoder(w)
	enc.Encode(re.gc.Games[gameID])
}

func (re RestExposer) getGame(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if gameID, ok := vars["GameID"]; !ok {
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

func (re RestExposer) playerJoinGame(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	g, ok := re.gc.Games[guid(vars["GameID"])]
	if !ok {
		// TODO: handle errors
		log.Fatalf("no such game: %v", vars["GameID"])
	}
	if g.controller.playerInGame(vars["PlayerID"]) {

	}
	p, ok := g.table.players
	p := newPlayer(vars["player"]) // TBD
}

func (re RestExposer) quitPlayer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	err := r.ParseForm()
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
	limited := io.LimitedReader{R: r.Body, N: 1048576} // meg of json ought to be enough
	data, err := ioutil.ReadAll(limited)
	if err != nil {
		log.Fatalf("Problem reading makeAct body: %v", err)
	}
	act := &Act{}
	err = json.Unmarshal(data, resp)
	if err != nil {
		log.Fatalf("couldn't decode makeAct json: %v", err)
	}
	g := re.gc.Games[gameID]
	g.controller.toGame <- *act
	w.WriteHeader(http.StatusOK)
}

func (re RestExposer) makeUser(um *UserMap) func(http.ResponseWriter, *http.Request) {
	return func(http.ResponseWriter, *http.Request) {
		credentials, err := parseAuthHeader(r.Header.Get("Authorization"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		username := credentials[0]
		password := []byte(credentials[1])
		playerID := createGuid()
		um.handles[username] = playerID
		um.passwords[playerID] = bcrypt.GenerateFromPassword(password, COST)

	}

}

func main() {
	UserMap := NewUserMap()
	gc := NewGameController()
	re := ExposeByREST(gc)
	r := mux.NewRouter()

	users := r.PathPrefix("/users").Subrouter()
	users.HandleFunc("/", re.makeUser).Methods("POST")

	user := users.PathPrefix("/{UserID:[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}}").Subrouter()
	user.HandleFunc("/", protector(UserMap, re.updateUser)).Methods("PUT")
	user.HandleFunc("/", protector(UserMap, re.removeUser)).Methods("DELETE")

	games := r.PathPrefix("/games").Subrouter()
	games.HandleFunc("/", re.getGames).Methods("GET")
	games.HandleFunc("/", protector(UserMap, re.makeGame)).Methods("POST") // consider not allowing users to make games

	game := games.PathPrefix("/{GameID:[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}}").Subrouter()
	game.HandleFunc("/", re.getGame).Methods("GET")

	players := game.PathPrefix("/players").Subrouter()
	players.HandleFunc("/", re.getPlayers).Methods("GET")
	players.HandleFunc("/", protector(UserMap, re.playerJoinGame)).Methods("POST")

	player := players.PathPrefix("/{PlayerID:[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}}").Subrouter()
	player.HandleFunc("/", protector(UserMap, re.quitPlayer)).Methods("DELETE")

	acts := player.PathPrefix("/acts").Subrouter()
	acts.HandleFunc("/", protector(UserMap, re.makeAct)).Methods("POST")

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
}

func NewUserMap() (um *UserMap) {
	um = new(UserMap)
	um.handles = make(map[string]guid)
	um.passwords = make(map[guid][]byte)
	return um
}
