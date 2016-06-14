'use strict';
var expect = require("chai").expect;
var logger = require('../../lib/log/logger');
var routes = require('../../lib/routes/policies');

var dummyPolicyJson = [{
	"instance_min_count" : 1,
	"instance_max_count" : 5,
	"scaling_rules" : [ {
		"metric_type" : "MemoryUsage",
		"stat_window_secs" : 300,
		"breach_duration_secs" : 600,
		"threshold" : 30,
		"operator" : "<",
		"cool_down_secs" : 300,
		"adjustment" : "-1%"
	}, {
		"metric_type" : "MemoryUsage",
		"stat_window_secs" : 300,
		"breach_duration_secs" : 600,
		"threshold" : 90,
		"operator" : ">=",
		"cool_down_secs" : 300,
		"adjustment" : "+1"
	} ],
	"schedules" : {
		"timezone" : "(GMT+08: 00)Asia/Shanghai",
		"recurring_schedule" : [ {
			"start_time" : "00: 00",
			"end_time" : "08: 00",
			"days_of_week" : [ 1, 2, 3 ],
			"days_of_month" : [ 1, 28 ],
			"instance_min_count" : 2,
			"instance_max_count" : 10
		} ],
		"specific_date" : [ {
			"start_date" : "2015-06-19",
			"start_time" : "00: 00",
			"end_date" : "2015-06-19",
			"end_time" : "23: 59",
			"instance_min_count" : 15,
			"instance_max_count" : 40
		} ]
	}
}];

describe('Clean up test database', function () {
  before(function () {
    this.Policy = require('../../lib/models').policy_json;
    //creating the test DB if does not exist.
    this.Policy.sequelize.sync().then(function () {
    logger.info('creating the table structure');
    });
});
  
  beforeEach(function(){
  this.Policy.truncate().then(function(result){
    logger.warn('Truncating the test db before test run !!!');
    });  
  });

  describe('create a Policy', function () {
    it('Success', function () {
        return this.Policy.create({  'policy_json':dummyPolicyJson, 'app_id': '12349' }).then(function (policy) {
        	logger.info('Policy created for app_id 12349');
        	expect(policy.policy_json).to.equal(dummyPolicyJson);
          
        });
    });
    it('Failure', function () {
    	this.Policy.create({  'policy_json':dummyPolicyJson, 'app_id': '12349' }).then(function (policy) {
        	logger.info('Policy created for app_id 12349');
        	expect(policy.policy_json).to.equal(dummyPolicyJson);
          
        });    	
    	return this.Policy.create({  'policy_json':dummyPolicyJson, 'app_id': '12349' }).then(function (policy) {
          }).catch(function(error){
        	  logger.error('Failed to create policy ',error);
        	  expect(error).to.not.be.null;
        	  expect(error).to.have.deep.property('name').equal('SequelizeUniqueConstraintError');
        	  expect(error).to.have.deep.property('message').equal('Validation error');
          });
     });
    
  });
  
});
