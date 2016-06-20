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
    data.should.have.property('services');
    JSON.stringify(data).length.should.equal(JSON.stringify(catalog).length);
    data.services[0].id.should.equal(catalog.services[0].id);
    data.services[0].name.should.equal(catalog.services[0].name);
    data.services[0].description.should.equal(catalog.services[0].description);
    data.services[0].plans[0].id.should.equal(catalog.services[0].plans[0].id);
    data.services[0].plans[0].name.should.equal(catalog.services[0].plans[0].name);
    data.services[0].plans[0].description.should.equal(catalog.services[0].plans[0].description);
  });

});