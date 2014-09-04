package main

import (
	"testing"
	"strconv"
)

const passes = 100

func TestS4(t *testing.T) {
	// these should return a string of length four that parses as a
	// hexadecimal number
	strings := make(map[string]int)
	for str, i := s4(), 0; i < passes; str, i = s4(), i+1 {
		strings[str]++
		if l := len(str); l != 4 {
			t.Errorf("got len == %v, expected len == %v", l, 4)
		}
		if _, err := strconv.ParseInt(str, 16, 64); err != nil {
			t.Errorf("got err == %v, when parsing %v as hex", err, str)
		}
	}
	for hex, value := range strings {
		if value > 10 {
			t.Errorf("saw hex %v > 10 times; error likely", hex)
		}
	}
}

func TestDeal(t *testing.T) {
	
}