'use strict';

var fs = require('fs');
var path = require('path');

var catalog = JSON.parse(
  fs.readFileSync(path.join(__dirname, '../../config/catalog.json'), 'utf8'));

var API = function API() {};
API.prototype.getCatalog = function() {
  return catalog;
}

module.exports = API;