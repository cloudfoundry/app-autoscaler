'use strict';

var fs = require('fs');
var path = require('path');
var should = require('should');
var API = require(path.join(__dirname, '../../lib/api/api.js'));

var catalog = JSON.parse(fs.readFileSync(path.join(__dirname, '../../config/catalog.json'), 'utf8'));
var api = new API();

describe('Unit Test for API', function() {

  it('getCatalog() should return catalog info with JSON format', function() {
    var data = api.getCatalog();
    JSON.stringify(data).should.equal(JSON.stringify(catalog));
  });

});