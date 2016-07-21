'use strict';

var expect = require("chai").expect;
var logger = require('../../../lib/log/logger');
var correctDBUri;

describe('Test Model Load Failures', function () {
  before(function() {
	  correctDBUri = process.env.DB_URI;
	  // Provide a wrong db uri
	  process.env.DB_URI = 'postgres://postgres:postgres@local_host:5432/autoscaler_test';
  });
	
  it('Should fail for a invalid DB_URI', function () {
    var models = require('../../../lib/models')(function (error) {
    	logger.error ('Error while establishing connection', error);
    	expect(error).to.not.be.null;
    });
  });
  
  after(function() {
	  process.env.DB_URI = correctDBUri;
  });
  
});
