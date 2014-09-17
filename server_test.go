package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

func Router() *mux.Router {
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

	return r
}

var r *mux.Router

func init() {
	r = Router()
	http.Handle("/", r)
}

func WAndReq(method, path, user, pass string) (*httptest.ResponseRecorder, *http.Request) {
	u := url.URL{Scheme: "http", Host: "localhost:8080", Path: path}
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(strings.ToUpper(method), u.String(), nil)
	req.SetBasicAuth(user, pass)
	return w, req
}

func WAndReqNoAuth(method, path string) (*httptest.ResponseRecorder, *http.Request) {
	u := url.URL{Scheme: "http", Host: "localhost:8080", Path: path}
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(strings.ToUpper(method), u.RequestURI(), nil)
	return w, req
}

func defaultRE() RestExposer {
	gc := NewGameController()
	re := RestExposer{gc: gc}
	return re
}

func defaults(method, path, user, pass string) (*httptest.ResponseRecorder, *http.Request, RestExposer, *UserMap) {
	w, req := WAndReq(method, path, user, pass)
	return w, req, defaultRE(), NewUserMap()
}

func parseResponse(w *httptest.ResponseRecorder) (code int, msg string, err error) {
	resp := make([]byte, 1024)
	n, err := w.Body.Read(resp)
	if err != nil {
		return 0, "", err
	}
	msg = strings.TrimSpace(string(resp[:n]))
	return w.Code, msg, nil
}

func extractGameID(w *httptest.ResponseRecorder) string {
	pg := new(PublicGame)
	json.Unmarshal(w.Body.Bytes(), pg)
	return pg.GameID
}

func TestMakeUserBadHeaderTooManyFields(t *testing.T) {
	re, UserMap := defaultRE(), NewUserMap()
	u := url.URL{Scheme: "http", Host: "localhost:8080", Path: "users"}
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(strings.ToUpper("POST"), u.RequestURI(), nil)
	req.Header.Add("foo", "bar")
	req.Header.Add("bar", "foo")
	re.makeUser(UserMap)(w, req)
	code, gotMsg, err := parseResponse(w)
	if err != nil {
		t.Errorf("Problem reading body: %v", err)
	}
	if code != http.StatusBadRequest {
		t.Errorf("unexpected status code: got %v, expected %v", code, http.StatusBadRequest)
	}
	expectedMsg := "You must provide properly-formatted credentials. Expecting HTTP Basic Authentication scheme and base64-encoded username:password pair."
	if gotMsg != expectedMsg {
		t.Errorf("unexpected error message: got %v, expected %v", gotMsg, expectedMsg)
	}
}

func TestMakeUserBadHeaderNotBasicAuth(t *testing.T) {
	re, UserMap := defaultRE(), NewUserMap()
	u := url.URL{Scheme: "http", Host: "localhost:8080", Path: "users"}
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(strings.ToUpper("POST"), u.RequestURI(), nil)
	var b bytes.Buffer
	enc := base64.NewEncoder(base64.StdEncoding, &b)
	enc.Write([]byte("brian:password"))
	req.Header.Add("Authorization", "NotBasic"+" "+b.String())
	re.makeUser(UserMap)(w, req)
	code, gotMsg, err := parseResponse(w)
	if err != nil {
		t.Errorf("Problem reading body: %v", err)
	}
	if code != http.StatusBadRequest {
		t.Errorf("unexpected status code: got %v, expected %v", code, http.StatusBadRequest)
	}
	expectedMsg := "You must provide properly-formatted credentials. You either did not use the HTTP Basic Authentication scheme or did not provide a username/password field. We have not yet checked whether it is a valid base64 encoding; we have only checked whether the field was present at all."
	if gotMsg != expectedMsg {
		t.Errorf("unexpected error message: got %v, expected %v", gotMsg, expectedMsg)
	}
}

