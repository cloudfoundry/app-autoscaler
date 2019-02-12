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
var messageUtil = require(path.join(__dirname, '../../lib/util/messageUtil.js'))(catalog)
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
      'username': "newusername",
      'password': "newpassword"
    });
}

var mockError = {
  'error': VALIDATION_ERROR_FROM_API_SERVER,
};

function initNockBindWithCredError(statusCode) {
  scope = nock(settings.apiserver.uri)
    .post(/\/v1\/apps\/.*\/credentials/)
    .replyWithError(mockError);
}

function initNockUnbindWithCred(statusCode) {
  scope = nock(settings.apiserver.uri)
    .delete(/\/v1\/apps\/.*\/credentials/)
    .reply(statusCode, {
      'success': true,
      'error': null,
      'result': "created"
    });
}

function initNockUnbindWithCredError(statusCode) {
  scope = nock(settings.apiserver.uri)
    .delete(/\/v1\/apps\/.*\/credentials/)
    .replyWithError(mockError);
}

function initNockUnBind(statusCode) {
  scope = nock(settings.apiserver.uri)
    .delete(/\/v1\/apps\/.*\/policy/)
    .reply(statusCode, {
      'success': true,
      'error': null,
      'result': "deleted"
    });
}

function initNockApiServerBindError(statusCode) {
  scope = nock(settings.apiserver.uri)
    .put(/\/v1\/apps\/.*\/policy/)
    .reply(statusCode, {
      'success': false,
      'error': VALIDATION_ERROR_FROM_API_SERVER,
      'result': null
    });
}

