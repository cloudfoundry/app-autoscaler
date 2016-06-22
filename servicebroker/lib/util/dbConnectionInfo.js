'use strict';

var path = require('path');
var cp = require('child_process');
var fs = require('fs');

var settings = JSON.parse(fs.readFileSync(path.join(__dirname, '../../config/settings.json'), 'utf8'));
var originalDbServerUri = settings.dbServer;
var dbServer = originalDbServerUri.replace(/\/$/g, ""); 
var dbName = settings.dbName.toLowerCase();
var dbConnectionUri = dbServer + "/" + dbName;

module.exports.dbServer = dbServer;
module.exports.dbName = dbName;
module.exports.dbConnectionUri = dbConnectionUri;
