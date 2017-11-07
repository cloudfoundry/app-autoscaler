'use strict'; 
module.exports = function(configFilePath) {
  var https = require('https');
  var http = require('http');
  var fs = require('fs');
  var path = require('path');
  var express = require('express');
  var helmet = require('helmet')
  var bodyParser = require('body-parser');
  var logger = require('./lib/log/logger');
  var HttpStatus = require('http-status-codes');
  var oauth = require('./lib/oauth/oauth')(configFilePath);

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
  var publicPort = settings.publicPort;
  
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

  var publicOptions = {};
    
  if(settings.publicTls){
    if(!fs.existsSync(settings.publicTls.keyFile)){
        logger.error("Invalid public TLS key path: " + settings.publicTls.keyFile);
        throw new Error("Invalid publicTls key path: " + settings.publicTls.keyFile);
    }
    if(!fs.existsSync(settings.publicTls.certFile)){
        logger.error("Invalid public TLS certificate path: " + settings.publicTls.certFile);
        throw new Error("Invalid public TLS certificate path: " + settings.publicTls.certFile);
    }
    if(!fs.existsSync(settings.publicTls.caCertFile)){
        logger.error("Invalid public TLS ca certificate path: " + settings.publicTls.caCertFile);
        throw new Error("Invalid public TLS ca certificate path: " + settings.publicTls.caCertFile);
    }

    publicOptions = {
        key: fs.readFileSync(settings.publicTls.keyFile),
        cert: fs.readFileSync(settings.publicTls.certFile),
        ca: fs.readFileSync(settings.publicTls.caCertFile)
    }
  }

  var app = express();
  app.use(helmet())
  app.use(helmet.contentSecurityPolicy({
    directives: {
      defaultSrc : ['\'self\'' ],
      scriptSrc : [ '\'self\''],
    },
    browserSniff: false
  }))
  app.use(helmet.noCache())
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

  var publicApp = express();
  publicApp.use(helmet())
  publicApp.use(helmet.contentSecurityPolicy({
    directives: {
      defaultSrc : ['\'self\'' ],
      scriptSrc : [ '\'self\''],
    },
    browserSniff: false
  }))
  publicApp.use(helmet.noCache())
  publicApp.use(bodyParser.json());
  publicApp.use(bodyParser.urlencoded({ extended: false }));
  publicApp.use('/v1/apps/:app_id/*',function(req, res, next){
    oauth.checkUserAuthorization(req,function(error,isSpaceDeveloper){
      if(error){
        if(error.statusCode == HttpStatus.UNAUTHORIZED){
          res.status(HttpStatus.UNAUTHORIZED).send({});
        }else{
          res.status(HttpStatus.INTERNAL_SERVER_ERROR).send({});
        }
      }else{
        if(isSpaceDeveloper){
          next();
        }else{
          res.status(HttpStatus.UNAUTHORIZED).send({});
        }
      }
    });
  });
  publicApp.use('/health', require('express-healthcheck')());
  var policies = require('./lib/routes/policies')(settings);
  var scalingHistories = require('./lib/routes/scalingHistories')(settings);
  var metrics = require('./lib/routes/metrics')(settings);
  publicApp.use('/v1/apps',policies);
  publicApp.use('/v1/apps',scalingHistories);
  publicApp.use('/v1/apps',metrics);
  publicApp.use(function(err, req, res, next) {
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
  
  var publicServer;
  if(settings.publicTls){
    publicServer = https.createServer(publicOptions, publicApp).listen(publicPort || 3003, function() {
        logger.info('Autoscaler public API server started in secure mode',{'port':publicServer.address().port} );    
    });
  }else{
    publicServer = http.createServer(publicApp).listen(publicPort || 3003, function() {
        logger.info('Autoscaler public API server started',{'port':publicServer.address().port} );    
    });
  }


  var gracefulShutdown = function(signal) {
    logger.info("Received " + signal + " signal, shutting down gracefully...");
    server.close(function() {
      logger.info('Everything is cleanly shutdown for internal API server');
      publicServer.close(function() {
        logger.info('Everything is cleanly shutdown for public API server');
        process.exit();
      });
    });
    
  }

  //listen for SIGUSR2 signal e.g. user-defined signal
  process.on ('SIGUSR2', function(){
    gracefulShutdown('SIGUSR2')
  });

  return {"internalServer": server, "publicServer": publicServer};
}
 
