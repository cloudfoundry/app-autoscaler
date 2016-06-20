'use strict';

var express = require('express');
var basicAuth = require('basic-auth');
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

var auth = function (req, res, next) {
  
  function unauthorized(res) {
    res.set('WWW-Authenticate', 'Basic realm=Authorization Required');
    return res.sendStatus(401);
  };

  var user = basicAuth(req);
  if (!user || !user.name || !user.pass) {
    return unauthorized(res);
  };

  if (user.name === settings.user && user.pass === settings.password) {
    return next();
  } else {
    return unauthorized(res);
  };
  next();
};

app.use(auth);

var router = express.Router();
var serviceBrokerApi = new api();
router.get('/catalog', function(req, res) {
  res.json(serviceBrokerApi.getCatalog());
});

app.use('/v2', router);
var server = app.listen(port);
module.exports = server;