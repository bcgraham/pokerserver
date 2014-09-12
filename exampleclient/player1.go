package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
)

var username = flag.String("u", "brian", "username to log into poker server with")
var password = flag.String("p", "password", "username to log into poker server with")
var host = flag.String("h", "10.0.5.234:8080", "ip:port string")

func main() {
	flag.Parse()
	u := url.URL{Scheme: "http", Host: *host}
	u.Path = "users"
	r, err := http.NewRequest("POST", u.RequestURI(), nil)
	if err != nil {
		fmt.Println("error")
	}
	r.SetBasicAuth("brian", "password")
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
	u.Path = "games"
	r, err = http.NewRequest("POST", u.RequestURI(), nil)
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
	r, err = http.NewRequest("POST", "http://10.0.5.234:8080/games/fbbbbb44-bc4f-c3f6-6519-d81bd1a66d8a/players/", nil)
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