func TestMakeUserBadHeaderNotBase64Encoded(t *testing.T) {
	re, UserMap := defaultRE(), NewUserMap()
	u := url.URL{Scheme: "http", Host: "localhost:8080", Path: "users"}
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(strings.ToUpper("POST"), u.RequestURI(), nil)
	req.Header.Add("Authorization", "Basic brian:password")
	re.makeUser(UserMap)(w, req)
	code, gotMsg, err := parseResponse(w)
	if err != nil {
		t.Errorf("Problem reading body: %v", err)
	}
	if code != http.StatusBadRequest {
		t.Errorf("unexpected status code: got %v, expected %v", code, http.StatusBadRequest)
	}
	expectedMsg := "You must provide properly-formatted credentials. We could not decode your base64-encoded username/password combination."
	if gotMsg != expectedMsg {
		t.Errorf("unexpected error message: got %v, expected %v", gotMsg, expectedMsg)
	}
}

func TestMakeUserBadUsername(t *testing.T) {
	w, req, re, UserMap := defaults("POST", "users", "", "password")
	re.makeUser(UserMap)(w, req)
	code, gotMsg, err := parseResponse(w)
	if err != nil {
		t.Errorf("Problem reading body: %v", err)
	}
	if code != http.StatusBadRequest {
		t.Errorf("unexpected status code: got %v, expected %v", code, http.StatusBadRequest)
	}
	expectedMsg := "Username cannot be zero-length; choose another username."
	if gotMsg != expectedMsg {
		t.Errorf("unexpected error message: got %v, expected %v", gotMsg, expectedMsg)
	}
}

func TestMakeGameNoAuth(t *testing.T) {
	re := defaultRE()
	UserMap := NewUserMap()
	w, req := WAndReqNoAuth("POST", "games")
	protector(UserMap, re.makeGame)(w, req)
	code, gotMsg, err := parseResponse(w)
	if err != nil {
		t.Errorf("Problem reading body: %v", err)
	}
	if code != http.StatusBadRequest {
		t.Errorf("unexpected status code: got %v, expected %v", w.Code, http.StatusBadRequest)
	}
	expectedMsg := "You must provide properly-formatted credentials. Expecting HTTP Basic Authentication scheme and base64-encoded username:password pair."
	if gotMsg != expectedMsg {
		t.Errorf("unexpected error message: got %v, expected %v", gotMsg, expectedMsg)
	}
}

func TestMakeGameWrongAuth(t *testing.T) {
	w, req, re, UserMap := defaults("POST", "users", "brian", "password")
	re.makeUser(UserMap)(w, req)
	if w.Code != http.StatusCreated {
		io.Copy(os.Stdout, w.Body)
		t.Errorf("could not make user: got %v, expected %v", w.Code, http.StatusCreated)
	}
	w, req = WAndReq("POST", "games", "brian", "wrongpassword")
	protector(UserMap, re.makeGame)(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("making a game without making a user account first; got %v, expected %v", w.Code, http.StatusUnauthorized)
	}
}

func TestMakeGame(t *testing.T) {
	w, req, re, UserMap := defaults("POST", "users", "brian", "password")
	re.makeUser(UserMap)(w, req)
	w, req = WAndReq("POST", "games", "brian", "password")
	protector(UserMap, re.makeGame)(w, req)
	if w.Code != http.StatusCreated {
		t.Errorf("making a game without making a user account first; got %v, expected %v", w.Code, http.StatusCreated)
	}
}

func TestPlayerJoinNonexistentGameNoAuth(t *testing.T) {
	w, req, re, UserMap := defaults("POST", "games/fbbbbb44-bc4f-c3f6-6519-d81bd1a66d8a/players", "brian", "password")
	protector(UserMap, re.playerJoinGame(UserMap))(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("making a game without making a user account first; got %v, expected %v", w.Code, http.StatusUnauthorized)
	}
}

func TestPlayerJoinNonexistentGameWrongAuth(t *testing.T) {
	w, req, re, UserMap := defaults("POST", "users", "brian", "password")
	re.makeUser(UserMap)(w, req)
	w, req = WAndReq("POST", "games/fbbbbb44-bc4f-c3f6-6519-d81bd1a66d8a/players", "brian", "wrongpassword")
	protector(UserMap, re.playerJoinGame(UserMap))(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("making a game without making a user account first; got %v, expected %v", w.Code, http.StatusUnauthorized)
	}
}

