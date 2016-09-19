'use strict';

var expect = require("chai").expect;
var logger = require('../../../lib/log/logger');
var bootstrap = require('../../../lib/utils/bootstrap');
var correctDBUri;
var correctSchedulerURI;

describe('Bootstrapping without mandatory environment', function () {
	context('When DB_URI is not set ', function(){
		before(function() {
			correctDBUri = process.env.DB_URI;
			correctSchedulerURI = process.env.SCHEDULER_URI;
			delete process.env.DB_URI;
		});
		after(function() {
			process.env.DB_URI = correctDBUri;
		});

		it('Should fail to start due to missing DB URI', function (done) {
			bootstrap.boot(function(error){
				expect(error).to.not.be.null; 
				expect(error.message).to.deep.equal('DB_URI not found');
				done();
			});
		});	
	});
	context('When SCHEDULER_URI is not set ', function(){
		before(function() {
			correctDBUri = process.env.DB_URI;
			correctSchedulerURI = process.env.SCHEDULER_URI;
			delete process.env.SCHEDULER_URI;
		});

		after(function() {
			process.env.SCHEDULER_URI = correctSchedulerURI;
		});

		it('Should fail to start due to missing SchedulerURI', function (done) {
			bootstrap.boot(function(error){
				expect(error).to.not.be.null; 
				expect(error.message).to.deep.equal('SCHEDULER_URI not found');
				done();
			});
		});	
	});
	context('When both DB_URI  and SCHEDULER_URI  are available', function(){
		it('Should start the application successfully', function (done) {
			bootstrap.boot(function(error){
				expect(error).to.be.null; 
				done();
			});
		});	
	});
});

