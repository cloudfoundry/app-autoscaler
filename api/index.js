'use strict'
var path = require('path');
var fs = require('fs');
var NodeCache = require('node-cache');
var logger = require('./lib/log/logger');
var args = process.argv;
if (!(args.length == 4 && args[2] == "-c" && args[3] != "")) {
  throw new Error("missing config file\nUsage:use '-c' option to specify the config file path");
}
var configFilePath = args[3];

if (!configFilePath || !fs.existsSync(configFilePath)) {
  logger.error("Invalid configuration file path: " + configFilePath);
  throw new Error('configuration file does not exist:' + configFilePath);
}
var settings = require(path.join(__dirname, './lib/config/setting.js'))((JSON.parse(
  fs.readFileSync(configFilePath, 'utf8'))));
var errorCallback = function(err) {
  if (err) {
    logger.error('server will exit', err);
    throw err;
  }
}
var credentialCache = new NodeCache();
var apiServer = require(path.join(__dirname, 'app.js'))(settings, credentialCache, errorCallback);