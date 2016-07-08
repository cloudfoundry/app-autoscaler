'use strict';

var fs = require('fs');
var path = require('path');
var supertest = require("supertest");
var expect = require('chai').expect;
var nock = require('nock');
var uuid = require('uuid');
var serviceInstance = require('../../lib/models')().service_instance;
var binding = require('../../lib/models')().binding;

var settings = JSON.parse(
  fs.readFileSync(path.join(__dirname, '../../config/settings.json'), 'utf8'));
var auth = new Buffer(settings.username + ":" + settings.password).toString('base64');
var messageUtil = require(path.join(__dirname, '../../lib/util/messageUtil.js'))()
var scope;

function initNockBind201() {
  scope = nock(settings.apiServerUri)
    .put(/\/v1\/apps\/.*\/policy/)
    .reply(201, {
      'success': true,
      'error': null,
      'result': "created"
    });
}

function initNockBind400() {
  scope = nock(settings.apiServerUri)
    .put(/\/v1\/apps\/.*\/policy/)
    .reply(400, {
      'success': false,
      'error': null,
      'result': "error"
    });
}

function initNockBind500() {
  scope = nock(settings.apiServerUri)
    .put(/\/v1\/apps\/.*\/policy/)
    .reply(500, {
      'success': false,
      'error': null,
      'result': "error"
    });
}



describe('binding RESTful API', function() {
  var server, serviceInstanceId, serviceInstanceId2, orgId, spaceId, appId,appId2, bindingId;
  serviceInstanceId = uuid.v4();
  orgId = uuid.v4();
  spaceId = uuid.v4();
  appId = uuid.v4();
  appId2 = uuid.v4();
  bindingId = uuid.v4();
  serviceInstanceId2 = uuid.v4();
  var fake_service_condition = {
    'serviceInstanceId': serviceInstanceId,
    'orgId': orgId,
    'spaceId': spaceId,
    where: { 'serviceInstanceId': serviceInstanceId, 'orgId': orgId, 'spaceId': spaceId },
  };
  var policy = { "policy": "testPolicy" };

  before(function(done) {
    serviceInstance.sequelize.sync().then(function(result) {
      binding.sequelize.sync().then(function(result) {
        done();
      }).catch(function(error1) {
        console.log("Failed to sync model binding, error: " + error1);
        done(error1);
      });
    }).catch(function(error2) {
      console.log("Failed to sync model serviceInstance, error: " + error2);
      done(error2);
    });
  });

  beforeEach(function(done) {
    delete require.cache[require.resolve('../../lib/index.js')];
    server = require(path.join(__dirname, '../../lib/index.js'));
    binding.truncate({ cascade: true }).then(function(result) {
      serviceInstance.truncate({ cascade: true }).then(function(result) {
        serviceInstance.findOrCreate(fake_service_condition).then(function(result) {
          done();
        }).catch(function(error1) {
          done(error1);
        });
      }).catch(function(error2) {
        done(error2);
      });
    }).catch(function(error3) {
      done(error3);
    });
  });

  afterEach(function(done) {
    server.close(done);
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
          .expect({"description":messageUtil.getMessage("POLICY_REQUIRED")}, done);
      });
    });

    context('when there is no record', function() {
      it("creates a new binding with 201", function(done) {
        initNockBind201();
        supertest(server)
          .put("/v2/service_instances/" + serviceInstanceId + "/service_bindings/" + bindingId)
          .set("Authorization", "Basic " + auth)
          .send({ "app_guid": appId, "parameters": policy })
          .expect(201)
          .expect('Content-Type', /json/)
          .expect({}, done);
      });
      context("when the api server return error", function() {
        it("return a 400", function(done) {
          initNockBind400();
          supertest(server)
            .put("/v2/service_instances/" + serviceInstanceId + "/service_bindings/" + bindingId)
            .set("Authorization", "Basic " + auth)
            .send({ "app_guid": appId, "parameters": policy })
            .expect(400)
            .expect('Content-Type', /json/)
            .expect({}, done);
        });
        it("return a 500", function(done) {
          initNockBind500();
          supertest(server)
            .put("/v2/service_instances/" + serviceInstanceId + "/service_bindings/" + bindingId)
            .set("Authorization", "Basic " + auth)
            .send({ "app_guid": appId, "parameters": policy })
            .expect(500)
            .expect('Content-Type', /json/)
            .expect({}, done);
        });
      });
    });

    context('when an binding already exists', function() {

      beforeEach(function(done) {
        initNockBind201();
        supertest(server)
          .put("/v2/service_instances/" + serviceInstanceId + "/service_bindings/" + bindingId)
          .set("Authorization", "Basic " + auth)
          .send({ "app_guid": appId, "parameters": policy })
          .expect(201)
          .expect('Content-Type', /json/)
          .expect({}, done);
      });

      context('when erviceInstanceId and appId are conflict with an existed one', function() {
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
      context('when bindingId are conflict with an existed one', function() {
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
        it('returns a 499', function(done) {
          supertest(server)
            .put("/v2/service_instances/" + serviceInstanceId2 + "/service_bindings/" + bindingId)
            .set("Authorization", "Basic " + auth)
            .set('Accept', 'application/json')
            .send({ "app_guid": appId, "parameters": policy })
            .expect(499)
            .expect({"description":messageUtil.getMessage("DUPLICATED_BIND",{"applicationId":appId})}, done);
        });
      });
    });

  });
});