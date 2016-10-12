'use strict';

var request = require('supertest');
var expect = require('chai').expect;
var fs = require('fs');
var app = require('../../../app.js');
var policy = require('../../../lib/models')().policy_json;
var logger = require('../../../lib/log/logger');
var nock = require('nock');
var HttpStatus = require('http-status-codes');
var routeHelper = require('../../../lib/routes/routeHelper');
var schedulerURI = process.env.SCHEDULER_URI;


describe('Policy Route helper ', function() {
	var fakePolicy;

    before(function() {
		fakePolicy = JSON.parse(fs.readFileSync(__dirname+'/../fakePolicy.json', 'utf8'));
	})

	beforeEach(function() {
		return policy.truncate();
	});

	it('should create a policy with schedule for app id 12345',function(done){
		var mockRequest = {
				body : fakePolicy,
				params : { 'app_id' : '12345' }
		};
		var mockSchedulerResponse = {'statusCode':HttpStatus.OK };
		routeHelper.createOrUpdatePolicy(mockRequest,mockSchedulerResponse,function(error,result){
			expect(result.statusCode).to.equal(HttpStatus.CREATED);
			expect(error).to.be.null;
			done();
		});
	});
	
	it('should return a 404 in trying to delete a policy for app id 12345 which does not exist',function(done){
		var mockRequest = {
				body : fakePolicy,
				params : { 'app_id' : '12345' }
		};
		var app_id = '12345';
		routeHelper.deletePolicy(mockRequest, function(error){
			expect(error).to.not.be.null;
			expect(error.statusCode).to.equal(404);
			done();
		});
	});	
	
	context('when policy already exists',function(){
		beforeEach(function(done){
			var mockRequest = {
					body : fakePolicy,
					params : { 'app_id' : '12348' }
			};
			var mockSchedulerResponse = {'statusCode':HttpStatus.OK };
			routeHelper.createOrUpdatePolicy(mockRequest,mockSchedulerResponse,function(error,result){
				done();
			});
		});
		it('should fail to update a policy with schedule due to internal error',function(done){
			//Mocking a request without any policy_json in the request body
			var mockRequest = {
					params : { 'app_id' : '12348' }
			};
			var mockSchedulerResponse = {'statusCode':HttpStatus.INTERNAL_SERVER_ERROR };
			routeHelper.createOrUpdatePolicy(mockRequest,mockSchedulerResponse,function(error,result){
				expect(error).not.to.be.null;
				expect(error.name).eql('SequelizeValidationError');
				expect(error.message).eql('notNull Violation: policy_json cannot be null');
				done();
			});
		});
	});

	context('Create a policy and delete',function(){
		beforeEach(function(done){
			var mockRequest = {
					body : fakePolicy,
					params : { 'app_id' : '12348' }
			};
			var mockSchedulerResponse = {'statusCode':HttpStatus.OK };
			routeHelper.createOrUpdatePolicy(mockRequest,mockSchedulerResponse,function(error,result){
				done();
			});
		});
		
		it('should delete the policy with app_id 12348',function(done){
			var mockRequest = {
					params : { 'app_id' : '12348' }
			};
			routeHelper.deletePolicy(mockRequest, function(error){
				expect(error).to.be.null;
				request(app)
				  .get('/v1/policies/12348')
				  .end(function(error,result) {
				      expect(result.statusCode).to.equal(404);
				      done();
				});    
			});
		});

		it('should fail to delete the policy due to an internal server error',function(done){
			var mockRequest = {
					params : {} // Not passing the app_id will throw an internal server error
			};

			routeHelper.deletePolicy(mockRequest, function(error){
				expect(error).to.not.be.null;
				expect(error.statusCode).to.equal(500);
				request(app)
				  .get('/v1/policies/12348')
				  .end(function(error,result) {
				      expect(result.statusCode).to.equal(200);
				      expect(result.body).to.deep.equal(fakePolicy);
				      done();
				});    
			});
		});
		
	});

	it('should fail to create a policy due to internal error',function(done){
		//Mocking a request without any app_id in the request param
		var mockRequest = {
				body : fakePolicy,
				params : { 'key' : 'value' }
		};
		var mockSchedulerResponse = {'statusCode':HttpStatus.INTERNAL_SERVER_ERROR };
		routeHelper.createOrUpdatePolicy(mockRequest,mockSchedulerResponse,function(error,result){
			expect(error).not.to.be.null;
			expect(error.name).eql('SequelizeDatabaseError');
			expect(error.message).eql('null value in column "app_id" violates not-null constraint');
			done();
		});
	});
	
	it('should create a policy without any schedule info for app id 12346',function(done){
		delete fakePolicy.schedules;
		var mockRequest = {
				body : fakePolicy,
				params : { 'app_id' : '12346' }
		};
		var mockSchedulerResponse = {'statusCode':HttpStatus.OK };
		routeHelper.createOrUpdatePolicy(mockRequest,mockSchedulerResponse,function(error,result){
			expect(result.statusCode).to.equal(HttpStatus.CREATED);
			expect(error).to.be.null;
			done();
		});
	});
	
});