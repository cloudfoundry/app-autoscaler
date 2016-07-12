'use strict';

var supertest = require("supertest");
var expect = require('chai').expect;
var nock = require('nock');
var uuid = require('uuid');

var fs = require('fs');
var path = require('path');
var settings = require(path.join(__dirname, '../../lib/config/setting.js'))((JSON.parse(
  fs.readFileSync(path.join(__dirname, '../../config/settings.json'), 'utf8'))));

var models = require('../../lib/models')(settings);
var serviceInstance = models.service_instance;
var binding = models.binding;

var auth = new Buffer(settings.username + ":" + settings.password).toString('base64');
var messageUtil = require(path.join(__dirname, '../../lib/util/messageUtil.js'))()
var scope;

function initNockBind(statusCode) {
  scope = nock(settings.apiServerUri)
    .put(/\/v1\/apps\/.*\/policy/)
    .reply(statusCode, {
      'success': true,
      'error': null,
      'result': "created"
    });
}

describe('binding RESTful API', function() {
  var server, serviceInstanceId, serviceInstanceId2, orgId, spaceId, appId, appId2, bindingId;
  serviceInstanceId = uuid.v4();
  orgId = uuid.v4();
  spaceId = uuid.v4();
  appId = uuid.v4();
  appId2 = uuid.v4();
  bindingId = uuid.v4();
  serviceInstanceId2 = uuid.v4();
  var service_condition = {
    'serviceInstanceId': serviceInstanceId,
    'orgId': orgId,
    'spaceId': spaceId,
    where: { 'serviceInstanceId': serviceInstanceId, 'orgId': orgId, 'spaceId': spaceId },
  };
  var service_condition2 = {
    'serviceInstanceId': serviceInstanceId2,
    'orgId': orgId,
    'spaceId': spaceId,
    where: { 'serviceInstanceId': serviceInstanceId2, 'orgId': orgId, 'spaceId': spaceId },
  };
  var policy = { "policy": "testPolicy" };
  before(function(done) {
    server = require(path.join(__dirname, '../../lib/index.js'));
    done();
  });
  beforeEach(function(done) {
    delete require.cache[require.resolve('../../lib/index.js')];
    binding.truncate({ cascade: true }).then(function(result) {
      serviceInstance.truncate({ cascade: true }).then(function(result) {
        serviceInstance.create(service_condition).then(function(result) {
          done();
        });
      });
    });
  });


  context('Bind service ', function() {

    context('when there is no policy in request', function() {
      it("return a 400", function(done) {
        supertest(server)
          .put("/v2/service_instances/" + serviceInstanceId + "/service_bindings/" + bindingId)
          .set("Authorization", "Basic " + auth)
          .send({ "app_guid": appId })
          .expect(400)
          .expect('Content-Type', /json/)
          .expect({ "description": messageUtil.getMessage("POLICY_REQUIRED") }, done);
      });
    });
    context('when the service instance does not exist', function() {
      it("return a 404", function(done) {
        supertest(server)
          .put("/v2/service_instances/" + serviceInstanceId2 + "/service_bindings/" + bindingId)
          .set("Authorization", "Basic " + auth)
          .send({ "app_guid": appId, "parameters": policy })
          .expect(404)
          .expect('Content-Type', /json/)
          .expect({ "description": messageUtil.getMessage("SERVICEINSTANCE_NOT_EXIST", { "serviceInstanceId": serviceInstanceId2 }) }, done);
      });
    });


    context('when there is no record', function() {
      it("creates a new binding with 201", function(done) {
        initNockBind(201);
        supertest(server)
          .put("/v2/service_instances/" + serviceInstanceId + "/service_bindings/" + bindingId)
          .set("Authorization", "Basic " + auth)
          .send({ "app_guid": appId, "parameters": policy })
          .expect(201)
          .expect('Content-Type', /json/)
          .expect({}, done);
      });
      context("when the api server returns error", function() {
        it("return a 400", function(done) {
          initNockBind(400);
          supertest(server)
            .put("/v2/service_instances/" + serviceInstanceId + "/service_bindings/" + bindingId)
            .set("Authorization", "Basic " + auth)
            .send({ "app_guid": appId, "parameters": policy })
            .expect(400)
            .expect('Content-Type', /json/)
            .expect({}, done);
        });
        it("return a 500", function(done) {
          initNockBind(500);
          supertest(server)
            .put("/v2/service_instances/" + serviceInstanceId + "/service_bindings/" + bindingId)
            .set("Authorization", "Basic " + auth)
            .send({ "app_guid": appId, "parameters": policy })
            .expect(500)
            .expect('Content-Type', /json/)
            .expect({}, done);
        });
        context('when the api server returns other error than 400, 500', function() {
        it('returns a 500', function(done) {
          initNockBind(300);
          supertest(server)
            .put("/v2/service_instances/" + serviceInstanceId + "/service_bindings/" + bindingId)
            .set("Authorization", "Basic " + auth)
            .set('Accept', 'application/json')
            .send({ "app_guid": appId, "parameters": policy })
            .expect(500)
            .expect('Content-Type', /json/)
            .expect({}, done);
        });
      });
      });
    });

    context('when a binding exists for the app', function() {

      beforeEach(function(done) {
        initNockBind(201);
        supertest(server)
          .put("/v2/service_instances/" + serviceInstanceId + "/service_bindings/" + bindingId)
          .set("Authorization", "Basic " + auth)
          .send({ "app_guid": appId, "parameters": policy })
          .expect(201)
          .expect('Content-Type', /json/)
          .expect({}, done);
      });

      context('when an app is already bound', function() {
        it('returns a 409', function(done) {
          supertest(server)
            .put("/v2/service_instances/" + serviceInstanceId + "/service_bindings/" + bindingId)
            .set("Authorization", "Basic " + auth)
            .set('Accept', 'application/json')
            .send({ "app_guid": appId, "parameters": policy })
            .expect(409)
            .expect('Content-Type', /json/)
            .expect({}, done);
        });
      });
      context('when the binding id already exists', function() {
        beforeEach(function(done) {
          serviceInstance.create(service_condition2).then(function(result) {
            done();
          });
        });
        it('returns a 409', function(done) {
          supertest(server)
            .put("/v2/service_instances/" + serviceInstanceId2 + "/service_bindings/" + bindingId)
            .set("Authorization", "Basic " + auth)
            .set('Accept', 'application/json')
            .send({ "app_guid": appId2, "parameters": policy })
            .expect(409)
            .expect('Content-Type', /json/)
            .expect({}, done);
        });
      });
      context('when an service instance has already bound to an application', function() {
        beforeEach(function(done) {
          serviceInstance.create(service_condition2).then(function(result) {
            done();
          });
        });
        it('returns a 499', function(done) {
          supertest(server)
            .put("/v2/service_instances/" + serviceInstanceId2 + "/service_bindings/" + bindingId)
            .set("Authorization", "Basic " + auth)
            .set('Accept', 'application/json')
            .send({ "app_guid": appId, "parameters": policy })
            .expect(499)
            .expect({ "description": messageUtil.getMessage("DUPLICATE_BIND", { "applicationId": appId }) }, done);
        });
      });
    });

  });
});