'use strict';
module.exports = function(configFilePath) {
  var fs = require('fs');
  var path = require('path');
  var express = require('express');
  var bodyParser = require('body-parser');
  var logger = require('./lib/log/logger');
  var HttpStatus = require('http-status-codes');

  if (!configFilePath || !fs.existsSync(configFilePath)) {
      logger.error("Invalid configuration file path: " + configFilePath);
      throw new Error('configuration file does not exist:' + configFilePath);
  }
  var settings = require(path.join(__dirname, './lib/config/settings.js'))((JSON.parse(
        fs.readFileSync(configFilePath, 'utf8'))));
  var validateResult = settings.validate();
  if (validateResult.valid === false) {
      logger.error("Invalid configuration: " + validateResult.message);
      throw new Error('settings.json is invalid');
  }
  var port = settings.port;
  var app = express();
  app.use(bodyParser.json());
  app.use(bodyParser.urlencoded({ extended: false }));
  app.use('/health', require('express-healthcheck')());
  var server = app.listen(port || 3002, function() {
      logger.info('Autoscaler API server started',{'port':server.address().port} ); 
      var policies = require('./lib/routes/policies')(settings);
      app.use('/v1/policies',policies);
      app.use(function(err, req, res, next) {
        var errorResponse = {};
        if (err) {
          errorResponse = {
            'success': false,
            'error': err,
            'result': null
          };
        }
        res.status(HttpStatus.BAD_REQUEST).json(errorResponse);
      });
        
  });  
  return server;
}
 