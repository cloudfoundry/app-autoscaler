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
var validateResult = settings.validate();
if (validateResult.valid === false) {
  logger.error("Invalid configuration: " + validateResult.message);
  throw new Error('settings.json is invalid');
}

var catalog = JSON.parse(fs.readFileSync(path.resolve(settings.serviceCatalogPath), 'utf8'));
var schemaValidation = JSON.parse(fs.readFileSync(path.resolve(settings.schemaValidationPath), 'utf8'));
for (let i = 0; i < catalog.services[0].plans.length; i++) {
  catalog.services[0].plans[i].schemas = schemaValidation.schemas
}

var errorCallback = function(err) {
  if (err) {
    logger.error('server will exit', err);
    throw err;
  }
}
var server = require(path.join(__dirname, '../lib/server.js'))(settings, catalog, errorCallback);