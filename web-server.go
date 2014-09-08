package controller

import (
	"encoding/json"
	"log"
	"net/http"

	mux "github.com/gorilla/mux"
)

func (a *authenticator) authenticate(token guid, requestor guid) (newid guid, valid bool) {
	if player, ok := a[token]; !ok || player != requestor {
		return guid(), false
	}
	delete(a, token)
	newid = guid()
	a[newid] = p
	return newid, true
}

func (a *authenticator) register(p *Player) (token guid) {
	token = guid()
	a[token] = p
	return token
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
	gameID := guid()
	re.Games[gameID] = NewGame()
	enc := json.NewEncoder(w)
	enc.Encode(re.Games[gameID])
}

func (re RestExposer) getGame(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if gameID, ok := vars["GameID"]; !ok {
		log.Fatalf("no such game: %v", vars["GameID"])
	}
	enc := json.NewEncoder(w)
	enc.Encode(re.Games[vars["GameID"]])
}

func (re RestExposer) getPlayers(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if g, ok := re.Games[vars["GameID"]]; !ok {
		// TODO: handle errors
		log.Fatalf("no such game: %v", vars["GameID"])
	}
	enc := json.NewEncoder(w)
	enc.Encode(g.Players)
}

func (re RestExposer) makePlayer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if g, ok := re.Games[vars["GameID"]]; !ok {
		// TODO: handle errors
		log.Fatalf("no such game: %v", vars["GameID"])
	}
	p := newPlayer(vars["player"]) // TBD
	token := re.authenticator.register(p)
	// send back authentication token and other info
}

func (re RestExposer) quitPlayer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	err := r.ParseForm()
	if g, ok := re.Games[vars["GameID"]]; !ok {
		// TODO: handle errors
		log.Fatalf("no such game: %v", vars["GameID"])
	}
	if player, ok := g.Players[vars["PlayerID"]]; !ok {
		// TODO: handle errors
		log.Fatalf("no such player: %v", vars["PlayerID"])
	}
	token := r.Form["token"]
	newtoken, valid := re.auth.authenticate(token, player)
	if !valid {
		log.Fatal("invalid token")
	}
	g.Players.Kill(player) // does not exist
	//send confirmation of kill
}

func main() {
	re := ExposeByREST(gc)

	gc := NewGameController()
	r := mux.NewRouter()

	games := r.PathPrefix("/games").Subrouter()
	games.HandleFunc("/", re.getGames).Methods("GET")
	games.HandleFunc("/", re.makeGame).Methods("POST")

	game := games.PathPrefix("/{GameID:[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}}").Subrouter()
	game.HandleFunc("/", re.getGame).Methods("GET")

	players := game.PathPrefix("/players").Subrouter()
	players.HandleFunc("/", re.getPlayers).Methods("GET")
	players.HandleFunc("/", re.makePlayer).Methods("POST")

	player := players.PathPrefix("/{PlayerID:[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}}").Subrouter()
	player.HandleFunc("/", re.quitPlayer).Methods("DELETE")

}

type RestExposer struct {
	*GameController
}

func ExposeByREST(gc *GameController) (re RestExposer) {
	re = new(RestExposer)
	re.GameController = gc
	return re
}
