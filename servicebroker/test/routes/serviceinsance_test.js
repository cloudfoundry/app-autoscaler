'use strict';

var fs = require('fs');
var path = require('path');
var supertest = require("supertest");
var should = require('should');
var uuid = require('uuid');
var serviceInstance = require('../../lib/models')().service_instance;

var settings = JSON.parse(
  fs.readFileSync(path.join(__dirname, '../../config/settings.json'), 'utf8'));
var auth = new Buffer(settings.username + ":" + settings.password).toString('base64');


describe('provisionService RESTful API', function() {
  var server, serviceId, orgId, spaceId, orgIdAgain, spaceIdAgain;
  serviceId = uuid.v4();
  orgId = uuid.v4();
  spaceId = uuid.v4();
  orgIdAgain = uuid.v4();
  spaceIdAgain = uuid.v4();

  before(function() {
    delete require.cache[require.resolve('../../lib/index.js')];
    server = require(path.join(__dirname, '../../lib/index.js'));
    serviceInstance.sequelize.sync();
    serviceInstance.truncate({ cascade: true });
  });

  after(function(done) {
    server.close(done);
  });

  beforeEach(function() {
    serviceInstance.truncate({ cascade: true });
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

    beforeEach(function(done) {
      serviceId = uuid.v4();
      orgId = uuid.v4();
      spaceId = uuid.v4();
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

    context('when orgId and spaceId are conflict to previous record', function(done) {

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