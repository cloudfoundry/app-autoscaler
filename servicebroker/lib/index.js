'use strict'
var path = require('path');
var fs = require('fs');
var logger = require(path.join(__dirname, './logger/logger.js'));

var args = process.argv;
if (!(args.length == 4 && args[2] == "-c" && args[3] != "")) {
    throw new Error("missing config file\nUsage:use '-c' option to specify the config file path");
}

var configFilePath = args[3];

if (!configFilePath || !fs.existsSync(configFilePath)) {
    logger.error("Invalid configuration file path: " + configFilePath);
    throw new Error('configuration file does not exist:' + configFilePath);
}
var settings = require(path.join(__dirname, './config/setting.js'))((JSON.parse(
    fs.readFileSync(configFilePath, 'utf8'))));
var serviceCatalogPath = path.resolve(settings.serviceCatalogPath);
var catalog = JSON.parse(fs.readFileSync(serviceCatalogPath, 'utf8'));
var validateResult = settings.validate();
if (validateResult.valid === false) {
    logger.error("Invalid configuration: " + validateResult.message);
    throw new Error('settings.json is invalid');
}

var server = require(path.join(__dirname, '../lib/server.js'))(settings, catalog);
