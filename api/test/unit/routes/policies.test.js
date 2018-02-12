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
var testSetting = require(path.join(__dirname, '../test.helper.js'))(relativePath,settings);
var API = require('../../../app.js');
var app;
var publicApp;
var servers;
var policy = require('../../../lib/models')(settings.db).policy_json;
var logger = require('../../../lib/log/logger');
var nock = require('nock');
var schedulerURI = testSetting.scheduler.uri ;
var serviceBrokerURI = testSetting.serviceBroker.uri;
var theUserId = "the-user-id";
function mockCF(count){
  nock("https://api.bosh-lite.com")
    .get("/v2/info")
    .times(count)
    .reply(200, { "authorization_endpoint": "https://uaa.bosh-lite.com" });
    nock("https://uaa.bosh-lite.com")
    .get("/userinfo")
    .times(count)
    .reply(200, { "user_id": theUserId });
    nock("https://api.bosh-lite.com")
    .get(/\/v2\/users\/.+\/spaces\?.+/)
    .times(count)
    .reply(200, {
      "total_results": 1,
      "total_pages": 1,
      "prev_url": null,
      "next_url": null
    });
}
function mockSchedulerPut(count, statusCode){
  nock(schedulerURI)
    .put(/\/v2\/schedules\/.+/)
    .times(count)
    .query({ 'guid': /.*/ })
    .reply(statusCode);
}
function mockSchedulerPutError(count, err){
  nock(schedulerURI)
    .put(/\/v2\/schedules\/.+/)
    .times(count)
    .query({ 'guid': /.*/ })
    .replyWithError(err);
}
function mockSchedulerDelete(count, statusCode){
  nock(schedulerURI)
    .delete(/\/v2\/schedules\/.+/)
    .times(count)
    .query({ 'guid': /.*/ })
    .reply(statusCode);
}
function mockBroker200(count){
  nock(serviceBrokerURI)
    .get(/\/v1\/apps\/.+\/service_bindings/)
    .times(count)
    .reply(200, { "message": "binding_info_exists" });
}
function mockBroker404(count){
  nock(serviceBrokerURI)
    .get(/\/v1\/apps\/.+\/service_bindings/)
    .times(count)
    .reply(404, { "message": "binding_info_not_found" });
}

