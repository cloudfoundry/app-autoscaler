'use strict'
var path = require('path');
var args = process.argv;
if (!(args.length == 4 && args[2] == "-c" && args[3] != "")) {
    throw new Error("missing config file\nUsage:use '-c' option to specify the config file path");
}
var apiServer = require(path.join(__dirname, 'app.js'))(args[3]);
