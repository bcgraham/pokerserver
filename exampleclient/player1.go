package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
)

func main() {
	r, err := http.NewRequest("POST", "http://localhost:8080/users/", nil)
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
	r, err = http.NewRequest("POST", "http://localhost:8080/games/", nil)
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
	r, err = http.NewRequest("POST", "http://localhost:8080/games/fbbbbb44-bc4f-c3f6-6519-d81bd1a66d8a/players/", nil)
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
