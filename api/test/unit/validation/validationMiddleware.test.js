'use strict';

var request = require('supertest');
var expect = require('chai').expect;
var fs = require('fs');
var path = require('path');
var settings = require(path.join(__dirname, '../../../lib/config/setting.js'))((JSON.parse(
  fs.readFileSync(path.join(__dirname, '../../../config/settings.json'), 'utf8'))));
var API = require('../../../app.js');
var app;
var logger = require('../../../lib/log/logger');
var policy = require('../../../lib/models')(settings.db).policy_json;
var validationMiddleware = require('../../../lib/validation/validationMiddleware');
var nock = require('nock');
var schedulerURI = settings.scheduler.uri;

describe('Validate Policy JSON Schema structure', function() {
  var policyContent;
  var fakePolicy;

  before(function() {
    policyContent = fs.readFileSync(__dirname+'/../fakePolicy.json', 'utf8');
    app = API(path.join(__dirname, '../../../config/settings.json'));
  });
  after(function(done){
      app.close(done);
  })
  beforeEach(function() {
    fakePolicy = JSON.parse(policyContent);
    return policy.truncate();
  });

  it('should validate policy schema successfully', function(done) {
    nock(schedulerURI)
    .put('/v2/schedules/12341')
    .query({'guid':/.*/})
    .reply(200);
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
  it('should fail validate policy schema in as instance_min_count value is not in range ', function(done) {
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
    nock(schedulerURI)
    .put('/v2/schedules/12345')
    .query({'guid':/.*/})
    .reply(200);
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
  it('should fail to validate policy schema as threshold value is negative', function(done) {
    fakePolicy.scaling_rules[0].threshold = -300;
    request(app)
    .put('/v1/apps/12346/policy',validationMiddleware)
    .send(fakePolicy)
    .end(function(error,result) {
      expect(result.statusCode).to.equal(400);
      expect(result.body.success).to.equal(false);
      expect(result.body.error).to.not.be.null;
      expect(result.body.error[0].property).to.equal('instance.scaling_rules[0].threshold');
      expect(result.body.error[0].message).to.equal('must have a minimum value of 0');
      expect(result.body.error[0].stack).to.equal('instance.scaling_rules[0].threshold must have a minimum value of 0');
      done();
    });

  });
  it('should validate policy schema without stat_window_secs in scaling_rules',function(done) {
    nock(schedulerURI)
    .put('/v2/schedules/12347')
    .query({'guid':/.*/})
    .reply(200);
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

  it('should fail to validate policy schema with not supported metric_type in scaling_rules', function(done) {
    fakePolicy.scaling_rules[0].metric_type = "not-supported-metric";
    request(app)
    .put('/v1/apps/12348/policy',validationMiddleware)
    .send(fakePolicy)
    .end(function(error,result) {
      expect(result.statusCode).to.equal(400);
      expect(result.body.success).to.equal(false);
      expect(result.body.error).to.not.be.null;
      expect(result.body.error[0].property).to.equal('instance.scaling_rules[0].metric_type');
      expect(result.body.error[0].message).to.equal('is not one of enum values: memoryused,memoryutil,responsetime,throughput');
      expect(result.body.error[0].stack).to.equal('instance.scaling_rules[0].metric_type is not one of enum values: memoryused,memoryutil,responsetime,throughput');
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
    nock(schedulerURI)
    .put('/v2/schedules/12351')
    .query({'guid':/.*/})
    .reply(200);
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
    nock(schedulerURI)
    .put('/v2/schedules/12352')
    .query({'guid':/.*/})
    .reply(200);
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
    nock(schedulerURI)
    .put('/v2/schedules/12353')
    .query({'guid':/.*/})
    .reply(200);
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
      expect(result.body.error[0].stack).to.equal('start_date_time 2015-06-02T10:00 is same or after end_date_time 2015-02-19T23:15 in specific_date :[1]');
      done();
    });
  });
  it('should fail to validate a policy when start_date_time is same as end_date_time in specific_date', function(done) {
    fakePolicy.schedules.specific_date[1].start_date_time = '2015-02-19T23:15';
    request(app)
    .put('/v1/apps/12361/policy',validationMiddleware)
    .send(fakePolicy)
    .end(function(error,result) {
      expect(result.statusCode).to.equal(400);
      expect(result.body.success).to.equal(false);
      expect(result.body.error).to.not.be.null;
      expect(result.body.error[0].message).to.equal('specific_date.start_date_time and specific_date.end_date_time values are not compatible');
      expect(result.body.error[0].property).to.equal('specific_date.start_date_time');
      expect(result.body.error[0].stack).to.equal('start_date_time 2015-02-19T23:15 is same or after end_date_time 2015-02-19T23:15 in specific_date :[1]');
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
  it('should successfully validate policy schema without start_date and end_date in recurring_schedule', function(done) {
    nock(schedulerURI)
    .put('/v2/schedules/12359')
    .query({'guid':/.*/})
    .reply(200);
     request(app)
    .put('/v1/apps/12359/policy',validationMiddleware)
    .send(fakePolicy)
    .end(function(error,result) {
      expect(result.statusCode).to.equal(201);
      expect(result.body.success).to.equal(true);
      expect(result.body.error).to.be.null;
      done();
    });
  });
  it('should successfully validate policy schema with start_date but without end_date in recurring_schedule', function(done) {
    nock(schedulerURI)
    .put('/v2/schedules/12359')
    .query({'guid':/.*/})
    .reply(200);
    delete  fakePolicy.schedules.recurring_schedule[0].end_date;
    request(app)
    .put('/v1/apps/12359/policy',validationMiddleware)
    .send(fakePolicy)
    .end(function(error,result) {
      expect(result.statusCode).to.equal(201);
      expect(result.body.success).to.equal(true);
      expect(result.body.error).to.be.null;
      done();
    });
  });
  it('should successfully validate policy schema with end_date but without start_date in recurring_schedule', function(done) {
    nock(schedulerURI)
    .put('/v2/schedules/12359')
    .query({'guid':/.*/})
    .reply(200);
    delete  fakePolicy.schedules.recurring_schedule[0].start_date;
    request(app)
    .put('/v1/apps/12359/policy',validationMiddleware)
    .send(fakePolicy)
    .end(function(error,result) {
      expect(result.statusCode).to.equal(201);
      expect(result.body.success).to.equal(true);
      expect(result.body.error).to.be.null;
      done();
    });
  });
  it('should fail to validate policy schema with wrong start_date format in recurring_schedule', function(done) {
    fakePolicy.schedules.recurring_schedule[0].start_date = '2016-76-900';
    request(app)
    .put('/v1/apps/12359/policy',validationMiddleware)
    .send(fakePolicy)
    .end(function(error,result) {
      expect(result.statusCode).to.equal(400);
      expect(result.body.success).to.equal(false);
      expect(result.body.error).to.not.be.null;
      expect(result.body.error[0].message).to.equal('is not any of [subschema 0],[subschema 1]');
      expect(result.body.error[0].property).to.equal('instance.schedules.recurring_schedule[0].start_date');
      expect(result.body.error[0].stack).to.equal('instance.schedules.recurring_schedule[0].start_date is not any of [subschema 0],[subschema 1]');
      done();
    });
  });
  it('should fail to validate policy schema with wrong end_date format in recurring_schedule', function(done) {
    fakePolicy.schedules.recurring_schedule[0].end_date = '23-07-2016';
    request(app)
    .put('/v1/apps/12359/policy',validationMiddleware)
    .send(fakePolicy)
    .end(function(error,result) {
      expect(result.statusCode).to.equal(400);
      expect(result.body.success).to.equal(false);
      expect(result.body.error).to.not.be.null;
      expect(result.body.error[0].message).to.equal('is not any of [subschema 0],[subschema 1]');
      expect(result.body.error[0].property).to.equal('instance.schedules.recurring_schedule[0].end_date');
      expect(result.body.error[0].stack).to.equal('instance.schedules.recurring_schedule[0].end_date is not any of [subschema 0],[subschema 1]');
      done();
    });
  });
  it('should fail to validate policy schema if start_date is after end_date in recurring_schedule', function(done) {
    fakePolicy.schedules.recurring_schedule[0].start_date = '2016-07-23';
    fakePolicy.schedules.recurring_schedule[0].end_date = '2016-05-12';
    request(app)
    .put('/v1/apps/12359/policy',validationMiddleware)
    .send(fakePolicy)
    .end(function(error,result) {
      expect(result.statusCode).to.equal(400);
      expect(result.body.success).to.equal(false);
      expect(result.body.error).to.not.be.null;
      expect(result.body.error[0].message).to.equal('recurring_schedule.start_date and recurring_schedule.end_date values are not compatible');
      expect(result.body.error[0].property).to.equal('recurring_schedule.start_date');
      expect(result.body.error[0].stack).to.equal('start_date 2016-07-23 is after end_date 2016-05-12 in recurring_schedule :[0]');
      done();
    });
  });
  
  it('should validate policy schema without initial_min_instance_count in recurring_schedule', function(done) {
    nock(schedulerURI)
    .put('/v2/schedules/12359')
    .query({'guid':/.*/})
    .reply(200);
    delete  fakePolicy.schedules.recurring_schedule[0].initial_min_instance_count;
    delete  fakePolicy.schedules.recurring_schedule[1].initial_min_instance_count;
    request(app)
    .put('/v1/apps/12359/policy',validationMiddleware)
    .send(fakePolicy)
    .end(function(error,result) {
      expect(result.statusCode).to.equal(201);
      expect(result.body.success).to.equal(true);
      expect(result.body.error).to.be.null;
      done();
    });
  });
  
  it('should validate policy schema without initial_min_instance_count in specific_date schedule', function(done) {
    nock(schedulerURI)
    .put('/v2/schedules/12359')
    .query({'guid':/.*/})
    .reply(200);
    delete  fakePolicy.schedules.specific_date[0].initial_min_instance_count;
    delete  fakePolicy.schedules.specific_date[1].initial_min_instance_count;
    request(app)
    .put('/v1/apps/12359/policy',validationMiddleware)
    .send(fakePolicy)
    .end(function(error,result) {
      expect(result.statusCode).to.equal(201);
      expect(result.body.success).to.equal(true);
      expect(result.body.error).to.be.null;
      done();
    });
  });
  
  it('should fail to validate policy schema if initial_min_instance_count is greater than instance_max_count in recurring_schedule', function(done) {
    fakePolicy.schedules.recurring_schedule[0].initial_min_instance_count = 11;
    request(app)
    .put('/v1/apps/12359/policy',validationMiddleware)
    .send(fakePolicy)
    .end(function(error,result) {
      expect(result.statusCode).to.equal(400);
      expect(result.body.success).to.equal(false);
      expect(result.body.error).to.not.be.null;
      expect(result.body.error[0].message).to.equal('recurring_schedule.initial_min_instance_count and recurring_schedule.instance_max_count values are not compatible');
      expect(result.body.error[0].property).to.equal('recurring_schedule.initial_min_instance_count');
      expect(result.body.error[0].stack).to.equal('initial_min_instance_count 11 is higher than instance_max_count 10 in recurring_schedule :[0]');
      done();
    });
  });
  
  it('should fail to validate policy schema if initial_min_instance_count is less than instance_min_count in recurring_schedule', function(done) {
    fakePolicy.schedules.recurring_schedule[1].initial_min_instance_count = 1;
    request(app)
    .put('/v1/apps/12359/policy',validationMiddleware)
    .send(fakePolicy)
    .end(function(error,result) {
      expect(result.statusCode).to.equal(400);
      expect(result.body.success).to.equal(false);
      expect(result.body.error).to.not.be.null;
      expect(result.body.error[0].message).to.equal('recurring_schedule.initial_min_instance_count and recurring_schedule.instance_min_count values are not compatible');
      expect(result.body.error[0].property).to.equal('recurring_schedule.initial_min_instance_count');
      expect(result.body.error[0].stack).to.equal('initial_min_instance_count 1 is lower than instance_min_count 3 in recurring_schedule :[1]');
      done();
    });
  });
  it('should fail to validate policy schema if initial_min_instance_count is greater than instance_max_count in specific_date schedule', function(done) {
    fakePolicy.schedules.specific_date[0].initial_min_instance_count = 5;
    request(app)
    .put('/v1/apps/12359/policy',validationMiddleware)
    .send(fakePolicy)
    .end(function(error,result) {
      expect(result.statusCode).to.equal(400);
      expect(result.body.success).to.equal(false);
      expect(result.body.error).to.not.be.null;
      expect(result.body.error[0].message).to.equal('specific_date.initial_min_instance_count and specific_date.instance_max_count values are not compatible');
      expect(result.body.error[0].property).to.equal('specific_date.initial_min_instance_count');
      expect(result.body.error[0].stack).to.equal('initial_min_instance_count 5 is higher than instance_max_count 4 in specific_date :[0]');
      done();
    });
  });
  
  it('should fail to validate policy schema if initial_min_instance_count is less than instance_min_count in recurring_schedule', function(done) {
    fakePolicy.schedules.specific_date[1].initial_min_instance_count = 1;
    request(app)
    .put('/v1/apps/12359/policy',validationMiddleware)
    .send(fakePolicy)
    .end(function(error,result) {
      expect(result.statusCode).to.equal(400);
      expect(result.body.success).to.equal(false);
      expect(result.body.error).to.not.be.null;
      expect(result.body.error[0].message).to.equal('specific_date.initial_min_instance_count and specific_date.instance_min_count values are not compatible');
      expect(result.body.error[0].property).to.equal('specific_date.initial_min_instance_count');
      expect(result.body.error[0].stack).to.equal('initial_min_instance_count 1 is lower than instance_min_count 2 in specific_date :[1]');
      done();
    });
  });

  it('Should fail to validate the policy for non integer initial_min_instance_count in recurring_schedule',function(done){
      fakePolicy.schedules.recurring_schedule[0].initial_min_instance_count = 1.3;
      request(app)
      .put('/v1/apps/12359/policy',validationMiddleware)
      .send(fakePolicy)
      .end(function(error,result) {
        expect(result.statusCode).to.equal(400);
        expect(result.body.success).to.equal(false);
        expect(result.body.error).to.not.be.null;
        expect(result.body.error[0].message).to.equal('is not of a type(s) integer');
        expect(result.body.error[0].property).to.equal('instance.schedules.recurring_schedule[0].initial_min_instance_count');
        expect(result.body.error[0].stack).to.equal('instance.schedules.recurring_schedule[0].initial_min_instance_count is not of a type(s) integer');
        done();
      });
  });

  it('Should fail to validate the policy for non integer initial_min_instance_count in specific_date',function(done){
    fakePolicy.schedules.specific_date[1].initial_min_instance_count = 1.3;
    request(app)
    .put('/v1/apps/12359/policy',validationMiddleware)
    .send(fakePolicy)
    .end(function(error,result) {
      expect(result.statusCode).to.equal(400);
      expect(result.body.success).to.equal(false);
      expect(result.body.error).to.not.be.null;
      expect(result.body.error[0].message).to.equal('is not of a type(s) integer');
      expect(result.body.error[0].property).to.equal('instance.schedules.specific_date[1].initial_min_instance_count');
      expect(result.body.error[0].stack).to.equal('instance.schedules.specific_date[1].initial_min_instance_count is not of a type(s) integer');
      done();
    });
  });

  it('should fail to validate policy where start_date and end_date are same but start_time is after end_time in recurring_schedule', function(done) {
    fakePolicy.schedules.recurring_schedule[0].start_date = '2016-07-23';
    fakePolicy.schedules.recurring_schedule[0].end_date = '2016-07-23';
    fakePolicy.schedules.recurring_schedule[0].start_time = '18:00';
    fakePolicy.schedules.recurring_schedule[0].end_time = '10:00';
    request(app)
    .put('/v1/apps/12359/policy',validationMiddleware)
    .send(fakePolicy)
    .end(function(error,result) {
      expect(result.statusCode).to.equal(400);
      expect(result.body.success).to.equal(false);
      expect(result.body.error).to.not.be.null;
      expect(result.body.error[0].message).to.equal('recurring_schedule.start_time and recurring_schedule.end_time values are not compatible');
      expect(result.body.error[0].property).to.equal('recurring_schedule.start_time');
      expect(result.body.error[0].stack).to.equal('start_time 18:00 is same or after end_time 10:00 in recurring_schedule :[0]');
      done();
    });
  });

  it('should fail to validate policy where start_date and end_date are same and start_time is same as end_time in recurring_schedule', function(done) {
    fakePolicy.schedules.recurring_schedule[0].start_date = '2016-07-23';
    fakePolicy.schedules.recurring_schedule[0].end_date = '2016-07-23';
    fakePolicy.schedules.recurring_schedule[0].start_time = '10:00';
    fakePolicy.schedules.recurring_schedule[0].end_time = '10:00';
    request(app)
    .put('/v1/apps/12359/policy',validationMiddleware)
    .send(fakePolicy)
    .end(function(error,result) {
      expect(result.statusCode).to.equal(400);
      expect(result.body.success).to.equal(false);
      expect(result.body.error).to.not.be.null;
      done();
    });
  });
  it('should validate policy where start_date and end_date are same but start_time is before end_time in recurring_schedule', function(done) {
    nock(schedulerURI)
    .put('/v2/schedules/12359')
    .query({'guid':/.*/})
    .reply(200);
    fakePolicy.schedules.recurring_schedule[0].start_date = '2016-07-23';
    fakePolicy.schedules.recurring_schedule[0].end_date = '2016-07-23';
    fakePolicy.schedules.recurring_schedule[0].start_time = '10:00';
    fakePolicy.schedules.recurring_schedule[0].end_time = '18:00';
    request(app)
    .put('/v1/apps/12359/policy',validationMiddleware)
    .send(fakePolicy)
    .end(function(error,result) {
      expect(result.statusCode).to.equal(201);
      expect(result.body.success).to.equal(true);
      expect(result.body.error).to.be.null;
      done();
    });
  });
});
