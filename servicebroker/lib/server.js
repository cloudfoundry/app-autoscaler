'use strict';
module.exports = function(settings, catalog, callback) {
    var https = require('https');
    var http = require('http');
    var express = require('express');
    var basicAuth = require('basic-auth');
    var bodyParser = require('body-parser');
    var fs = require('fs');
    var path = require('path');
    var logger = require(path.join(__dirname, './logger/logger.js'));
    var models = require(path.join(__dirname, './models'))(settings.db, callback);
    
    var publicPort = settings.publicPort;
    var port = settings.port;
    var healthPort = settings.healthPort;
    var publicOptions = {};
    
    if(settings.publicTls){
        if(!fs.existsSync(settings.publicTls.keyFile)){
            logger.error("Invalid TLS key path: " + settings.publicTls.keyFile);
            throw new Error("Invalid TLS key path: " + settings.publicTls.keyFile);
        }
        if(!fs.existsSync(settings.publicTls.certFile)){
            logger.error("Invalid TLS certificate path: " + settings.publicTls.certFile);
            throw new Error("Invalid TLS certificate path: " + settings.publicTls.certFile);
        }
        if(!fs.existsSync(settings.publicTls.caCertFile)){
            logger.error("Invalid TLS ca certificate path: " + settings.publicTls.caCertFile);
            throw new Error("Invalid TLS ca certificate path: " + settings.publicTls.caCertFile);
        }

        publicOptions = {
            key: fs.readFileSync(settings.publicTls.keyFile),
            cert: fs.readFileSync(settings.publicTls.certFile),
            ca: fs.readFileSync(settings.publicTls.caCertFile)
        }
    }

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
    var auth = function(req, res, next) {
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
    };


    //define the sequence of middleware
    app.use(auth);
    app.use(bodyParser.urlencoded({ extended: true }));
    app.use(bodyParser.json());
    require('./routes')(app, settings, catalog, models);

    var internalApp = express();
    internalApp.use(bodyParser.urlencoded({ extended: true }));
    internalApp.use(bodyParser.json());
    require('./internalRoutes')(internalApp, settings);

    var healthApp = express();
    var health = require('./healthendpoint/health')()  
    healthApp.use('/', health);

    var publicServer;
    if(settings.publicTls){
        publicServer = https.createServer(publicOptions, app).listen(publicPort, function() {
            var port = publicServer.address().port;
            logger.info('Service broker server is running in secure mode', { 'port': port });
        });
    }else{
        publicServer = http.createServer(app).listen(publicPort, function() {
            var port = publicServer.address().port;
            logger.info('Service broker server is running', { 'port': port });
        });
    }

    var internalServer;
    if(settings.tls){
        internalServer = https.createServer(options, internalApp).listen(port, function() {
            var port = internalServer.address().port;
            logger.info('Service broker internal server is running in secure mode', { 'port': port });
        });
    }else{
        internalServer = http.createServer(internalApp).listen(port, function() {
            var port = internalServer.address().port;
            logger.info('Service broker internal server is running', { 'port': port });
        });
    }

    var healthServer;
    healthServer = http.createServer(healthApp).listen(healthPort || 0, function () {
      logger.info('Service broker health server is running', { 'port': healthServer.address().port });
    });

    var gracefulShutdown = function(signal) {
        logger.info("Received " + signal + " signal, shutting down gracefully...");
        publicServer.close(function() {
            logger.info('Everything is cleanly shutdown for service broker server');
            internalServer.close(function() {
                logger.info('Everything is cleanly shutdown for service broker internal server');
                healthServer.close(function(){
                    logger.info('Everything is cleanly shutdown for service broker health server');
                    process.exit();
                    
                });
            });
        });
    }

    //listen for SIGUSR2 signal e.g. user-defined signal
    process.on ('SIGUSR2', function(){
        gracefulShutdown('SIGUSR2')
    });

    return  {"internalServer": internalServer, "publicServer": publicServer, "healthServer": healthServer};;
}
