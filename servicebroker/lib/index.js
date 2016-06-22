'use strict';

var express = require('express');
var basicAuth = require('basic-auth');
var fs = require('fs');
var path = require('path');
var bodyParser = require('body-parser');
var async = require('asyncawait/async');
var await = require('asyncawait/await');
var API = require(path.join(__dirname, './api/api.js'));

var settings = JSON.parse(
  fs.readFileSync(path.join(__dirname, '../config/settings.json'), 'utf8'));
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

  if (user.name === settings.user && user.pass === settings.password) {
    return next();
  } else {
    return unauthorized(res);
  };
  next();
};

var router = express.Router();
var serviceBrokerApi = new API();

router.get('/catalog', function(req, res) {
  res.json(serviceBrokerApi.getCatalog());
});

router.put('/service_instances/:serviceId', async(function(req, res) {
  var serviceId = req.params.serviceId;
  var orgId = req.body.organization_guid;
  var spaceId = req.body.space_guid
  var result = await (serviceBrokerApi.provisionService(serviceId, orgId, spaceId));
  var code = result.code;
  if (code === 200 || code === 201) {
    res.status(code).send({ "dashboard_url": result.dashboard_url });
    res.end();
  } else {
    res.sendStatus(code);
  }
}));


//define the sequence of middleware
app.use(auth);
app.use(bodyParser.urlencoded({ extended: true }));
app.use(bodyParser.json());
app.use('/v2', router);

var server = app.listen(port, function () {
    var port = server.address().port;
    console.log('Service broker app is running and listening at port %s', port);
});
module.exports = server;