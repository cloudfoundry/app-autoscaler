'use strict';

var express = require('express');
var fs = require('fs');
var path = require('path');
var bodyParser = require('body-parser');
var api = require(path.join(__dirname, './api/api.js'));

var settings = JSON.parse(
  fs.readFileSync(path.join(__dirname, '../config/settings.json'), 'utf8'));
var port = process.env.PORT || settings.port;

var app = express();
app.use(bodyParser.urlencoded({ extended: true }));
app.use(bodyParser.json());

var router = express.Router();
var serviceBrokerApi = new api();
router.get('/catalog', function(req, res) {
  res.json(serviceBrokerApi.getCatalog());
});

app.use('/v2', router);
var server = app.listen(port);
module.exports = server;