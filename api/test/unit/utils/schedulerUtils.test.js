'use strict';

var request = require('supertest');
var expect = require('chai').expect;
var fs = require('fs');
var policy = require('../../../lib/models')().policy_json;
var logger = require('../../../lib/log/logger');
var nock = require('nock');
var schedulerUtils = require('../../../lib/utils/schedulerUtils');
var schedulerURI = process.env.SCHEDULER_URI;

describe('Scheduler Utility functions', function() {
  var fakePolicy;

  before(function() {
    fakePolicy = JSON.parse(fs.readFileSync(__dirname+'/../fakePolicy.json', 'utf8'));
  })

  beforeEach(function() {
    return policy.truncate();
  })

  it('should create a new schedule for app id 12345',function(done){
    nock(schedulerURI)
    .put('/v2/schedules/12345')
    .reply(204);
    var mockRequest = {
        body : fakePolicy,
        params : { 'app_id' : '12345' }
    };
    schedulerUtils.createOrUpdateSchedule(mockRequest,function(error,result){
      expect(result.statusCode).to.equal(200);
      expect(error).to.be.null;
      done();
    });
  });

  it('should fail to create schedule due to error for app id 12346',function(done){
    var mockError = { 'message':'Failed to create schedules due to an internal' + 
        ' error in scheduler','details':'fake body' };
    nock(schedulerURI)
    .put('/v2/schedules/12346')
    .replyWithError(mockError);
    var mockRequest = {
        body : fakePolicy,
        params : { 'app_id' : '12346' }
    };
    schedulerUtils.createOrUpdateSchedule(mockRequest,function(error,result){
      expect(error).to.not.be.null;
      expect(result.statusCode).to.equal(500);
      expect(error).to.deep.equal(mockError);
      done();
    });
  });

  it('should fail to create a schedule due to internal validation error in scheduler module for app id 12347',function(done){
    nock(schedulerURI)
    .put('/v2/schedules/12347')
    .reply(function(uri, requestBody) {
      return [
              400,
              'fake body',
              ];
    });
    var mockRequest = {
        body : fakePolicy,
        params : { 'app_id' : '12347' }
    };
    schedulerUtils.createOrUpdateSchedule(mockRequest,function(error,result){
      expect(result.statusCode).to.equal(400);
      expect(error).to.not.be.null;
      expect(error.details).to.equal('fake body');
      expect(error.message).to.equal('Failed to create schedules due to validation error in scheduler');
      done();
    });
  });
  
  it('should fail to create a schedule due to un-accepted response code (other than 400 ) in scheduler module for app id 12348',function(done){
    nock(schedulerURI)
    .put('/v2/schedules/12348')
    .reply(function(uri, requestBody) {
      return [
              405,
              'fake body',
              ];
    });
    var mockRequest = {
        body : fakePolicy,
        params : { 'app_id' : '12348' }
    };
    schedulerUtils.createOrUpdateSchedule(mockRequest,function(error,result){
      expect(result.statusCode).to.equal(500);
      expect(error).to.not.be.null;
      expect(error.details).to.equal('fake body');
      expect(error.message).to.equal('Failed to create schedules due to an internal error in scheduler');
      console.log(error);
      done();
    });
  });

  context('when a schedules already exists' ,function() {
    beforeEach(function(done) {
      nock(schedulerURI)
      .put('/v2/schedules/12345')
      .reply(204);
      var mockRequest = {
          body : fakePolicy,
          params : { 'app_id' : '12345' }
      };
      schedulerUtils.createOrUpdateSchedule(mockRequest,function(error,result){
        done();
      });
    });
    it('should update a schedule for app id 12345',function(done){
      nock(schedulerURI)
      .put('/v2/schedules/12345')
      .reply(200);
      var mockRequest = {
          body : fakePolicy,
          params : { 'app_id' : '12345' }
      };
      schedulerUtils.createOrUpdateSchedule(mockRequest,function(error,result){
        expect(result.statusCode).to.equal(200);
        expect(error).to.be.null;
        done();
      });
    });  
  });
});

