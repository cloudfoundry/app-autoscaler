'use strict'; 
module.exports = function(configFilePath) {
  var https = require('https');
  var http = require('http');
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
  var settings = require(path.join(__dirname, './lib/config/setting.js'))((JSON.parse(
        fs.readFileSync(configFilePath, 'utf8'))));
  var validateResult = settings.validate();
  if (validateResult.valid === false) {
      logger.error("Invalid configuration: " + validateResult.message);
      throw new Error('settings.json is invalid');
  }
  var port = settings.port;
  
  var options = {};
    
  if(settings.tls){
    if(!fs.existsSync(settings.tls.keyFile)){
        logger.error("Invalid TLS key path: " + settings.tls.keyFile);
        throw new Error("Invalid TLS key path: " + settings.tls.keyFile);
    }
    if(!fs.existsSync(settings.tls.certFile)){
        logger.error("Invalid TLS certificate path: " + settings.tls.certFile);
        throw new Error("Invalid TLS certificate path: " + settings.tls.certFile);
    }
    if(!fs.existsSync(settings.tls.caCertFile)){
        logger.error("Invalid TLS ca certificate path: " + settings.tls.caCertFile);
        throw new Error("Invalid TLS ca certificate path: " + settings.tls.caCertFile);
    }

    options = {
        key: fs.readFileSync(settings.tls.keyFile),
        cert: fs.readFileSync(settings.tls.certFile),
        ca: fs.readFileSync(settings.tls.caCertFile)
    }
  }
  var app = express();
  app.use(bodyParser.json());
  app.use(bodyParser.urlencoded({ extended: false }));
  app.use('/health', require('express-healthcheck')());
  var policies = require('./lib/routes/policies')(settings);
  var scalingHistories = require('./lib/routes/scalingHistories')(settings);
  var metrics = require('./lib/routes/metrics')(settings);
  app.use('/v1/apps',policies);
  app.use('/v1/apps',scalingHistories);
  app.use('/v1/apps',metrics);
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

  var server;
  if(settings.tls){
    server = https.createServer(options, app).listen(port || 3002, function() {
        logger.info('Autoscaler API server started in secure mode',{'port':server.address().port} );    
    });
  }else{
    server = http.createServer(app).listen(port || 3002, function() {
        logger.info('Autoscaler API server started',{'port':server.address().port} );    
    });
  }


  var gracefulShutdown = function(signal) {
    logger.info("Received " + signal + " signal, shutting down gracefully...");
    server.close(function() {
      logger.info('Everything is cleanly shutdown');
      process.exit();
    })
  }

  //listen for SIGUSR2 signal e.g. user-defined signal
  process.on ('SIGUSR2', function(){
    gracefulShutdown('SIGUSR2')
  });

  return server;
}
 
