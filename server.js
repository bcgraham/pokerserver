// set up =================================
// get all the tools we need
var express  = require('express');
var app      = express();
var port     = process.env.PORT || 8080;
var bodyParser      = require('body-parser');

// set up our express application
app.use(bodyParser()); // get information from html forms
app.use(express.static('public')); // serve static files out of public
app.set('view engine', 'ejs'); // set up ejs for templating

// routes ===================================
// require('./app/routes.js'); // load our routes
app.get('/', function(req, res) {
        res.send('{"cats":"dogs"}'); // load the index.ejs file
    });

// launch ===================================
app.listen(port);
console.log('Listening on port ' + port);