func TestPlayerJoinNonexistentGame(t *testing.T) {
	w, req, re, UserMap := defaults("POST", "users", "brian", "password")
	re.makeUser(UserMap)(w, req)
	w, req = WAndReq("POST", "games/fbbbbb44-bc4f-c3f6-6519-d81bd1a66d8a/players", "brian", "password")
	protector(UserMap, re.playerJoinGame(UserMap))(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("making a game without making a user account first; got %v, expected %v", w.Code, http.StatusNotFound)
	}
}

func TestPlayerJoinGameWithoutAuthenticating(t *testing.T) {
	w, req, re, UserMap := defaults("POST", "users", "brian", "password")
	re.makeUser(UserMap)(w, req)
	w, req = WAndReq("POST", "games", "brian", "password")
	protector(UserMap, re.makeGame)(w, req)
	w, req = WAndReqNoAuth("POST", "games/fbbbbb44-bc4f-c3f6-6519-d81bd1a66d8a/players")
	protector(UserMap, re.playerJoinGame(UserMap))(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("making a game without making a user account first; got %v, expected %v", w.Code, http.StatusBadRequest)
	}
}

func TestPlayerJoinGameBadAuth(t *testing.T) {
	w, req, re, UserMap := defaults("POST", "users", "brian", "password")
	re.makeUser(UserMap)(w, req)
	w, req = WAndReq("POST", "games", "brian", "password")
	protector(UserMap, re.makeGame)(w, req)
	w, req = WAndReq("POST", "games/fbbbbb44-bc4f-c3f6-6519-d81bd1a66d8a/players", "brian", "wrongpassword")
	protector(UserMap, re.playerJoinGame(UserMap))(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("making a game without making a user account first; got %v, expected %v", w.Code, http.StatusUnauthorized)
	}
}

func TestPlayerJoinGame(t *testing.T) {
	w, req := WAndReq("POST", "users", "brian", "password")
	r = Router()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Errorf("got code %v, expected code %v", w.Code, http.StatusCreated)
	}
	w, req = WAndReq("POST", "games", "brian", "password")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Errorf("got code %v, expected code %v", w.Code, http.StatusCreated)
	}
	gameID := extractGameID(w)
	w, req = WAndReq("POST", "games/"+gameID+"/players", "brian", "password")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Errorf("making a game without making a user account first; got %v, expected %v", w.Code, http.StatusCreated)
	}
}

func TestTwoPlayersJoinGame(t *testing.T) {
	re, UserMap := defaultRE(), NewUserMap()
	w, req := WAndReq("POST", "users", "brian", "password")
	re.makeUser(UserMap)(w, req)
	if w.Code != http.StatusCreated {
		t.Errorf("making player correctly; got %v, expected %v", w.Code, http.StatusCreated)
	}
	w, req = WAndReq("POST", "users", "jake", "password")
	re.makeUser(UserMap)(w, req)
	if w.Code != http.StatusCreated {
		t.Errorf("making player correctly; got %v, expected %v", w.Code, http.StatusCreated)
	}
	w, req = WAndReq("POST", "games", "brian", "password")
	protector(UserMap, re.makeGame)(w, req)
	if w.Code != http.StatusCreated {
		t.Errorf("making game correctly; got %v, expected %v", w.Code, http.StatusCreated)
	}
	gameID := extractGameID(w)
	w, req = WAndReq("POST", "games/"+gameID+"/players", "brian", "password")
	protector(UserMap, re.playerJoinGame(UserMap))(w, req)
	if w.Code != http.StatusCreated {
		t.Errorf("player joining game correctly; got %v, expected %v", w.Code, http.StatusCreated)
	}
	w, req = WAndReq("POST", "games/"+gameID+"/players", "jake", "password")
	protector(UserMap, re.playerJoinGame(UserMap))(w, req)
	if w.Code != http.StatusCreated {
		t.Errorf("player joining game correctly; got %v, expected %v", w.Code, http.StatusCreated)
	}
}

func TestGetGames(t *testing.T) {
	re, UserMap := defaultRE(), NewUserMap()
	w, req := WAndReq("POST", "users", "brian", "password")
	re.makeUser(UserMap)(w, req)
	w, req = WAndReq("POST", "games", "brian", "password")
	protector(UserMap, re.makeGame)(w, req)
	re.getGames(w, req)
	fmt.Println("getting games...")
	fmt.Println(w.Body.String())
}
