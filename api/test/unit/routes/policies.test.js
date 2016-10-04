'use strict';

var request = require('supertest');
var expect = require('chai').expect;
var fs = require('fs');
var app = require('../../../app.js');
var policy = require('../../../lib/models')().policy_json;
var logger = require('../../../lib/log/logger');
var nock = require('nock');
var schedulerURI = process.env.SCHEDULER_URI ;

describe('Routing Policy Creation', function() {
  var fakePolicy;

  before(function() {
    fakePolicy = JSON.parse(fs.readFileSync(__dirname+'/../fakePolicy.json', 'utf8'));
  })

  beforeEach(function() {
    return policy.truncate();
  });

  it('should create a policy for app id 12345', function(done) {
    nock(schedulerURI)
    .put('/v2/schedules/12345')
    .reply(200);
    request(app)
    .put('/v1/policies/12345')
    .send(fakePolicy)
    .end(function(error,result) {
      expect(result.statusCode).to.equal(201);
      expect(result.headers.location).exist;
      expect(result.headers.location).to.be.equal('/v1/policies/12345');
      expect(result.body.success).to.equal(true);
      expect(result.body.error).to.be.null;
      expect(result.body.result.policy_json).eql(fakePolicy);
      done();
    });
  });

  it('should fail to create a policy for validation error in scheduler for app id 12346', function(done) {
    nock(schedulerURI)
    .put('/v2/schedules/12346')
    .reply(400);
    request(app)
    .put('/v1/policies/12346')
    .send(fakePolicy)
    .end(function(error,result) {
      expect(result.statusCode).to.equal(400);
      expect(result.body.error.message).eql('Failed to create schedules due to validation error in scheduler');
      expect(result.body.success).eql(false);
      done();
    });
  });

  it('should fail to create a policy for internal error in scheduler for app id 12347', function(done) {
  var mockError = { 'message':'Failed to create schedules due to an internal' + 
        ' error in scheduler','details':'fake body' };
    nock(schedulerURI)
    .put('/v2/schedules/12347')
    .replyWithError(mockError);
    request(app)
    .put('/v1/policies/12347')
    .send(fakePolicy)
    .end(function(error,result) {
      expect(result.statusCode).to.equal(500);
      expect(result.body.error.message).eql('Failed to create schedules due to an internal error in scheduler');
      expect(result.body.error.details).eql('fake body');
      expect(result.body.success).eql(false);
      done();
    });
  });
  
  context('when a policy already exists' ,function() {
    beforeEach(function(done) {
      nock(schedulerURI)
      .put('/v2/schedules/12345')
      .reply(200);
      request(app)
      .put('/v1/policies/12345')
      .send(fakePolicy).end(done)

    });
    it('should update the existing policy for app id 12345', function(done) {
      nock(schedulerURI)
      .put('/v2/schedules/12345')
      .reply(204);
      request(app)
      .put('/v1/policies/12345')
      .send(fakePolicy)
      .end(function(error,result) {
        expect(result.statusCode).to.equal(200);
        expect(result.body.success).to.equal(true);
        expect(result.body.result[0].policy_json).eql(fakePolicy);
        expect(result.body.error).to.be.null;
        done();
      });
    });

    it('should successfully get the details of the policy with app id 12345',function(done){
      request(app)
      .get('/v1/policies/12345')
      .end(function(error,result) {
        expect(result.statusCode).to.equal(200);
        expect(result.body).to.deep.equal(fakePolicy);
        done();
      });    
    });
    it('should successfully delete the policy with app id 12345',function(done){
      nock(schedulerURI)
      .delete('/v2/schedules/12345')
      .reply(200);
      
      request(app)
      .delete('/v1/policies/12345')
      .end(function(error,result) {
        expect(result.statusCode).to.equal(200);
        done();
      });
    });

    it('should fail to delete the policy with app id 12345 due to internal server error',function(done){
        nock(schedulerURI)
        .delete('/v2/schedules/12345')
        .reply(500);
        
        request(app)
        .delete('/v1/policies/12345')
        .end(function(error,result) {
          expect(result.statusCode).to.equal(500);
          done();
        });
      });

  });

  context('when policy does not exists' ,function() {
    it('should fail to delete a non existing policy with app id 12345',function(done){
      nock(schedulerURI)
      .delete('/v2/schedules/12345')
      .reply(404);
    
      request(app)
      .delete('/v1/policies/12345')
      .end(function(error,result) {
        expect(result.statusCode).to.equal(404);
        done();
      });    
    });
    
    it('should fail to get the details of a non existing policy with app id 12345',function(done){
      request(app)
      .get('/v1/policies/12345')
      .end(function(error,result) {
        expect(result.statusCode).to.equal(404);
        expect(result.body).eql({});
        done();
      });    
    });

  });
});
