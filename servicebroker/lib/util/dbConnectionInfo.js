'use strict';
var path = require('path');
var fs = require('fs');
var settings = JSON.parse(fs.readFileSync(path.join(__dirname, '../../config/settings.json'), 'utf8'));

var dbUri = settings.dbUri.replace(/\/$/g, "").toLowerCase();
var dbName = dbUri.slice(dbUri.lastIndexOf("/")+1, dbUri.length);
var dbServer = dbUri.slice(0, dbUri.lastIndexOf("/"));

module.exports.dbUri = dbUri;
module.exports.dbName = dbName;
module.exports.dbServer = dbServer;
