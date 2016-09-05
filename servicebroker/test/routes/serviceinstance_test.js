'use strict';
var supertest = require("supertest");
var uuid = require('uuid');

var fs = require('fs');
var path = require('path');
var settings = require(path.join(__dirname, '../../lib/config/setting.js'))((JSON.parse(
  fs.readFileSync(path.join(__dirname, '../../config/settings.json'), 'utf8'))));

var models = require('../../lib/models')(settings.db.uri);
var serviceInstance = models.service_instance;

var auth = new Buffer(settings.username + ":" + settings.password).toString('base64');

describe('service instance RESTful API', function() {
  var server, serviceInstanceId, orgId, spaceId, orgIdAgain, spaceIdAgain;
  serviceInstanceId = uuid.v4();
  orgId = uuid.v4();
  spaceId = uuid.v4();
  orgIdAgain = uuid.v4();
  spaceIdAgain = uuid.v4();


  before(function() {
    delete require.cache[require.resolve('../../lib/index.js')];
    server = require(path.join(__dirname, '../../lib/index.js'));
  });

  after(function(done) {
    server.close(done);
  });

  beforeEach(function(done) {
    serviceInstance.truncate({ cascade: true });
    done();
  });

  context('Provision service', function() {
    context('when there is no record', function() {
      it("creates a new instance with 201", function(done) {
        supertest(server)
          .put("/v2/service_instances/" + serviceInstanceId)
          .set("Authorization", "Basic " + auth)
          .send({ "organization_guid": orgId, "space_guid": spaceId })
          .expect(201)
          .expect('Content-Type', /json/)
          .expect({
            dashboard_url: ''
          }, done);
      });
    });

    context('when an instance already exists', function() {

      beforeEach(function(done) {
        supertest(server)
          .put("/v2/service_instances/" + serviceInstanceId)
          .set("Authorization", "Basic " + auth)
          .send({ "organization_guid": orgId, "space_guid": spaceId })
          .expect(201)
          .expect('Content-Type', /json/)
          .expect({
            dashboard_url: ''
          }, done);
      });

      context('when orgId and spaceId are identical', function() {
        it('returns a 200', function(done) {
          supertest(server)
            .put("/v2/service_instances/" + serviceInstanceId)
            .set("Authorization", "Basic " + auth)
            .set('Accept', 'application/json')
            .send({ "organization_guid": orgId, "space_guid": spaceId })
            .expect(200)
            .expect('Content-Type', /json/)
            .expect({
              dashboard_url: ''
            }, done);
        });
      });

      context('when orgId and spaceId are conflict to previous record', function() {
        it('returns a 409', function(done) {
          supertest(server)
            .put("/v2/service_instances/" + serviceInstanceId)
            .set("Authorization", "Basic " + auth)
            .send({ "organization_guid": orgIdAgain, "space_guid": spaceIdAgain })
            .expect(409, done);
        });
      });
    });

  });


  context('Deprovision service ', function() {
    context('when there is no record', function() {
      it("delete an nonexist instance with 410", function(done) {
        supertest(server)
          .delete("/v2/service_instances/" + "nonexistid")
          .set("Authorization", "Basic " + auth)
          .expect(410)
          .expect('Content-Type', /json/)
          .expect({}, done);
      });
    });

    context('when an instance already exists', function() {
      beforeEach(function(done) {
        supertest(server)
            .put("/v2/service_instances/" + serviceInstanceId)
            .set("Authorization", "Basic " + auth)
            .send({ "organization_guid": orgId, "space_guid": spaceId })
            .expect(201)
            .expect('Content-Type', /json/)
            .expect({
              dashboard_url: ''
            }, done);
      });

      it("delete an instance with 200", function(done) {
        supertest(server)
          .delete("/v2/service_instances/" + serviceInstanceId)
          .set("Authorization", "Basic " + auth)
          .expect(200)
          .expect('Content-Type', /json/)
          .expect({}, done);
      });
    });


  });

});
