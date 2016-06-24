'use strict';

var fs = require('fs');
var path = require('path');
var supertest = require("supertest");
var should = require('should');
var uuid = require('uuid');

var settings = JSON.parse(
  fs.readFileSync(path.join(__dirname, '../../config/settings.json'), 'utf8'));
var auth = new Buffer(settings.username + ":" + settings.password).toString('base64');
describe('provisionService', function() {
  var server;
  var serviceId = uuid.v4();
  var orgId = uuid.v4();
  var spaceId = uuid.v4();
  var orgIdAgain = uuid.v4();
  var spaceIdAgain = uuid.v4();
  beforeEach(function() {
    delete require.cache[require.resolve('../../lib/index.js')];
    server = require(path.join(__dirname, '../../lib/index.js'));
  });

  afterEach(function(done) {
    server.close(done);
  });

  it("creates a new instance", function(done) {
    supertest(server)
      .put("/v2/service_instances/" + serviceId)
      .set("Authorization", "Basic " + auth)
      .send({ "organization_guid": orgId, "space_guid": spaceId })
      .expect(201)
      .expect('Content-Type', /json/)
      .expect({
        dashboard_url: ''
      }, done);
  });

  context('when an instance already exists', function(done) {
    beforeEach(function() {
      supertest(server)
        .put("/v2/service_instances/" + serviceId)
        .set("Authorization", "Basic " + auth)
        .send({ "organization_guid": orgId, "space_guid": spaceId })
        .expect(201);
    });

    context('when orgId and spaceId are identical', function(done) {
      it('returns a 200', function(done) {
        supertest(server)
          .put("/v2/service_instances/" + serviceId)
          .set("Authorization", "Basic " + auth)
          .send({ "organization_guid": orgId, "space_guid": spaceId })
          .expect(200)
          .expect('Content-Type', /json/)
          .expect({
            dashboard_url: ''
          }, done);
      });
    });

    context('when serviceId already exists', function(done) {
      it('returns a 409', function(done) {
        supertest(server)
          .put("/v2/service_instances/" + serviceId)
          .set("Authorization", "Basic " + auth)
          .send({ "organization_guid": orgIdAgain, "space_guid": spaceIdAgain })
          .expect(409, done);
      });
    });
  });
});