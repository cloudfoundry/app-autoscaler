'use strict'
var path = require('path');
var logger = require('./logger/logger');
var args = process.argv;
if (!(args.length == 4 && args[2] == "-c" && args[3] != "")) {
    throw new Error("missing config file\nUsage:use '-c' option to specify the config file path");
}
var server = require(path.join(__dirname, '../lib/server.js'))(args[3]);

var gracefulShutdown = function(signal) {
  logger.info("Received " + signal + " signal, shutting down gracefully...");
  server.shutdown(function() {
    logger.info('Everything is cleanly shutdown');
    process.exit();
  })
}

//listen for SIGINT signal e.g. Ctrl-C
process.on ('SIGINT', function(){
  gracefulShutdown('SIGINT')
});

//listen for SIGUSR2 signal e.g. user-defined signal
process.on ('SIGUSR2', function(){
  gracefulShutdown('SIGUSR2')
});
