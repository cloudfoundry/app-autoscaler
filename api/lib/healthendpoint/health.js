'use strict';
var client = require('prom-client');
var register = client.register
var collectDefaultMetrics = client.collectDefaultMetrics;
collectDefaultMetrics({ prefix: 'autoscaler_apiserver_' });
module.exports = function () {
  

  var express = require('express');
  var router = express.Router();
  router.get('/', function (req, res) {
    res.set('Content-Type', register.contentType);
    res.send(register.metrics());
  });
  return router;
}