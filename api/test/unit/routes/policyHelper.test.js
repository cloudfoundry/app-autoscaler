'use strict';

var request = require('supertest');
var expect = require('chai').expect;
var fs = require('fs');
var path = require('path');
var uuidV4 = require('uuid/v4');
var settings = require(path.join(__dirname, '../../../lib/config/setting.js'))((JSON.parse(
	fs.readFileSync(path.join(__dirname, '../../../config/settings.json'), 'utf8'))));
var relativePath = path.relative(process.cwd(), path.join(__dirname, "../../../../test-certs"));
var testSetting = require(path.join(__dirname, '../test.helper.js'))(relativePath,settings);
var models = require('../../../lib/models')(testSetting.db, function(){});
var API = require('../../../app.js');
var app;
var publicApp;
var healthApp;
var servers;
var policy = models.policy_json;
var logger = require('../../../lib/log/logger');
var nock = require('nock');
var HttpStatus = require('http-status-codes');
var policyHelper = require('../../../lib/routes/policyHelper')(models);


describe('Policy Route helper ', function() {
	var fakePolicy;

    before(function() {
		fakePolicy = JSON.parse(fs.readFileSync(__dirname+'/../fakePolicy.json', 'utf8'));
		servers = API(testSetting, function(){});
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
	beforeEach(function() {
		if (settings.serviceOffering) {
			nock(settings.serviceOffering.serviceBroker.uri)
			.get(/\/v1\/apps\/.+\/service_bindings/)
			.reply(200, {"message": "binding_info_exists"});
		}
		return policy.truncate();
	});

    context ('Create a policy', function () {
    	it('should create with schedules for app id 12345',function(done){
    		var mockRequest = {
    				body : fakePolicy,
    				params : { 'app_id' : '12345' },
    				query : { 'policy_guid' : uuidV4()}
    		};

    		policyHelper.createOrUpdatePolicy(mockRequest, function(error,result){
    			expect(result.statusCode).to.equal(HttpStatus.CREATED);
    			expect(error).to.be.null;
    			done();
    		});
    	});

    	it('should fail due to an internal error',function(done){
    		//Mocking a request without any app_id in the request param
    		var mockRequest = {
    				body : fakePolicy,
    				params : { 'key' : 'value' },
    				query : { 'policy_guid' : uuidV4()}
    		};
    		policyHelper.createOrUpdatePolicy(mockRequest, function(error,result){
    			expect(error).not.to.be.null;
    			expect(error.message).eql('WHERE parameter "app_id" has invalid "undefined" value');
    			done();
    		});
    	});
    	
    	it('should create without any schedules info for app id 12346',function(done){
    		delete fakePolicy.schedules;
    		var mockRequest = {
    				body : fakePolicy,
    				params : { 'app_id' : '12346' },
    				query : { 'policy_guid' : uuidV4()}
    		};

    		policyHelper.createOrUpdatePolicy(mockRequest, function(error,result){
    			expect(result.statusCode).to.equal(HttpStatus.CREATED);
    			expect(error).to.be.null;
    			done();
    		});
    	});
    	
    });
	
    context ('Delete a policy', function () {
    	it('should return a 404 for app id 12345 which does not exist',function(done){
    		var mockRequest = {
    				body : fakePolicy,
    				params : { 'app_id' : '12345' },
    				query : { 'policy_guid' : uuidV4()}
    		};
    		var app_id = '12345';
    		policyHelper.deletePolicy(mockRequest, function(error){
    			expect(error).to.not.be.null;
    			expect(error.statusCode).to.equal(404);
    			done();
    		});
    	});	
    	
    });
	
	context('when policy already exists',function(){
		beforeEach(function(done){
			var mockRequest = {
					body : fakePolicy,
					params : { 'app_id' : '12348' },
    				query : { 'policy_guid' : uuidV4()}
			};
			policyHelper.createOrUpdatePolicy(mockRequest, function(error,result){
				done();
			});
		});

		it('should update with schedules',function(done){
			var mockRequest = {
					params : { 'app_id' : '12348' },
					body: fakePolicy,
    				query : { 'policy_guid' : uuidV4()}
			};
			
			policyHelper.createOrUpdatePolicy(mockRequest, function(error,result){
				expect(error).to.be.null;
				expect(result.statusCode).to.equal(HttpStatus.OK);
				done();
			});
		});
		
		it('should fail to update with schedules due to an internal error',function(done){
			//Mocking a request without any policy_json in the request body
			var mockRequest = {
					params : { 'app_id' : '12348' },
    				query : { 'policy_guid' : uuidV4()}
			};
			
			policyHelper.createOrUpdatePolicy(mockRequest, function(error,result){
				expect(error).not.to.be.null;
				expect(error.name).eql('AssertionError');
				expect(error.message).eql('expected null not to be null');
				done();
			});
		});
	});

	context('Create a policy and delete',function(){
		beforeEach(function(done){
			var mockRequest = {
					body : fakePolicy,
					params : { 'app_id' : '12348' },
    				query : { 'policy_guid' : uuidV4()}
			};

			policyHelper.createOrUpdatePolicy(mockRequest, function(error,result){
				done();
			});
		});
		
		it('should delete the policy with app_id 12348',function(done){
			var mockRequest = {
					params : { 'app_id' : '12348' }
			};
			policyHelper.deletePolicy(mockRequest, function(error){
				expect(error).to.be.null;
				request(app)
				  .get('/v1/apps/12348/policy')
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

			policyHelper.deletePolicy(mockRequest, function(error){
				expect(error).to.not.be.null;
				expect(error.statusCode).to.equal(500);
				request(app)
				  .get('/v1/apps/12348/policy')
				  .end(function(error,result) {
				      expect(result.statusCode).to.equal(200);
				      expect(result.body).to.deep.equal(fakePolicy);
				      done();
				});    
			});
		});
		
	});
	
});