describe('binding RESTful API', function() {
  var servers, publicServer, internalServer, healthServer, serviceInstanceId, serviceInstanceId2, orgId, spaceId, appId, appId2, bindingId;
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

    servers = BrokerServer(settings, catalog, function(){});
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


  context('Bind service ', function() {

    context('when there is no record', function() {
      it("creates a new binding with 201", function(done) {
        initNockBind(201);
        initNockBindWithCred(201);
        supertest(publicServer)
          .put("/v2/service_instances/" + serviceInstanceId + "/service_bindings/" + bindingId)
          .set("Authorization", "Basic " + auth)
          .send({ "app_guid": appId, "parameters": policy })
          .expect(201)
          .expect('Content-Type', /json/)
          .expect({})
          .end(function(err, res) {
            expect(res.body.credentials.custom_metrics.username).to.equal('newusername');
            expect(res.body.credentials.custom_metrics.password).to.equal('newpassword');

            binding.count({ where: { bindingId: bindingId } }).then(function(countRes) {
              expect(countRes).to.equal(1);
              done();
            })
          });
      });

      it("failed to create a new binding because of credential generation failure", function(done) {
        initNockBindWithCredError(500);
        initNockBind(201);
        supertest(publicServer)
          .put("/v2/service_instances/" + serviceInstanceId + "/service_bindings/" + bindingId)
          .set("Authorization", "Basic " + auth)
          .send({ "app_guid": appId, "parameters": policy })
          .expect(500)
          .expect('Content-Type', /json/)
          .expect({})
          .end(function(err, res) {
            binding.count({ where: { bindingId: bindingId } }).then(function(countRes) {
              expect(countRes).to.equal(0);
              expect(res.statusCode).to.equal(500);
              done();
            })
          });
      });

      context('when there is no policy in request', function() {
        it("return a 201", function(done) {
          initNockBindWithCred(201);
          supertest(publicServer)
            .put("/v2/service_instances/" + serviceInstanceId + "/service_bindings/" + bindingId)
            .set("Authorization", "Basic " + auth)
            .send({ "app_guid": appId })
            .expect(202)
            .expect('Content-Type', /json/)
            .expect({credentials: {}})
            .end(function(err, res) {
              expect(res.body.credentials.custom_metrics.username).to.equal('newusername');
              expect(res.body.credentials.custom_metrics.password).to.equal('newpassword');
              binding.count({ where: { bindingId: bindingId } }).then(function(countRes) {
                expect(countRes).to.equal(1);
                done();
              })
            });
        });
      });
      context("when the api server returns error", function() {
        it("return a 400", function(done) {
          initNockApiServerBindError(400);
          supertest(publicServer)
            .put("/v2/service_instances/" + serviceInstanceId + "/service_bindings/" + bindingId)
            .set("Authorization", "Basic " + auth)
            .send({ "app_guid": appId, "parameters": policy })
            .expect(400)
            .expect('Content-Type', /json/)
            .expect({'description': VALIDATION_ERROR_FROM_API_SERVER})
            .end(function(err, res) {
              console.log(err);
              binding.count({ where: { bindingId: bindingId } }).then(function(countRes) {
                expect(countRes).to.equal(0);
                done();
              })
            });
        });
        it("return a 500", function(done) {
          initNockBind(500);
          supertest(publicServer)
            .put("/v2/service_instances/" + serviceInstanceId + "/service_bindings/" + bindingId)
            .set("Authorization", "Basic " + auth)
            .send({ "app_guid": appId, "parameters": policy })
            .expect(500)
            .expect('Content-Type', /json/)
            .expect({})
            .end(function(err, res) {
              binding.count({ where: { bindingId: bindingId } }).then(function(countRes) {
                expect(countRes).to.equal(0);
                done();
              })
            });
        });
        context('when the api server returns other error than 400, 500', function() {
          it('returns a 500', function(done) {
            initNockBind(300);
            supertest(publicServer)
              .put("/v2/service_instances/" + serviceInstanceId + "/service_bindings/" + bindingId)
              .set("Authorization", "Basic " + auth)
              .set('Accept', 'application/json')
              .send({ "app_guid": appId, "parameters": policy })
              .expect(500)
              .expect('Content-Type', /json/)
              .expect({})
              .end(function(err, res) {
                binding.count({ where: { bindingId: bindingId } }).then(function(countRes) {
                  expect(countRes).to.equal(0);
                  done();
                })
              });
          });
        });
      });
    });

    context('when a binding exists for the app', function() {

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

      context('when an app is already bound', function() {
        it('returns a 409', function(done) {
          supertest(publicServer)
            .put("/v2/service_instances/" + serviceInstanceId + "/service_bindings/" + bindingId)
            .set("Authorization", "Basic " + auth)
            .set('Accept', 'application/json')
            .send({ "app_guid": appId, "parameters": policy })
            .expect(409)
            .expect('Content-Type', /json/)
            .expect({})
            .end(function(err, res) {
              binding.count({ where: { bindingId: bindingId } }).then(function(countRes) {
                expect(countRes).to.equal(1);
                done();
              })
            });
        });
      });
      context('when the binding id already exists', function() {
        beforeEach(function(done) {
          serviceInstance.create(service_condition2).then(function(result) {
            done();
          });
        });
        it('returns a 409', function(done) {
          supertest(publicServer)
            .put("/v2/service_instances/" + serviceInstanceId2 + "/service_bindings/" + bindingId)
            .set("Authorization", "Basic " + auth)
            .set('Accept', 'application/json')
            .send({ "app_guid": appId2, "parameters": policy })
            .expect(409)
            .expect('Content-Type', /json/)
            .expect({})
            .end(function(err, res) {
              binding.count({ where: { bindingId: bindingId } }).then(function(countRes) {
                expect(countRes).to.equal(1);
                done();
              })
            });
        });
      });
      context('when an service instance has already bound to an application', function() {
        beforeEach(function(done) {
          serviceInstance.create(service_condition2).then(function(result) {
            done();
          });
        });
        it('returns a 409', function(done) {
          supertest(publicServer)
            .put("/v2/service_instances/" + serviceInstanceId2 + "/service_bindings/" + bindingId)
            .set("Authorization", "Basic " + auth)
            .set('Accept', 'application/json')
            .send({ "app_guid": appId, "parameters": policy })
            .expect(409)
            .expect({ "description": messageUtil.getMessage("DUPLICATE_BIND", { "applicationId": appId }) })
            .end(function(err, res) {
              binding.count({ where: { bindingId: bindingId } }).then(function(countRes) {
                expect(countRes).to.equal(1);
                done();
              })
            });
        });
      });
    });
    context('when serviceInstanceId is undefined', function() {
      it("return a 404", function(done) {
        supertest(publicServer)
          .put("/v2/service_instances//service_bindings/" + bindingId)
          .set("Authorization", "Basic " + auth)
          .send({ "app_guid": appId })
          .expect(404)
          .end(function(err, res) {
            binding.count({ where: { bindingId: bindingId } }).then(function(countRes) {
              expect(countRes).to.equal(0);
              done();
            })
          });
      });
    });
    context('when serviceInstanceId is blank space', function() {
      it("return a 400", function(done) {
        supertest(publicServer)
          .put("/v2/service_instances/   /service_bindings/" + bindingId)
          .set("Authorization", "Basic " + auth)
          .send({ "app_guid": appId })
          .expect(400)
          .end(function(err, res) {
            binding.count({ where: { bindingId: bindingId } }).then(function(countRes) {
              expect(countRes).to.equal(0);
              done();
            })
          });
      });
    });
    context('when bindingId is undefined', function() {
      it("return a 404", function(done) {
        supertest(publicServer)
          .put("/v2/service_instances/" + serviceInstanceId + "/service_bindings/")
          .set("Authorization", "Basic " + auth)
          .send({ "app_guid": appId })
          .expect(404)
          .end(function(err, res) {
            binding.count({ where: { bindingId: bindingId } }).then(function(countRes) {
              expect(countRes).to.equal(0);
              done();
            })
          });
      });
    });
    context('when bindingId is blank space', function() {
      it("return a 404", function(done) {
        supertest(publicServer)
          .put("/v2/service_instances/" + serviceInstanceId + "/service_bindings/" + "   ")
          .set("Authorization", "Basic " + auth)
          .send({ "app_guid": appId })
          .expect(404)
          .end(function(err, res) {
            binding.count({ where: { bindingId: bindingId } }).then(function(countRes) {
              expect(countRes).to.equal(0);
              done();
            })
          });
      });
    });

    context('when appId is undefined', function() {
      it("return a 400", function(done) {
        supertest(publicServer)
          .put("/v2/service_instances/" + serviceInstanceId + "/service_bindings/" + bindingId)
          .set("Authorization", "Basic " + auth)
          .expect(400)
          .end(function(err, res) {
            binding.count({ where: { bindingId: bindingId } }).then(function(countRes) {
              expect(countRes).to.equal(0);
              done();
            })
          });
      });
    });
    context('when appId is blank space', function() {
      it("return a 404", function(done) {
        supertest(publicServer)
          .put("/v2/service_instances/" + serviceInstanceId + "/service_bindings/" + bindingId)
          .set("Authorization", "Basic " + auth)
          .send({ "app_guid": "  " })
          .expect(400)
          .end(function(err, res) {
            binding.count({ where: { bindingId: bindingId } }).then(function(countRes) {
              expect(countRes).to.equal(0);
              done();
            })
          });
      });
    });

    context('when the service instance does not exist', function() {
      it("return a 404", function(done) {
        supertest(publicServer)
          .put("/v2/service_instances/" + serviceInstanceId2 + "/service_bindings/" + bindingId)
          .set("Authorization", "Basic " + auth)
          .send({ "app_guid": appId, "parameters": policy })
          .expect(404)
          .expect('Content-Type', /json/)
          .expect({ "description": messageUtil.getMessage("SERVICEINSTANCE_NOT_EXIST", { "serviceInstanceId": serviceInstanceId2 }) })
          .end(function(err, res) {
            binding.count({ where: { bindingId: bindingId } }).then(function(countRes) {
              expect(countRes).to.equal(0);
              done();
            })
          });
      });
    });
  });
  context('Unbind service', function() {

    context('when a binding exists for the app', function() {
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
      it("it deletes the binding", function(done) {
        initNockUnBind(200);
        initNockUnbindWithCred(200);
        supertest(publicServer)
          .delete("/v2/service_instances/" + serviceInstanceId + "/service_bindings/" + bindingId)
          .set("Authorization", "Basic " + auth)
          .expect(200)
          .expect('Content-Type', /json/)
          .expect({})
          .end(function(err, res) {
            binding.count({ where: { bindingId: bindingId } }).then(function(countRes) {
              expect(countRes).to.equal(0);
              done();
            })
          });
      });

      it("it could not delete the binding because of credential deletion failure", function(done) {
        initNockUnBind(200);
        initNockUnbindWithCredError(500);
        supertest(publicServer)
          .delete("/v2/service_instances/" + serviceInstanceId + "/service_bindings/" + bindingId)
          .set("Authorization", "Basic " + auth)
          .expect(500)
          .expect('Content-Type', /json/)
          .expect({})
          .end(function(err, res) {
            binding.count({ where: { bindingId: bindingId } }).then(function(countRes) {
              expect(countRes).to.equal(1);
              expect(res.statusCode).to.equal(500);
              done();
            })
          });
      });

      context("when the api server returns error", function() {
        context("when the api server returns a 400", function() {
          it("return a 400", function(done) {
            initNockUnBind(400);
            supertest(publicServer)
              .delete("/v2/service_instances/" + serviceInstanceId + "/service_bindings/" + bindingId)
              .set("Authorization", "Basic " + auth)
              .expect(500)
              .expect('Content-Type', /json/)
              .expect({})
              .end(function(err, res) {
                binding.count({ where: { bindingId: bindingId } }).then(function(countRes) {
                  expect(countRes).to.equal(1);
                  done();
                })
              });
          });
        });

        context("when the api server returns a 404", function() {
          it("return a 404", function(done) {
            initNockUnBind(404);
            supertest(publicServer)
              .delete("/v2/service_instances/" + serviceInstanceId + "/service_bindings/" + bindingId)
              .set("Authorization", "Basic " + auth)
              .expect(200)
              .expect('Content-Type', /json/)
              .expect({})
              .end(function(err, res) {
                binding.count({ where: { bindingId: bindingId } }).then(function(countRes) {
                  expect(countRes).to.equal(0);
                  done();
                })
              });
          });
        });

        context("when the api server returns a 500", function() {
          it("return a 500", function(done) {
            initNockUnBind(500);
            supertest(publicServer)
              .delete("/v2/service_instances/" + serviceInstanceId + "/service_bindings/" + bindingId)
              .set("Authorization", "Basic " + auth)
              .expect(500)
              .expect('Content-Type', /json/)
              .expect({})
              .end(function(err, res) {
                binding.count({ where: { bindingId: bindingId } }).then(function(countRes) {
                  expect(countRes).to.equal(1);
                  done();
                })
              });
          });
        });

        context('when the api server returns other error than 400, 500', function() {
          it('returns a 500', function(done) {
            initNockUnBind(300);
            supertest(publicServer)
              .delete("/v2/service_instances/" + serviceInstanceId + "/service_bindings/" + bindingId)
              .set("Authorization", "Basic " + auth)
              .set('Accept', 'application/json')
              .expect(500)
              .expect('Content-Type', /json/)
              .expect({})
              .end(function(err, res) {
                binding.count({ where: { bindingId: bindingId } }).then(function(countRes) {
                  expect(countRes).to.equal(1);
                  done();
                })
              });
          });
        });
      });
    });
    context('when serviceInstanceId is undefined', function() {
      it("return a 404", function(done) {
        supertest(publicServer)
          .put("/v2/service_instances//service_bindings/" + bindingId)
          .set("Authorization", "Basic " + auth)
          .expect(404)
          .end(function(err, res) {
            binding.count({ where: { bindingId: bindingId } }).then(function(countRes) {
              expect(countRes).to.equal(0);
              done();
            })
          });
      });
    });
    context('when serviceInstanceId is blank space', function() {
      it("return a 400", function(done) {
        supertest(publicServer)
          .put("/v2/service_instances/   /service_bindings/" + bindingId)
          .set("Authorization", "Basic " + auth)
          .expect(400)
          .end(function(err, res) {
            binding.count({ where: { bindingId: bindingId } }).then(function(countRes) {
              expect(countRes).to.equal(0);
              done();
            })
          });
      });
    });

    context('when bindingId is undefined', function() {
      it("return a 404", function(done) {
        supertest(publicServer)
          .put("/v2/service_instances/" + serviceInstanceId + "/service_bindings/")
          .set("Authorization", "Basic " + auth)
          .expect(404)
          .end(function(err, res) {
            binding.count({ where: { bindingId: bindingId } }).then(function(countRes) {
              expect(countRes).to.equal(0);
              done();
            })
          });
      });
    });
    context('when bindingId is blank space', function() {
      it("return a 404", function(done) {
        supertest(publicServer)
          .put("/v2/service_instances/" + serviceInstanceId + "/service_bindings/" + "   ")
          .set("Authorization", "Basic " + auth)
          .expect(404)
          .end(function(err, res) {
            binding.count({ where: { bindingId: bindingId } }).then(function(countRes) {
              expect(countRes).to.equal(0);
              done();
            })
          });
      });
    });
    context('when the binding does not exist for the app', function() {
      it('return 410', function(done) {
        supertest(publicServer)
          .delete("/v2/service_instances/" + serviceInstanceId + "/service_bindings/" + bindingId)
          .set("Authorization", "Basic " + auth)
          .expect(410)
          .expect('Content-Type', /json/)
          .expect({})
          .end(function(err, res) {
            binding.count({ where: { bindingId: bindingId } }).then(function(countRes) {
              expect(countRes).to.equal(0);
              done();
            })
          });
      });
    });
  });
});
