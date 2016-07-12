'use strict';

var request = require('supertest');  
var expect = require('chai').expect;
var fs = require('fs');
var app = require('../../../app.js');
var logger = require('../../../lib/log/logger');
var validationMiddleware = require('../../../lib/validation/validationMiddleware');

describe('Validate Policy JSON Schema structure', function() {
  var fakePolicy;
  before(function(done) {
    this.policy = require('../../../lib/models')().policy_json;
    var _this = this;
    this.policy.sequelize.sync().then(function(success) {
      _this.policy.truncate().then(function(result) {
        done();
      });  
    }, function(error) {
      logger.error('Failed to setup database for test');
      done(error);
    });
  });
  
  beforeEach(function(){
    fakePolicy = JSON.parse(fs.readFileSync(__dirname+'/../fakePolicy.json', 'utf8'));
  });
  
  it('should validate policy schema successfully', function(done) {
    request(app)
    .put('/v1/apps/12341/policy',validationMiddleware)
    .send(fakePolicy)
    .end(function(error,result) {
      expect(result.statusCode).to.equal(201);
      expect(result.body.success).to.equal(true);
      expect(result.body.error).to.be.null;
      done();
    });

  });
  it('should fail validate policy schema in the absence of required property instance_min_count ', function(done) {
    delete fakePolicy.instance_min_count;
    request(app)
    .put('/v1/apps/12342/policy',validationMiddleware)
    .send(fakePolicy)
    .end(function(error,result) {
      expect(result.statusCode).to.equal(400);
      expect(result.body.success).to.equal(false);
      expect(result.body.error).to.not.be.null;
      expect(result.body.error[0].message).to.equal('requires property "instance_min_count"');
      expect(result.body.error[0].stack).to.equal('instance requires property "instance_min_count"');
      expect(result.body.error[0].schema).to.equal('/policySchema');
      done();
    });

  });
  it('should fail validate policy schema in as instance_min_count value is note in range ', function(done) {
    fakePolicy.instance_min_count = -1;
    request(app)
    .put('/v1/apps/12343/policy',validationMiddleware)
    .send(fakePolicy)
    .end(function(error,result) {
      expect(result.statusCode).to.equal(400);
      expect(result.body.success).to.equal(false);
      expect(result.body.error).to.not.be.null;
      expect(result.body.error[0].message).to.equal('must have a minimum value of 1');
      expect(result.body.error[0].stack).to.equal('instance.instance_min_count must have a minimum value of 1');
      done();
    });

  });
  it('should validate policy schema in the absesnce of schedules ', function(done) {
    delete fakePolicy.schedules;
    request(app)
    .put('/v1/apps/12344/policy',validationMiddleware)
    .send(fakePolicy)
    .end(function(error,result) {
      expect(result.statusCode).to.equal(201);
      expect(result.body.success).to.equal(true);
      expect(result.body.error).to.be.null;
      done();
    });

  });
  it('should validate policy schema in the absesnce of scaling_rules ', function(done) {
    delete fakePolicy.scaling_rules;
    request(app)
    .put('/v1/apps/12345/policy',validationMiddleware)
    .send(fakePolicy)
    .end(function(error,result) {
      expect(result.statusCode).to.equal(201);
      expect(result.body.success).to.equal(true);
      expect(result.body.error).to.be.null;
      done();
    });

  });
  it('should fail to validate policy schema as threshold value is not in the range ', function(done) {
    fakePolicy.scaling_rules[0].threshold = 300;
    request(app)
    .put('/v1/apps/12346/policy',validationMiddleware)
    .send(fakePolicy)
    .end(function(error,result) {
      expect(result.statusCode).to.equal(400);
      expect(result.body.success).to.equal(false);
      expect(result.body.error).to.not.be.null;
      expect(result.body.error[0].property).to.equal('instance.scaling_rules[0].threshold');
      expect(result.body.error[0].message).to.equal('must have a maximum value of 100');
      expect(result.body.error[0].stack).to.equal('instance.scaling_rules[0].threshold must have a maximum value of 100');
      done();
    });

  });
  it('should validate policy schema without stat_window_secs in scaling_rules',function(done) {
    delete fakePolicy.scaling_rules[0].stat_window_secs;
    request(app)
    .put('/v1/apps/12347/policy',validationMiddleware)
    .send(fakePolicy)
    .end(function(error,result) {
      expect(result.statusCode).to.equal(201);
      expect(result.body.success).to.equal(true);
      expect(result.body.error).to.be.null;
      done();
    });

  });
  it('should fail to validate policy schema without metric_type in  scaling_rules', function(done) {
    delete fakePolicy.scaling_rules[0].metric_type;
    request(app)
    .put('/v1/apps/12348/policy',validationMiddleware)
    .send(fakePolicy)
    .end(function(error,result) {
      expect(result.statusCode).to.equal(400);
      expect(result.body.success).to.equal(false);
      expect(result.body.error).to.not.be.null;
      expect(result.body.error[0].property).to.equal('instance.scaling_rules[0]');
      expect(result.body.error[0].message).to.equal('requires property "metric_type"');
      expect(result.body.error[0].stack).to.equal('instance.scaling_rules[0] requires property "metric_type"');
      done();
    });

  });
  it('should fail to validate policy schema with wrong adjustment parameter value in scaling_rules', function(done) {
    fakePolicy.scaling_rules[0].adjustment = '10%';
    request(app)
    .put('/v1/apps/1230897/policy',validationMiddleware)
    .send(fakePolicy)
    .end(function(error,result) {
      expect(result.statusCode).to.equal(400);
      expect(result.body.success).to.equal(false);
      expect(result.body.error).to.not.be.null;
      expect(result.body.error[0].property).to.equal('instance.scaling_rules[0].adjustment');
      expect(result.body.error[0].message).to.equal('does not match pattern "^[-|+][1-9]+[0-9]*$"');
      expect(result.body.error[0].stack).to.equal('instance.scaling_rules[0].adjustment does not match pattern "^[-|+][1-9]+[0-9]*$"');
      done();
    });

  });
  it('should fail to validate policy schema with wrong operator parameter value in scaling_rules', function(done) {
    fakePolicy.scaling_rules[0].operator = '+';
    request(app)
    .put('/v1/apps/12350/policy',validationMiddleware)
    .send(fakePolicy)
    .end(function(error,result) {
      expect(result.statusCode).to.equal(400);
      expect(result.body.success).to.equal(false);
      expect(result.body.error).to.not.be.null;
      expect(result.body.error[0].message).to.equal('is not one of enum values: <,>,<=,>=');
      expect(result.body.error[0].property).to.equal('instance.scaling_rules[0].operator');
      expect(result.body.error[0].stack).to.equal('instance.scaling_rules[0].operator is not one of enum values: <,>,<=,>=');
      done();
    });

  });
  it('should fail to validate policy schema with wrong timezone in schedules', function(done) {
    fakePolicy.schedules.timezone = '(GMT+08: 00) Wrong/Timezone';
    request(app)
    .put('/v1/apps/12351/policy',validationMiddleware)
    .send(fakePolicy)
    .end(function(error,result) {
      expect(result.statusCode).to.equal(400);
      expect(result.body.success).to.equal(false);
      expect(result.body.error).to.not.be.null;
      expect(result.body.error[0].property).to.equal('instance.schedules.timezone');
      done();
    });

  });
  it('should successfully validate policy schema irespective of spaces in timezone', function(done) {
    fakePolicy.schedules.timezone = '   Asia   /  Shanghai ';
    request(app)
    .put('/v1/apps/12351/policy',validationMiddleware)
    .send(fakePolicy)
    .end(function(error,result) {
      expect(result.statusCode).to.equal(201);
      expect(result.body.success).to.equal(true);
      expect(result.body.error).to.be.null;
      done();
    });

  });
  it('should validate policy schema with only recurring_schedule without specific_date', function(done) {
    delete fakePolicy.schedules.specific_date;
    request(app)
    .put('/v1/apps/12352/policy',validationMiddleware)
    .send(fakePolicy)
    .end(function(error,result) {
      expect(result.statusCode).to.equal(201);
      expect(result.body.success).to.equal(true);
      expect(result.body.error).to.be.null;
      done();
    });

  });

  it('should validate policy schema with only specific_date without recurring_schedule', function(done) {
    delete fakePolicy.schedules.recurring_schedule;
    request(app)
    .put('/v1/apps/12353/policy',validationMiddleware)
    .send(fakePolicy)
    .end(function(error,result) {
      expect(result.statusCode).to.equal(201);
      expect(result.body.success).to.equal(true);
      expect(result.body.error).to.be.null;
      done();
    });

  });
  it('should fail to validate policy schema without both specific_date and recurring_schedule', function(done) {
    delete fakePolicy.schedules.recurring_schedule;
    delete fakePolicy.schedules.specific_date;
    request(app)
    .put('/v1/apps/12354/policy',validationMiddleware)
    .send(fakePolicy)
    .end(function(error,result) {
      expect(result.statusCode).to.equal(400);
      expect(result.body.success).to.equal(false);
      expect(result.body.error).to.not.be.null;
      expect(result.body.error[0].message).to.equal('is not any of [subschema 0],[subschema 1]');
      expect(result.body.error[0].property).to.equal('instance.schedules');
      expect(result.body.error[0].stack).to.equal('instance.schedules is not any of [subschema 0],[subschema 1]');
      done();
    });
  });

  it('should fail to validate policy schema with wrong days_of_week value', function(done) {
    fakePolicy.schedules.recurring_schedule[0].days_of_week = [16,56];
    request(app)
    .put('/v1/apps/12355/policy',validationMiddleware)
    .send(fakePolicy)
    .end(function(error,result) {
      expect(result.statusCode).to.equal(400);
      expect(result.body.success).to.equal(false);
      expect(result.body.error).to.not.be.null;
      expect(result.body.error[0].message).to.equal('is not one of enum values: 1,2,3,4,5,6,7');
      expect(result.body.error[0].property).to.equal('instance.schedules.recurring_schedule[0].days_of_week[0]');
      expect(result.body.error[0].stack).to.equal('instance.schedules.recurring_schedule[0].days_of_week[0] is not one of enum values: 1,2,3,4,5,6,7');
      done();
    });

  });
  it('should fail to validate policy schema with duplicate days_of_week value', function(done) {
    fakePolicy.schedules.recurring_schedule[0].days_of_week = [1,2,3,2];
    request(app)
    .put('/v1/apps/12356/policy',validationMiddleware)
    .send(fakePolicy)
    .end(function(error,result) {
      expect(result.statusCode).to.equal(400);
      expect(result.body.success).to.equal(false);
      expect(result.body.error).to.not.be.null;
      expect(result.body.error[0].message).to.equal('contains duplicate item');
      expect(result.body.error[0].property).to.equal('instance.schedules.recurring_schedule[0].days_of_week');
      expect(result.body.error[0].stack).to.equal('instance.schedules.recurring_schedule[0].days_of_week contains duplicate item');
      done();
    });

  });
  it('should fail to validate policy schema with duplicate days_of_month value', function(done) {
    fakePolicy.schedules.recurring_schedule[0].days_of_month = [10,23,31,10];
    request(app)
    .put('/v1/apps/12357/policy',validationMiddleware)
    .send(fakePolicy)
    .end(function(error,result) {
      expect(result.statusCode).to.equal(400);
      expect(result.body.success).to.equal(false);
      expect(result.body.error).to.not.be.null;
      expect(result.body.error[0].message).to.equal('contains duplicate item');
      expect(result.body.error[0].property).to.equal('instance.schedules.recurring_schedule[0].days_of_month');
      expect(result.body.error[0].stack).to.equal('instance.schedules.recurring_schedule[0].days_of_month contains duplicate item');
      done();
    });

  });
  it('should fail to validate policy schema with wrong days_of_month value', function(done) {
    fakePolicy.schedules.recurring_schedule[0].days_of_month = [36,56];
    request(app)
    .put('/v1/apps/12358/policy',validationMiddleware)
    .send(fakePolicy)
    .end(function(error,result) {
      expect(result.statusCode).to.equal(400);
      expect(result.body.success).to.equal(false);
      expect(result.body.error).to.not.be.null;
      expect(result.body.error[0].message).to.equal('is not one of enum values: 1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30,31');
      expect(result.body.error[0].property).to.equal('instance.schedules.recurring_schedule[0].days_of_month[0]');
      expect(result.body.error[0].stack).to.equal('instance.schedules.recurring_schedule[0].days_of_month[0] is not one of enum values: 1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30,31');
      done();
    });

  });
  it('should fail to validate policy schema with wrong start_time format in recurring_schedule', function(done) {
    fakePolicy.schedules.recurring_schedule[0].start_time = '123:566 PM';
    request(app)
    .put('/v1/apps/12359/policy',validationMiddleware)
    .send(fakePolicy)
    .end(function(error,result) {
      expect(result.statusCode).to.equal(400);
      expect(result.body.success).to.equal(false);
      expect(result.body.error).to.not.be.null;
      expect(result.body.error[0].message).to.equal('does not conform to the "timeFormat" format');
      expect(result.body.error[0].property).to.equal('instance.schedules.recurring_schedule[0].start_time');
      expect(result.body.error[0].stack).to.equal('instance.schedules.recurring_schedule[0].start_time does not conform to the "timeFormat" format');
      done();
    });

  });
  it('should fail to validate policy schema with wrong start_date_time format in specific_date', function(done) {
    fakePolicy.schedules.specific_date[0].start_date_time = '2015-06-19';
    request(app)
    .put('/v1/apps/12360/policy',validationMiddleware)
    .send(fakePolicy)
    .end(function(error,result) {
      expect(result.statusCode).to.equal(400);
      expect(result.body.success).to.equal(false);
      expect(result.body.error).to.not.be.null;
      expect(result.body.error[0].message).to.equal('does not conform to the "dateTimeFormat" format');
      expect(result.body.error[0].property).to.equal('instance.schedules.specific_date[0].start_date_time');
      expect(result.body.error[0].stack).to.equal('instance.schedules.specific_date[0].start_date_time does not conform to the "dateTimeFormat" format');
      done();
    });

  });

  it('should fail to validate a valid policy schema with wrong policy attribute value', function(done) {
    fakePolicy.schedules.specific_date[1].start_date_time = '2015-06-02T10:00';
    request(app)
    .put('/v1/apps/12361/policy',validationMiddleware)
    .send(fakePolicy)
    .end(function(error,result) {
      expect(result.statusCode).to.equal(400);
      expect(result.body.success).to.equal(false);
      expect(result.body.error).to.not.be.null;
      expect(result.body.error[0].message).to.equal('specific_date.start_date_time and specific_date.end_date_time values are not compatible');
      expect(result.body.error[0].property).to.equal('specific_date.start_date_time');
      expect(result.body.error[0].stack).to.equal('start_date_time 2015-06-02T10:00 is after end_date_time 2015-02-19T23:15 in specific_date :[1]');
      done();
    });
  });
  it('should fail to validate a policy schema with recurring_schedule having both days_of_week and days_of_month property',function(done){
    fakePolicy.schedules.recurring_schedule[0].days_of_month = [1,15];
    request(app)
    .put('/v1/apps/12355/policy',validationMiddleware)
    .send(fakePolicy)
    .end(function(error,result) {
      expect(result.statusCode).to.equal(400);
      expect(result.body.success).to.equal(false);
      expect(result.body.error).to.not.be.null;
      expect(result.body.error[0].message).to.equal('is not exactly one from [subschema 0],[subschema 1]');
      expect(result.body.error[0].property).to.equal('instance.schedules.recurring_schedule[0]');
      expect(result.body.error[0].stack).to.equal('instance.schedules.recurring_schedule[0] is not exactly one from [subschema 0],[subschema 1]');
      done();
    });
  });
});

