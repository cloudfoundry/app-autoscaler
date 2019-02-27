'use strict';

process.env.NODE_TLS_REJECT_UNAUTHORIZED = '0';
var supertest = require("supertest");
var expect = require('chai').expect;
var nock = require('nock');
var uuid = require('uuid');

var fs = require('fs');
var path = require('path');
var BrokerServer = require(path.join(__dirname, '../../lib/server.js'));
var configFilePath = path.join(__dirname, '../../config/settings.json');
var catalog = JSON.parse(fs.readFileSync(path.join(__dirname, '../../config/catalog.json'), 'utf8'));
var settings = require(path.join(__dirname, '../../lib/config/setting.js'))((JSON.parse(
  fs.readFileSync(configFilePath, 'utf8'))));

var models = require('../../lib/models')(settings.db);
var serviceInstance = models.service_instance;
var binding = models.binding;

var auth = new Buffer(settings.username + ":" + settings.password).toString('base64');
var scope;

var VALIDATION_ERROR_FROM_API_SERVER = "validation error from apiserver";

function initNockBind(statusCode) {
  scope = nock(settings.apiserver.uri)
    .put(/\/v1\/apps\/.*\/policy/)
    .reply(statusCode, {
      'success': true,
      'error': null,
      'result': "created"
    });
}

function initNockBindWithCred(statusCode) {
  scope = nock(settings.apiserver.uri)
    .post(/\/v1\/apps\/.*\/credentials/)
    .reply(statusCode, {
      'success': true,
      'error': null,
      'result': "created"
    });
}

describe('binding RESTful API', function() {
  var servers, publicServer, internalServer,healthServer, serviceInstanceId, orgId, spaceId, appId, bindingId;
  serviceInstanceId = uuid.v4();
  orgId = uuid.v4();
  spaceId = uuid.v4();
  appId = uuid.v4();
  bindingId = uuid.v4();
  var service_condition = {
    'serviceInstanceId': serviceInstanceId,
    'orgId': orgId,
    'spaceId': spaceId,
    where: { 'serviceInstanceId': serviceInstanceId, 'orgId': orgId, 'spaceId': spaceId },
  };
  var policy = { "policy": "testPolicy" };
  before(function(done) {
    servers = BrokerServer(settings, catalog);
    publicServer = servers.publicServer;
    internalServer = servers.internalServer;
    healthServer = servers.healthServer;
    done();
  });
  after(function(done) {
    publicServer.close(function() {
      internalServer.close(function(){
        healthServer.close(done);
      });
    })
  });
  beforeEach(function(done) {
    binding.truncate({ cascade: true }).then(function(result) {
      serviceInstance.truncate({ cascade: true }).then(function(result) {
        serviceInstance.create(service_condition).then(function(result) {
          done();
        });
      });
    });
  });

  context("check binding", function() {
    context("application has not been bound to autoscaler", function() {
      it("return 200 with null", function(done) {
        supertest(internalServer)
          .get("/v1/apps/" + appId + "/service_bindings")
          .end(function(error, result) {
            expect(error).to.equal(null);
            expect(result.statusCode).to.equal(200);
            expect(result.body.binding).to.deep.equal(null);
            done();
          });
      });
    });

    context("application has been bound to autoscaler", function() {
      beforeEach(function(done) {
        initNockBind(201);
        initNockBindWithCred(200);
        supertest(publicServer)
          .put("/v2/service_instances/" + serviceInstanceId + "/service_bindings/" + bindingId)
          .set("Authorization", "Basic " + auth)
          .send({ "app_guid": appId, "parameters": policy })
          .expect(201)
          .expect('Content-Type', /json/)
          .expect({})
          .end(function(err, res) {
            binding.count({ where: { bindingId: bindingId } }).then(function(countRes) {
              expect(countRes).to.equal(1);
              done();
            })
          });
      });
      it("return 200 with binding", function(done) {
        supertest(internalServer)
          .get("/v1/apps/" + appId + "/service_bindings")
          .end(function(error, result) {
            expect(error).to.equal(null);
            expect(result.statusCode).to.equal(200);
            expect(result.body.binding.bindingId).to.equal(bindingId);
            expect(result.body.binding.appId).to.equal(appId);
            expect(result.body.binding.serviceInstanceId).to.equal(serviceInstanceId);
            done();
          });
      });
    });
  });

});