'use strict';

var request = require('supertest');
var expect = require('chai').expect;
var fs = require('fs');
var path = require('path');
var settings = require(path.join(__dirname, '../../../lib/config/settings.js'))((JSON.parse(
  fs.readFileSync(path.join(__dirname, '../../../config/settings.json'), 'utf8'))));
var policy = require('../../../lib/models')(settings).policy_json;
var logger = require('../../../lib/log/logger');
var nock = require('nock');
var schedulerUtils = require('../../../lib/utils/schedulerUtils')(settings);
var schedulerURI = settings.schedulerUri;

describe('Scheduler Utility functions', function() {
  var fakePolicy;

  before(function() {
    fakePolicy = JSON.parse(fs.readFileSync(__dirname+'/../fakePolicy.json', 'utf8'));
  })

  beforeEach(function() {
    return policy.truncate();
  })

  context ('Creating schedules during policy creation', function () {
	  it('should create new schedules for app id 12345',function(done){
		    nock(schedulerURI)
		    .put('/v2/schedules/12345')
		    .reply(204);
		    var mockRequest = {
		        body : fakePolicy,
		        params : { 'app_id' : '12345' }
		    };
		    schedulerUtils.createOrUpdateSchedule(mockRequest,function(error){
		      expect(error).to.be.null;
		      done();
		    });
		  });
	
	  it('should fail to create schedules due to error for app id 12346',function(done){
		    var mockError = { 'message':'Failed to create schedules due to an internal' + 
		        ' error in scheduler','details':'fake body' };
		    nock(schedulerURI)
		    .put('/v2/schedules/12346')
		    .replyWithError(mockError);
		    var mockRequest = {
		        body : fakePolicy,
		        params : { 'app_id' : '12346' }
		    };
		    schedulerUtils.createOrUpdateSchedule(mockRequest,function(error){
		      expect(error).to.not.be.null;
		      expect(error.statusCode).to.equal(500);
		      expect(error).to.deep.equal(mockError);
		      done();
		    });
		  });
	
	  it('should fail to create schedules due to internal validation error in scheduler module for app id 12347',function(done){
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
		    schedulerUtils.createOrUpdateSchedule(mockRequest,function(error){
		      expect(error.statusCode).to.equal(400);
		      expect(error).to.not.be.null;
		      expect(error.details).to.equal('fake body');
		      expect(error.message).to.equal('Failed to create schedules due to validation error in scheduler');
		      done();
		    });
		  });
		  
      it('should fail to create schedules due to un-accepted response code (other than 400 ) in scheduler module for app id 12348',function(done){
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
		    schedulerUtils.createOrUpdateSchedule(mockRequest,function(error){
		      expect(error.statusCode).to.equal(500);
		      expect(error).to.not.be.null;
		      expect(error.details).to.equal('fake body');
		      expect(error.message).to.equal('Failed to create schedules due to an internal error in scheduler');
		      done();
		    });
	   });
	  
  });
  
  context ("Deleting schedules during policy deletion", function() {
	  
	  it('should pass (No error) if the scheduler returns 404 (No schedules) for app',function(done){
	    	nock(schedulerURI)
	        .delete('/v2/schedules/12345_NoSchedules')
	        .reply(404);
	        var mockRequest = {
	                body : fakePolicy,
	                params : { 'app_id' : '12345_NoSchedules' }
	        };
	        schedulerUtils.deleteSchedules(mockRequest ,function(error){
	          expect(error).to.be.null;
	          done();
	        });
	   });

	  it('should fail due to an internal server error 500 response code from scheduler',function(done){
	  	  nock(schedulerURI)
	      .delete('/v2/schedules/12345_ErrorFromScheduler')
	      .reply(500);

	  	  var mockRequest = {
	              body : fakePolicy,
	              params : { 'app_id' : '12345_ErrorFromScheduler' }
	      };
	      schedulerUtils.deleteSchedules(mockRequest, function(error){
	        expect(error).to.not.be.null;
	        expect(error.statusCode).to.equal(500);
	        done();
	      });
	 });

	  it('should fail due to an internal error with the request',function(done){
		  	nock(schedulerURI)
		      .delete('/v2/schedules/')
		      .reply(503);
		      var mockRequest = {
		              body : fakePolicy,
		              params : { 'app_id' : '123456' }
		      };
		      schedulerUtils.deleteSchedules(mockRequest, function(error){
			    expect(error).to.not.be.null;
		        expect(error.statusCode).to.equal(500);
		        done();
		      });
	  });
	  
	  it('should successfully delete schedules for app id 12345',function(done){
		  	nock(schedulerURI)
		      .delete('/v2/schedules/12345')
		      .reply(200);
		      var mockRequest = {
		              body : fakePolicy,
		              params : { 'app_id' : '12345' }
		      };
		      schedulerUtils.deleteSchedules(mockRequest, function(error){
		        expect(error).to.be.null;
		        done();
		      });
		 });
	  
  });

  context('when schedules already exists' ,function() {
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
      schedulerUtils.createOrUpdateSchedule(mockRequest,function(error){
        expect(error).to.be.null;
        done();
      });
    });  
  });
});

