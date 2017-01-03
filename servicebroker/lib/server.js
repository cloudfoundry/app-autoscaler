'use strict';
module.exports = function(configFilePath) {
    var https = require('https')
    var express = require('express');
    var basicAuth = require('basic-auth');
    var bodyParser = require('body-parser');
    var fs = require('fs');
    var path = require('path');
    if (!configFilePath || !fs.existsSync(configFilePath)) {
        logger.error("Invalid configuration file path: " + configFilePath);
        throw new Error('configuration file does not exist:' + configFilePath);
    }

    var logger = require(path.join(__dirname, './logger/logger.js'));
    var settings = require(path.join(__dirname, './config/setting.js'))((JSON.parse(
        fs.readFileSync(configFilePath, 'utf8'))));
    var validateResult = settings.validate();
    if (validateResult.valid === false) {
        logger.error("Invalid configuration: " + validateResult.message);
        throw new Error('settings.json is invalid');
    }
    var port = settings.port;
    var options = {};
    
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
        next();
    };

    //define the sequence of middleware
    app.use(auth);
    app.use(bodyParser.urlencoded({ extended: true }));
    app.use(bodyParser.json());
    require('./routes')(app, settings);


    var server = https.createServer(options, app).listen(port, function() {
        var port = server.address().port;
        logger.info('Service broker app is running', { port: port });
    });
    return server;
}
