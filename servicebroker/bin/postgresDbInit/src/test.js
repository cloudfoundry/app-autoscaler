'use strict';

var path = require('path');
var cp = require('child_process');
var fs = require('fs');

var settings = JSON.parse(fs.readFileSync(path.join(__dirname, '../../../config/settings.json'), 'utf8'));
var dbConnectionInfo = require(path.join(__dirname, '../../../lib/util/dbConnectionInfo.js'))(settings.dbUri);

console.log (dbConnectionInfo);
