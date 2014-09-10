package main

import (
	"fmt"
	"strings"
	"testing"
)

//Hands - No Ties
var sf1 Hand = strings.Fields("TD JD QD KD AD") // Straight Flush
var sf2 Hand = strings.Fields("6C 7C 8C 9C TC") // Straight Flush
var fk1 Hand = strings.Fields("9D 9H 9S 9C 7D") // Four of a Kind
var fk2 Hand = strings.Fields("9D 9H 9S 9C 3D") // Four of a Kind
var fk3 Hand = strings.Fields("4D 4H 4S 4C AC") // Four of a Kind
var fh1 Hand = strings.Fields("TD TC TH 7C 7D") // Full House
var fh2 Hand = strings.Fields("TD TC TH 4C 4D") // Full House
var fh3 Hand = strings.Fields("9D 9C 9H 7C 7D") // Full House
var fl1 Hand = strings.Fields("TD AD 6D 7D 9D") // Flush
var fl2 Hand = strings.Fields("TC AC 6C 4C 9C") // Flush
var fl3 Hand = strings.Fields("TH KH 6H 7H 9H") // Flush
var st1 Hand = strings.Fields("TC JC QC KS AS") // 10-A straight
var st2 Hand = strings.Fields("2C 3C 4C 5S 6S") // 2-6 straight
var st3 Hand = strings.Fields("AS 2S 3S 4S 5C") // A-5 straight
var tk1 Hand = strings.Fields("TS TD TC JC 2S") // Three of a kind
var tk2 Hand = strings.Fields("5S 5D 5C 9C 6S") // Three of a kind
var tp1 Hand = strings.Fields("5C 5H TH TC 2S") // two pair
var tp2 Hand = strings.Fields("5S 5D 9H 9C AS") // two pair
var tp3 Hand = strings.Fields("4S 4D 2H 2C KS") // two pair
var pr1 Hand = strings.Fields("2S 3C JD 4S 4H") // pair
var pr2 Hand = strings.Fields("2S 3C AD TS TC") // pair
var pr3 Hand = strings.Fields("2S 3C KD QS QC") // pair
var ah1 Hand = strings.Fields("AS 2S 3S 4S 6C") // A high
var sh1 Hand = strings.Fields("2S 3S 4S 6C 8D") // 8 high
var sh2 Hand = strings.Fields("2S 3S 4S 6C 7D") // 7 high

//Sorted highest ranking to lowest ranking
var allHands []Hand = []Hand{sf1, sf2, fk1, fk2, fk3, fh1, fh2, fh3, fl1, fl2, fl3, st1, st2, st3, tk1, tk2, tp1, tp2, tp3, ah1, sh1, sh2}

//Test
func TestFindWinningHand(t *testing.T) {
	//no Ties
	//Run through every 3-hand combination of hands above
	//Find the winning hands. In each case should be the hand with the lower index in allHands
	fmt.Println("==== Three player game ====")
	for i := 0; i < len(allHands)-2; i++ {
		for q := i + 1; q < len(allHands)-1; q++ {
			for n := q + 1; n < len(allHands); n++ {
				testGame := append(make([]Hand, 0), allHands[i], allHands[q], allHands[n])
				winners := findWinningHands(testGame)
				if len(winners) != 1 {
					t.Errorf("too many winners")
				}
				if !areHandsEq(winners[0], allHands[i]) {
					t.Errorf("%v is winner, should be %v",
						winners[0], allHands[i])
				} else {
					fmt.Println(testGame, " --> ", winners)
				}
			}
		}
	}

	fmt.Println("==== Single player game ====")
	testGame := append(make([]Hand, 0), allHands[10])
	winners := findWinningHands(testGame)
	if len(winners) != 1 && !areHandsEq(winners[0], allHands[10]) {
		t.Errorf("Incorrect number of winners. Incorrect winner")
	} else {
		fmt.Println(testGame, " --> ", winners)
	}

	//Test Tie manually
	testGame = append(make([]Hand, 0), allHands[10], allHands[8], allHands[8])
	winners = findWinningHands(testGame)
	if len(winners) != 2 && !areHandsEq(winners[0], allHands[8]) && !areHandsEq(winners[0], allHands[8]) {
		t.Errorf("%v are the winner, should be %v tied. Winners len=%v",
			winners, allHands[3], len(winners))
	}
}

// func TestRawRanksAndSuits(t *testing.T) {
//     //Test legimitate conversion
//     var fl1 Hand = strings.Fields("TD AD 6D 7D 9D") // Flush
//     var sh1 Hand = strings.Fields("2S 3S 4S 6C 8D") // 8 high
//     var tk1 Hand = strings.Fields("TS TD TC JC 2S") // Three of a kind

//     //Test failure on invalid suits/ranks
//     var invalid1 Hand = strings.Fields("MS VD TC JC 2S")
//     var invalid2 Hand = strings.Fields("0S TD TC JC 2S")

// }
