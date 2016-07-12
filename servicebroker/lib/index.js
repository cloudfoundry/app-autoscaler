'use strict';

var express = require('express');
var basicAuth = require('basic-auth');
var bodyParser = require('body-parser');

var fs = require('fs');
var path = require('path');
var logger = require(path.join(__dirname, './logger/logger.js'));
var settings = require(path.join(__dirname, './config/setting.js'))((JSON.parse(
  fs.readFileSync(path.join(__dirname, '../config/settings.json'), 'utf8'))));

var port = process.env.PORT || settings.port;

var app = express();
var auth = function (req, res, next) {
  function unauthorized(res) {
    res.set('WWW-Authenticate', 'Basic realm="serviceBrokerAuth"');
    return res.sendStatus(401);
  };

  var user = basicAuth(req);
  if (!user || !user.name || !user.pass) {
    return unauthorized(res);
  };

  if (user.name === settings.username && user.pass === settings.password) {
    return next();
  } else {
    return unauthorized(res);
  };
  next();
};

//define the sequence of middleware
app.use(auth);
app.use(bodyParser.urlencoded({ extended: true }));
app.use(bodyParser.json());
require('./routes')(app, settings);


var server = app.listen(port, function () {
    var port = server.address().port;
    logger.info('Service broker app is running and listening at port ' + port);
});
module.exports = server;