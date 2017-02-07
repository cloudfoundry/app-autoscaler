'use strict'
var path = require('path');
var args = process.argv;
if (!(args.length == 4 && args[2] == "-c" && args[3] != "")) {
    throw new Error("missing config file\nUsage:use '-c' option to specify the config file path");
}
var server = require(path.join(__dirname, '../lib/server.js'))(args[3]);
