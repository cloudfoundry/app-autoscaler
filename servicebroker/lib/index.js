'use strict'
var path = require('path');
var args = process.argv;
if (!(args.length == 6 && args[2] == "-c" && args[3] != "" && args[4] == "-sc" && args[5] != "")) {
    throw new Error("missing config file\nUsage:use '-c' option to specify the config file path and -sc to specify service catalog file path");
}
var server = require(path.join(__dirname, '../lib/server.js'))(args[3],args[5]);
