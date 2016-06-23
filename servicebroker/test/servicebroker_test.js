'use strict';

var fs = require('fs');
var path = require('path');
var should = require('should');
var uuid = require('uuid');
var async = require('asyncawait/async');
var await = require('asyncawait/await');
var ServiceBroker = require(path.join(__dirname, '../lib/servicebroker.js'));

var catalog = JSON.parse(fs.readFileSync(path.join(__dirname, '../config/catalog.json'), 'utf8'));
var servicebroker = new ServiceBroker();

describe('Unit Test for Service Broker', function() {

  it('getCatalog() should return catalog info with JSON format', function() {
    var data = servicebroker.getCatalog();
    JSON.stringify(data).should.equal(JSON.stringify(catalog));
  });
  it('provisionService() should return 201 when serviceId, orgId and spaceId are new ones', async(function() {
    var serviceId = uuid.v4();
    var orgId = uuid.v4();
    var spaceId = uuid.v4();
    var result = await (servicebroker.provisionService(serviceId, orgId, spaceId));
    result.code.should.equal(201);
  }));
  it('provisionService() should return 200 when serviceId, orgId and spaceId are all the same as an exsited row in database', async(function() {
    var serviceId = uuid.v4();
    var orgId = uuid.v4();
    var spaceId = uuid.v4();
    var result = await (servicebroker.provisionService(serviceId, orgId, spaceId));
    result.code.should.equal(201);
    var result = await (servicebroker.provisionService(serviceId, orgId, spaceId));
    result.code.should.equal(200);
  }));
  it('provisionService() should return 409 when serviceId is duplicated but orgId and spaceId are not', async(function() {
    var serviceId = uuid.v4();
    var orgId = uuid.v4();
    var spaceId = uuid.v4();
    var result = await (servicebroker.provisionService(serviceId, orgId, spaceId));
    result.code.should.equal(201);
    orgId = uuid.v4();
    spaceId = uuid.v4();
    var result = await (servicebroker.provisionService(serviceId, orgId, spaceId));
    result.code.should.equal(409);
  }));

});