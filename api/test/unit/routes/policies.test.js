'use strict';

var request = require('supertest');  
var expect = require('chai').expect;
var app = require('../../../app.js');
var logger = require('../../../lib/log/logger');
var fakePolicy = {
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
};

describe('Routing Policy', function() {
  before(function() {
    this.models = require('../../../lib/models')();
    this.Policy = this.models.policy_json;
    this.Policy.sequelize.sync().then(function() {
      logger.info('creating the table structure');
    });
  });
  describe('Policy Creation', function() {
    it('should create a policy for app id 12345', function(done) {
      request(app)
      .put('/v1/apps/12345/policy')
      .send(fakePolicy)
      .end(function(error,result) {
        if (error) {
          expect(error).to.not.be.null;
          throw error;
        }
        expect(result.statusCode).to.equal(200);
        done();
      });
    
    });

    it('should fail to create another policy for app id 12345', function(done) {
      request(app)
      .put('/v1/apps/12345/policy')
      .send(fakePolicy)
      .end(function(error,result) {
        if (error) {
          expect(error).to.not.be.null;
          throw error;
        }
        expect(result.statusCode).to.equal(400);
        done();
      });
    });
  });

  after(function() {
    this.Policy.truncate().then(function(result) {
      logger.warn('Truncating the test db before test run !!!');
    });  
  });
});
