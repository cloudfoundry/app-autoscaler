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


    var server;
    if(settings.tls){
        server = https.createServer(options, app).listen(port, function() {
            var port = server.address().port;
            logger.info('Service broker app is running in secure mode', { port: port });
        });
    }else{
        server = http.createServer(app).listen(port, function() {
            var port = server.address().port;
            logger.info('Service broker app is running', { port: port });
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
