'use strict';

process.env.NODE_TLS_REJECT_UNAUTHORIZED = '0';
var request = require('supertest');
var expect = require('chai').expect;
var fs = require('fs');
var path = require('path');
var uuidV4 = require('uuid/v4');
var settings = require(path.join(__dirname, '../../../lib/config/setting.js'))((JSON.parse(
  fs.readFileSync(path.join(__dirname, '../../../config/settings.json'), 'utf8'))));
var relativePath = path.relative(process.cwd(), path.join(__dirname, "../../../../test-certs"));
var testSetting = require(path.join(__dirname, '../test.helper.js'))(relativePath, settings);
var API = require('../../../app.js');
var HttpStatus = require('http-status-codes');
var app;
var publicApp;
var healthApp;
var servers;
var policy = require('../../../lib/models')(settings.db).policy_json;
var logger = require('../../../lib/log/logger');
var nock = require('nock');
var schedulerURI = testSetting.scheduler.uri;
var theUserId = "the-user-id";
var theUserToken = "token-type the-user-token"

function mockCF(count) {
  nock("https://api.bosh-lite.com")
    .get("/v2/info")
    .times(count)
    .reply(HttpStatus.OK, { "token_endpoint": "https://uaa.bosh-lite.com" });
  nock("https://uaa.bosh-lite.com")
    .post("/check_token?token=the-user-token")
    .times(count)
    .reply(HttpStatus.OK, { "scope": ["cloud_controller.read","password.write","cloud_controller.write","openid","network.admin","network.write","uaa.user"] });
  nock("https://uaa.bosh-lite.com")
    .get("/userinfo")
    .times(count)
    .reply(HttpStatus.OK, { "user_id": theUserId });
  nock("https://api.bosh-lite.com")
    .get(/\/v2\/users\/.+\/spaces\?.+/)
    .times(count)
    .reply(HttpStatus.OK, {
      "total_results": 1,
      "total_pages": 1,
      "prev_url": null,
      "next_url": null
    });
}

function mockSchedulerPut(count, statusCode) {
  nock(schedulerURI)
    .put(/\/v1\/apps\/.+\/schedules/)
    .times(count)
    .query({ 'guid': /.*/ })
    .reply(statusCode);
}

function mockSchedulerPutError(count, err) {
  nock(schedulerURI)
    .put(/\/v1\/apps\/.+\/schedules/)
    .times(count)
    .query({ 'guid': /.*/ })
    .replyWithError(err);
}
function mockSchedulerPutMessage(count, statusCode, message) {
  nock(schedulerURI)
    .put(/\/v1\/apps\/.+\/schedules/)
    .times(count)
    .query({ 'guid': /.*/ })
    .reply(statusCode, message);
}

function mockSchedulerDelete(count, statusCode) {
  nock(schedulerURI)
    .delete(/\/v1\/apps\/.+\/schedules/)
    .times(count)
    .reply(statusCode);

}

function mockBroker200WithBinding(count) {

  if (testSetting.serviceOffering.enabled) {
    nock(testSetting.serviceOffering.serviceBroker.uri)
      .get(/\/v1\/apps\/.+\/service_bindings/)
      .times(count)
      .reply(HttpStatus.OK, {
        "binding": {
          "bindingId": "an-binding-id",
          "appId": "an-app-id",
          "serviceInstanceId": "an-service-instance-id"
        }
      });
  }

}

function mockBroker200WithoutBinding(count) {
  if (testSetting.serviceOffering.enabled) {
    nock(testSetting.serviceOffering.serviceBroker.uri)
      .get(/\/v1\/apps\/.+\/service_bindings/)
      .times(count)
      .reply(HttpStatus.OK, { "binding": null });
  }
}

