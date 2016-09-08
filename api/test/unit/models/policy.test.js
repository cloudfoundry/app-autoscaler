'use strict';
var expect = require('chai').expect;
var fs = require('fs');
var moment = require('moment');
var logger = require('../../../lib/log/logger');
var policy = require('../../../lib/models')().policy_json;

describe('Create a Policy', function() {
  var fakePolicy;

  before(function() {
    fakePolicy = JSON.parse(fs.readFileSync(__dirname+'/../fakePolicy.json', 'utf8'));
  });

  context ('Test create policies', function () {
	  beforeEach(function() {
		policy.truncate();
	  });

	  it('Should create a policy for app id 99999 successfully', function() {
	    return policy.create({ 'policy_json':fakePolicy, 'app_id': '99999' })
	    .then(function(policy) {
	      expect(policy.policy_json).to.deep.equal(fakePolicy);
	      expect(policy).to.have.property('updated_at').to.not.be.null;
	    });
	  });

	  it('Should fail to create a policy with duplicate app_id 99999', function(done) {
	    policy.create({ 'policy_json':fakePolicy, 'app_id': '99999' })
	      .then(function(savedPolicy) {
		    policy.create({ 'policy_json':fakePolicy, 'app_id': '99999' })
		    .catch(function(error) {
		        expect(error).to.not.be.null;
		        expect(error).to.have.deep.property('name').equal('SequelizeUniqueConstraintError');
		        expect(error).to.have.deep.property('message').equal('Validation error');
		        done();
		    });
	    });
	  });	  
  });
  
  context ('Test create and update policies with updated_at checks', function () {
	  beforeEach(function() {
		policy.truncate();
	  });

	  it('Create a policy with app_id 99999 and then updates the policy for the same', function(done) {
		    policy.create({ 'policy_json':fakePolicy, 'app_id': '99999' })
		      .then(function(createdPolicy) {
		    	  expect(createdPolicy).to.have.property('updated_at').to.not.be.null;
		    	  setTimeout(function(){
			    	  policy.update({ 'policy_json':fakePolicy, 'app_id': '99999' },{ where: { app_id: '99999' } ,returning:true})
			    	    .then(function(result) {
			    	      var updatedPolicy = result[1][0];
			    	      expect(updatedPolicy.policy_json).to.deep.equal(fakePolicy);
			    	      expect(updatedPolicy).to.have.property('updated_at').to.not.be.null;
			    	      // Ensure the updated timestamp isAfter created timestamp
			    	      expect(moment(moment(updatedPolicy.updated_at).format('YYYY-MM-DDTHH:mm:ss'))
			    	      		.isAfter(moment(createdPolicy.updated_at).format('YYYY-MM-DDTHH:mm:ss'))).to.equal(true);
				          done();
				      });
		    	  }, 1000);
		    	  // Adding a delay of 1 second before updating the policy for the same app_id 99999 to check the trigger update 
		    });
	  });	  
  });
  
});
