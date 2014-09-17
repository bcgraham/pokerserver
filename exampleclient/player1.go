package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
)

var host = flag.String("host", "localhost:8080", "host:port combination to use")
var user = flag.String("u", "brian", "username to use for duration of script")
var pass = flag.String("p", "password", "password to use for duration of script")

func request(method, path string) (*http.Request, error) {
	u := url.URL{Scheme: "http", Host: *host, Path: path}
	r, err := http.NewRequest(strings.ToUpper(method), u.String(), nil)
	if err != nil {
		return r, err
	}
	r.SetBasicAuth(*user, *pass)
	return r, nil
}

func init() {
	flag.Parse()
}

func main() {
	var r *http.Request
	var err error
	r, err = request("POST", "users")
	if err != nil {
		panic(err)
	}
	playerID, err = register()
	if err != nil {
		panic(err)
	}
	gameID := joinAnyGame()
	waitForTurn()

	r, err = request("POST", "users")
	if err != nil {
		fmt.Println("error")
	}
	c := http.Client{}
	resp, err := c.Do(r)
	if err != nil {
		log.Fatalf("problem with request: %v", err)
	}
	page := make([]byte, 1024)
	n, err := resp.Body.Read(page)
	if err != nil && err != io.EOF {
		log.Fatalf("problem reading response body: %v", err)
	}
	fmt.Printf("resp len: %v\n", n)
	fmt.Printf("resp code: %v\n", resp.StatusCode)
	fmt.Println(string(page[:n]))
	resp.Body.Close()
	r, err = http.NewRequest("POST", "http://10.0.1.2:8080/games/", nil)
	if err != nil {
		fmt.Println("error")
	}
	r.SetBasicAuth("brian", "password")
	c = http.Client{}
	resp, err = c.Do(r)
	if err != nil {
		log.Fatalf("problem with request: %v", err)
	}
	page = make([]byte, 1024)
	n, err = resp.Body.Read(page)
	if err != nil && err != io.EOF {
		log.Fatalf("problem reading response body: %v", err)
	}
	fmt.Printf("resp len: %v\n", n)
	fmt.Printf("resp code: %v\n", resp.StatusCode)
	fmt.Println(string(page[:n]))
	resp.Body.Close()
	r, err = http.NewRequest("POST", "http://10.0.1.2:8080/games/fbbbbb44-bc4f-c3f6-6519-d81bd1a66d8a/players/", nil)
	if err != nil {
		fmt.Println("error")
	}
	r.SetBasicAuth("brian", "password")
	c = http.Client{}
	resp, err = c.Do(r)
	if err != nil {
		log.Fatalf("problem with request: %v", err)
	}
	page = make([]byte, 1024)
	n, err = resp.Body.Read(page)
	if err != nil && err != io.EOF {
		log.Fatalf("problem reading response body: %v", err)
	}
	fmt.Printf("resp len: %v\n", n)
	fmt.Printf("resp code: %v\n", resp.StatusCode)
	fmt.Println(string(page[:n]))
	resp.Body.Close()
}

type turn struct {
	hole     []string
	toCall   money
	minRaise money
}

type money uint64