describe('Routing Policy Creation', function() {
  var fakePolicy;
  

  before(function() {
    fakePolicy = JSON.parse(fs.readFileSync(__dirname+'/../fakePolicy.json', 'utf8'));
    servers = API(testSetting, function(){});
    app = servers.internalServer;
    publicApp = servers.publicServer;
  })
  after(function(done){
    app.close(function(){
      publicApp.close(done);
    });
  })
  beforeEach(function() {
    nock.cleanAll();
    mockCF(3);

    return policy.truncate();
  });
  
  context("create policy", function() {
    context("when binding info exists", function() {
      beforeEach(function() {
        mockBroker200(1);
        mockSchedulerPut(1, 200);
      });
      it('should create a policy for app id 12345', function(done) {
        request(publicApp)
          .put('/v1/apps/12345/policy')
          .set("Authorization","fake-token")
          .send(fakePolicy)
          .end(function(error, result) {
            expect(result.statusCode).to.equal(201);
            expect(result.headers.location).exist;
            expect(result.headers.location).to.be.equal('/v1/apps/12345/policy');
            expect(result.body.success).to.equal(true);
            expect(result.body.error).to.be.null;
            expect(result.body.result.policy_json).eql(fakePolicy);
            expect(result.body.result.guid).to.not.be.null;
            done();
          });
      });
    });

    context("when binding info does not exist", function() {
      beforeEach(function() {
        mockBroker404(1);
      });
      it('should return 403', function(done) {
        request(publicApp)
          .put('/v1/apps/12345/policy')
          .set("Authorization","fake-token")
          .send(fakePolicy)
          .end(function(error, result) {
            expect(result.statusCode).to.equal(403);
            done();
          });
      });
    });

    context("CSP", function(){
      beforeEach(function(){
        mockBroker200(1);
        mockSchedulerPut(1, 200);
      });
      it('dummy call to test CSP response headers', function(done) {
        request(publicApp)
          .put('/v1/apps/12344/policy')
          .set("Authorization","fake-token")
          .send(fakePolicy)
          .end(function(error, result) {
            expect(result.statusCode).to.equal(201);
            expect(result.headers.location).exist;
            expect(result.headers.location).to.be.equal('/v1/apps/12344/policy');
            expect(result.body.success).to.equal(true);
            expect(result.body.error).to.be.null;
            expect(result.body.result.policy_json).eql(fakePolicy);
            expect(result.body.result.guid).to.not.be.null;
            expect(result.headers).to.have.deep.property('content-security-policy', 'default-src \'self\'; script-src \'self\'');
            expect(result.headers).to.have.deep.property('x-content-type-options', 'nosniff')
            expect(result.headers).to.have.deep.property('cache-control', 'no-store, no-cache, must-revalidate, proxy-revalidate')
            done();
          });
      });
    });
    
    context("when there is validation error in scheduler", function(){
      beforeEach(function(){
        mockBroker200(1);
        mockSchedulerPut(1, 400);
      });
      it('should fail to create a policy for app id 12346', function(done) {
        request(publicApp)
          .put('/v1/apps/12346/policy')
          .set("Authorization","fake-token")
          .send(fakePolicy)
          .end(function(error, result) {
            expect(result.statusCode).to.equal(400);
            expect(result.body.error.message).eql('Failed to create schedules due to validation error in scheduler');
            expect(result.body.success).eql(false);
            done();
          });
      });
    });
    
    context("when there is internal error in scheduler", function(){
      beforeEach(function(){
        var mockError = {
          'message': 'Failed to create schedules due to an internal' +
            ' error in scheduler',
          'details': 'fake body'
        };
        mockBroker200(1);
        mockSchedulerPutError(1, mockError)
      });
      it('should fail to create a policy for app id 12347', function(done) {
        request(publicApp)
          .put('/v1/apps/12347/policy')
          .set("Authorization","fake-token")
          .send(fakePolicy)
          .end(function(error, result) {
            expect(result.statusCode).to.equal(500);
            expect(result.body.error.message).eql('Failed to create schedules due to an internal error in scheduler');
            expect(result.body.error.details).eql('fake body');
            expect(result.body.success).eql(false);
            done();
          });
      });
    });
    
  });//end of create policy

  context("update policy", function(){
    var initialGuid;

    beforeEach(function(done) {
      mockBroker200(1);
      mockSchedulerPut(1,200);
      request(publicApp)
      .put('/v1/apps/12345/policy')
      .set("Authorization","fake-token")
      .send(fakePolicy).end(function(error, result) {
        initialGuid = result.body.result.guid;
        done();
      })
    });
    context("when binding info exists", function(){
      beforeEach(function() {
        mockBroker200(1);
        mockSchedulerPut(1,204);
      });
      it('should update the existing policy for app id 12345', function(done) {
        request(publicApp)
        .put('/v1/apps/12345/policy')
        .set("Authorization","fake-token")
        .send(fakePolicy)
        .end(function(error,result) {
          expect(result.statusCode).to.equal(200);
          expect(result.body.success).to.equal(true);
          expect(result.body.result[0].policy_json).eql(fakePolicy);
          expect(result.body.error).to.be.null;
          expect(result.body.result[0].guid).to.not.eql(initialGuid);
          done();
        });
      });
    });
    context("when binding info does not exist", function(){
      beforeEach(function() {
        mockBroker404(1);
        mockSchedulerPut(1,204);
      });
      it('should return 403', function(done) {
        request(publicApp)
        .put('/v1/apps/12345/policy')
        .set("Authorization","fake-token")
        .send(fakePolicy)
        .end(function(error,result) {
          expect(result.statusCode).to.equal(403);
          done();
        });
      });
    });
  });
  

  context('delete policy' ,function() {
    var initialGuid;

    beforeEach(function(done) {
      mockBroker200(1);
      mockSchedulerPut(1,200);
      request(publicApp)
      .put('/v1/apps/12345/policy')
      .set("Authorization","fake-token")
      .send(fakePolicy).end(function(error, result) {
        initialGuid = result.body.result.guid;
        done();
      })
    });

    context("when binding info exists", function(){
      beforeEach(function() {
        mockBroker200(1);
        mockSchedulerDelete(1,200);
      });
      it('should successfully delete the policy with app id 12345',function(done){
        request(publicApp)
        .delete('/v1/apps/12345/policy')
        .set("Authorization","fake-token")
        .expect(200)
        .end(function(error) {
          expect(error).to.be.null;
          done();
        });
      });
    });

    context("when binding info does not exist", function(){
      beforeEach(function() {
        mockBroker404(1);
        mockSchedulerPut(1,200);
      });
      it('should return 403',function(done){
        request(publicApp)
        .delete('/v1/apps/12345/policy')
        .set("Authorization","fake-token")
        .expect(403)
        .end(function(error) {
          expect(error).to.be.null;
          done();
        });
      });
    });
    context('when policy does not exist' ,function() {
      beforeEach(function(done){
        mockBroker200(2);
        mockSchedulerDelete(1,200);
        request(publicApp)
        .delete('/v1/apps/12345/policy')
        .set("Authorization","fake-token")
        .end(function(error,result) {
          expect(result.statusCode).to.equal(200);
          done();
        });  
      });

      it('should return 404 while deleting policy with app id 12345',function(done){
        request(publicApp)
        .delete('/v1/apps/12345/policy')
        .set("Authorization","fake-token")
        .end(function(error,result) {
          expect(result.statusCode).to.equal(404);
          done();
        });    
      });
    });
    context("when there is internal error in scheduler",  function(){
      beforeEach(function(){
        mockBroker200(1);
        mockSchedulerDelete(1, 500)
      });
      it('should fail to delete the policy with app id 12345 due to internal server error',function(done){
        request(publicApp)
        .delete('/v1/apps/12345/policy')
        .set("Authorization","fake-token")
        .end(function(error,result) {
          expect(result.statusCode).to.equal(500);
          done();
        });
      });
    });
    

  });//end of delete policy

  context('get policy', function(){
    var initialGuid;
    context('when policy exists', function(){
      beforeEach(function(done) {
        mockBroker200(1);
        mockSchedulerPut(1,200);
        request(publicApp)
        .put('/v1/apps/12345/policy')
        .set("Authorization","fake-token")
        .send(fakePolicy).end(function(error, result) {
          initialGuid = result.body.result.guid;
          done();
        })
      });
      context("when binding info exists", function(){
        beforeEach(function(done){
          mockBroker200(1);
          done();
        });
        it('should successfully get the details of the policy with app id 12345',function(done){
          request(publicApp)
          .get('/v1/apps/12345/policy')
          .set("Authorization","fake-token")
          .end(function(error,result) {
            expect(result.statusCode).to.equal(200);
            expect(result.body).to.deep.equal(fakePolicy);
            done();
          });    
        });
      });

      context("when binding info does not exist", function(){
        beforeEach(function(done){
          mockBroker404(1);
          done();
        });
        it('should return 403',function(done){
          request(publicApp)
          .get('/v1/apps/12345/policy')
          .set("Authorization","fake-token")
          .end(function(error,result) {
            expect(result.statusCode).to.equal(403);
            done();
          });    
        });
      });
    });
    
    context('when policy does not exist' ,function() {
      beforeEach(function(){
        mockBroker200(1);
      });

      it('should fail to get the details of a non existing policy with app id 12345',function(done){
        request(publicApp)
        .get('/v1/apps/12345/policy')
        .set("Authorization","fake-token")
        .end(function(error,result) {
          expect(result.statusCode).to.equal(404);
          expect(result.body).eql({});
          done();
        });    
      });
    });
  });//end of get policy
});
