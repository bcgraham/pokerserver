// var json = {
//       "gameID":"fbbbbb44-bc4f-c3f6-6519-d81bd1a66d8a",
//       "table":[
//          {
//             "playerID":"2fd318e4-0947-64b9-8b63-14f5a5824b8b",
//             "handle":"BOB",
//             "state":"active",
//             "wealth":15002,
//             "bet_so_far":0,
//             "small blind":true
//          },
//          {
//             "playerID":"0dd7d693-06f9-b5ba-694d-f8779579fa8a",
//             "handle":"",
//             "state":"active",
//             "wealth":4006,
//             "bet_so_far":0,
//             "small blind":false
//          }
//       ],
//       "turn":{
//          "playerID":"2fd318e4-0947-64b9-8b63-14f5a5824b8b",
//          "bet_so_far":0,
//          "bet_to_player":0,
//          "minimum_raise":0,
//          "expiry":"2014-09-24 15:14:53.021303798 -0400 EDT"
//       },
//       "cards":{
//          "hole":[
//             "AC",
//             "TD"
//          ],
//          "flop":[
//             "2S",
//             "3S",
//             "JS"
//          ],
//          "turn":[
//             "4D"
//          ],
//          "river":[
//             "KD"
//          ]
//       },
//       "pots":[
//          {
//             "size":60,
//             "players":[
//                "8c479795-adcb-a3c1-bebd-4bdf54016f4a",
//                "2fd318e4-0947-64b9-8b63-14f5a5824b8b",
//                "0dd7d693-06f9-b5ba-694d-f8779579fa8a"
//             ]
//          },
//          {
//             "size":0,
//             "players":[
//                "2fd318e4-0947-64b9-8b63-14f5a5824b8b",
//                "0dd7d693-06f9-b5ba-694d-f8779579fa8a"
//             ]
//          },
//          {
//             "size":0,
//             "players":[
//                "2fd318e4-0947-64b9-8b63-14f5a5824b8b",
//                "0dd7d693-06f9-b5ba-694d-f8779579fa8a"
//             ]
//          }
//       ]
//    }


function getJSON(){
    $.ajax({
       beforeSend: function(xhr) { xhr.setRequestHeader("Authorization", "Basic " + btoa("jake" + ":" + "password")); },
       url: "http://localhost:8080/games/fbbbbb44-bc4f-c3f6-6519-d81bd1a66d8a/",
       jsonp: function( data ) {
        $("#handle").html(data.table[0].handle)
        $("#wealth").html(data.table[0].wealth)
        $("#bet").html(data.table[0].bet_so_far)    
        $("#hole1 > img").attr("src", data.cards.hole[0] + ".png")
        $("#hole2 > img").attr("src", data.cards.hole[1] + ".png")
        $("#flop1 > img").attr("src", data.cards.flop[0] + ".png")
        $("#flop2 > img").attr("src", data.cards.flop[1] + ".png")
        $("#flop3 > img").attr("src", data.cards.flop[2] + ".png")
        $("#turn > img").attr("src", data.cards.turn[0] + ".png")
        $("#river > img").attr("src", data.cards.river[0] + ".png")
       }
    });
}

$(function() {
    getJSON()
});