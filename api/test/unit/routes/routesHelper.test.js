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