describe('Routing Policy Creation', function () {
  var fakePolicy;


  before(function () {
    fakePolicy = JSON.parse(fs.readFileSync(__dirname + '/../fakePolicy.json', 'utf8'));
    servers = API(testSetting, function () { });
    app = servers.internalServer;
    publicApp = servers.publicServer;
    healthApp = servers.healthServer;
  })
  after(function(done) {
    app.close(function() {
      publicApp.close(function(){
        healthApp.close(done);
      });
    });
  })
  beforeEach(function () {
    nock.cleanAll();
    mockCF(3);

    return policy.truncate();
  });

  context("create policy", function () {
    context("when binding info exists", function () {
      beforeEach(function () {
        mockBroker200WithBinding(1);
        mockSchedulerPut(1, 200);
      });
      it('should create a policy for app id 12345', function (done) {
        request(publicApp)
          .put('/v1/apps/12345/policy')
          .set("Authorization", theUserToken)
          .send(fakePolicy)
          .end(function (error, result) {
            expect(result.statusCode).to.equal(201);
            expect(result.headers.location).exist;
            expect(result.headers.location).to.be.equal('/v1/apps/12345/policy');
            expect(result.body).eql(fakePolicy);
            done();
          });
      });
    });

    context("when binding info does not exist", function () {
      beforeEach(function () {
        mockBroker200WithoutBinding(1);
      });
      it('should return 403', function (done) {
        if (testSetting.serviceOffering.enabled) {
          request(publicApp)
            .put('/v1/apps/12345/policy')
            .set("Authorization", theUserToken)
            .send(fakePolicy)
            .end(function (error, result) {
              expect(result.statusCode).to.equal(403);
              done();
            });
        } else {
          this.skip();
        }
      });
    });

    context("CSP", function () {
      beforeEach(function () {
        mockBroker200WithBinding(1);
        mockSchedulerPut(1, 200);
      });
      it('dummy call to test CSP response headers', function (done) {
        request(publicApp)
          .put('/v1/apps/12344/policy')
          .set("Authorization", theUserToken)
          .send(fakePolicy)
          .end(function (error, result) {
            expect(result.statusCode).to.equal(201);
            expect(result.headers.location).exist;
            expect(result.headers.location).to.be.equal('/v1/apps/12344/policy');
            expect(result.body).eql(fakePolicy);
            expect(result.headers).to.have.deep.property('content-security-policy', 'default-src \'self\'; script-src \'self\'');
            expect(result.headers).to.have.deep.property('x-content-type-options', 'nosniff')
            expect(result.headers).to.have.deep.property('cache-control', 'no-store, no-cache, must-revalidate, proxy-revalidate')
            done();
          });
      });
    });

    context("when there is validation error in scheduler", function () {
      beforeEach(function () {
        mockBroker200WithBinding(1);
        mockSchedulerPutMessage(1, 400, 'Specific Date Schedule start_date_time should be after current date');
      });
      it('should fail to create a policy for app id 12346', function (done) {
        request(publicApp)
          .put('/v1/apps/12346/policy')
          .set("Authorization", theUserToken)
          .send(fakePolicy)
          .end(function (error, result) {
            expect(result.statusCode).to.equal(400);
            expect(result.body.error).eql('Failed to create schedules due to validation error in scheduler, details:\nSpecific Date Schedule start_date_time should be after current date');
            done();
          });
      });
    });

    context("when there is internal error in scheduler", function () {
      beforeEach(function () {
        var mockError = {
          'message': 'Failed to create schedules due to an internal error in scheduler',
        };
        mockBroker200WithBinding(1);
        mockSchedulerPutError(1, mockError)
      });
      it('should fail to create a policy for app id 12347', function (done) {
        request(publicApp)
          .put('/v1/apps/12347/policy')
          .set("Authorization", theUserToken)
          .send(fakePolicy)
          .end(function (error, result) {
            expect(result.statusCode).to.equal(500);
            expect(result.body.error).eql('Failed to create schedules due to an internal error in scheduler');
            done();
          });
      });
    });

  }); //end of create policy

  context("update policy", function () {
    beforeEach(function (done) {
      mockBroker200WithBinding(1);
      mockSchedulerPut(1, 200);
      request(publicApp)
        .put('/v1/apps/12345/policy')
        .set("Authorization", theUserToken)
        .send(fakePolicy).end(function (error, result) {
          done();
        })
    });
    context("when binding info exists", function () {
      beforeEach(function () {
        mockBroker200WithBinding(1);
        mockSchedulerPut(1, 204);
      });
      it('should update the existing policy for app id 12345', function (done) {
        request(publicApp)
          .put('/v1/apps/12345/policy')
          .set("Authorization", theUserToken)
          .send(fakePolicy)
          .end(function (error, result) {
            expect(result.statusCode).to.equal(200);
            expect(result.body).eql(fakePolicy);
            done();
          });
      });
    });
    context("when binding info does not exist", function () {
      beforeEach(function () {
        mockBroker200WithoutBinding(1);
        mockSchedulerPut(1, 204);
      });
      it('should return 403', function (done) {
        if (testSetting.serviceOffering.enabled) {
          request(publicApp)
            .put('/v1/apps/12345/policy')
            .set("Authorization", theUserToken)
            .send(fakePolicy)
            .end(function (error, result) {
              expect(result.statusCode).to.equal(403);
              done();
            });
        } else {
          this.skip();
        }
      });
    });
  });


  context('delete policy', function () {

    beforeEach(function (done) {
      mockBroker200WithBinding(1);
      mockSchedulerPut(1, 200);
      request(publicApp)
        .put('/v1/apps/12345/policy')
        .set("Authorization", theUserToken)
        .send(fakePolicy).end(function (error, result) {
          done();
        })
    });

    context("when binding info exists", function () {
      beforeEach(function () {
        mockBroker200WithBinding(1);
        mockSchedulerDelete(1, 200);
      });
      it('should successfully delete the policy with app id 12345', function (done) {
        request(publicApp)
          .delete('/v1/apps/12345/policy')
          .set("Authorization", theUserToken)
          .expect(200)
          .end(function (error) {
            expect(error).to.be.null;
            done();
          });
      });
    });

    context("when binding info does not exist", function () {
      beforeEach(function () {
        mockBroker200WithoutBinding(1);
        mockSchedulerPut(1, 200);
      });
      it('should return 403', function (done) {
        if (testSetting.serviceOffering.enabled) {
          request(publicApp)
            .delete('/v1/apps/12345/policy')
            .set("Authorization", theUserToken)
            .expect(403)
            .end(function (error) {
              expect(error).to.be.null;
              done();
            });
        } else {
          this.skip();
        }
      });
    });
    context('when policy does not exist', function () {
      beforeEach(function (done) {
        mockBroker200WithBinding(2);
        mockSchedulerDelete(1, 200);
        request(publicApp)
          .delete('/v1/apps/12345/policy')
          .set("Authorization", theUserToken)
          .end(function (error, result) {
            expect(result.statusCode).to.equal(200);
            done();
          });
      });

      it('should return 404 while deleting policy with app id 12345', function (done) {
        request(publicApp)
          .delete('/v1/apps/12345/policy')
          .set("Authorization", theUserToken)
          .end(function (error, result) {
            expect(result.statusCode).to.equal(404);
            done();
          });
      });
    });
    context("when there is internal error in scheduler", function () {
      beforeEach(function () {
        mockBroker200WithBinding(1);
        mockSchedulerDelete(1, 500)
      });
      it('should fail to delete the policy with app id 12345 due to internal server error', function (done) {
        request(publicApp)
          .delete('/v1/apps/12345/policy')
          .set("Authorization", theUserToken)
          .end(function (error, result) {
            expect(result.statusCode).to.equal(500);
            done();
          });
      });
    });


  }); //end of delete policy

  context('get policy', function () {
    context('when policy exists', function () {
      beforeEach(function (done) {
        mockBroker200WithBinding(1);
        mockSchedulerPut(1, 200);
        request(publicApp)
          .put('/v1/apps/12345/policy')
          .set("Authorization", theUserToken)
          .send(fakePolicy).end(function (error, result) {
            done();
          })
      });
      context("when binding info exists", function () {
        beforeEach(function (done) {
          mockBroker200WithBinding(1);
          done();
        });
        it('should successfully get the details of the policy with app id 12345', function (done) {
          request(publicApp)
            .get('/v1/apps/12345/policy')
            .set("Authorization", theUserToken)
            .end(function (error, result) {
              expect(result.statusCode).to.equal(200);
              expect(result.body).to.deep.equal(fakePolicy);
              done();
            });

        });
      });

      context("when binding info does not exist", function () {
        beforeEach(function (done) {
          mockBroker200WithoutBinding(1);
          done();
        });
        it('should return 403', function (done) {
          if (testSetting.serviceOffering.enabled) {
            request(publicApp)
              .get('/v1/apps/12345/policy')
              .set("Authorization", theUserToken)
              .end(function (error, result) {
                expect(result.statusCode).to.equal(403);
                done();
              });
          } else {
            this.skip();
          }
        });
      });
    });

    context('when policy does not exist', function () {
      beforeEach(function () {
        mockBroker200WithBinding(1);
      });

      it('should fail to get the details of a non existing policy with app id 12345', function (done) {
        request(publicApp)
          .get('/v1/apps/12345/policy')
          .set("Authorization", theUserToken)
          .end(function (error, result) {
            expect(result.statusCode).to.equal(404);
            expect(result.body).eql({});
            done();
          });
      });
    });
  }); //end of get policy
});