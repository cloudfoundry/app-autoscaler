'use strict';


var request = require('supertest');
var expect = require('chai').expect;
var fs = require('fs');
var path = require('path');
var _ = require('underscore');
var settings = require(path.join(__dirname, '../../../lib/config/setting.js'))((JSON.parse(
  fs.readFileSync(path.join(__dirname, '../../../config/settings.json'), 'utf8'))));
var relativePath = path.relative(process.cwd(), path.join(__dirname, "../../../../test-certs"));
var testSetting = require(path.join(__dirname, '../test.helper.js'))(relativePath, settings);

var API = require('../../../app.js');
var app;
var publicApp;
var healthApp;

var servers;
var logger = require('../../../lib/log/logger');
var policy = require('../../../lib/models')(testSetting.db).policy_json;
var validationMiddleware = require('../../../lib/validation/validationMiddleware');
var nock = require('nock');
var schedulerURI = testSetting.scheduler.uri;

var rewire = require('rewire');
var schemaValidatorPrivate = rewire('../../../lib/validation/schemaValidator');

describe('Validate Policy JSON Schema structure', function () {
  var policyContent;
  var fakePolicy;
  var policyTypes = ["full", "scaling_rules_only", "schedules_only"];
  var scheduleTypes = ["full", "recurring_only", "specific_date_only"];

  var buildTestPolicy = function (policy, type) {
    if (type === "scaling_rules_only") {
      delete policy.schedules;
    } else if (type === "schedules_only") {
      delete policy.scaling_rules;
    }
    return policy;
  }
  var buildTestSchedule = function (policy, type) {
    if (type === "recurring_only") {
      delete policy.schedules.specific_date;
    } else if (type === "specific_date_only") {
      delete policy.schedules.recurring_schedule;
    }
    return policy;
  }

  before(function () {
    servers = API(testSetting, function () { });
    app = servers.internalServer;
    publicApp = servers.publicServer;
    healthApp = servers.healthServer;
    policyContent = fs.readFileSync(__dirname + '/../fakePolicy.json', 'utf8');
  });
  after(function(done) {
    app.close(function() {
      publicApp.close(function(){
        healthApp.close(done);
      });
    });
  })
  beforeEach(function () {
    if (testSetting.serviceOffering) {
      nock(testSetting.serviceOffering.serviceBroker.uri)
        .get(/\/v1\/apps\/.+\/service_bindings/)
        .reply(200, { "message": "binding_info_exists" });
    }
    fakePolicy = JSON.parse(policyContent);
    return policy.truncate();
  });

  beforeEach(function () {
    nock(schedulerURI)
      .put('/v1/apps/fakeID/schedules')
      .query({ 'guid': /.*/ })
      .reply(200);
  });

  context('policy schema & attribute', function () {
    var requiredProperties = schemaValidatorPrivate.__get__('getPolicySchema')().required;
    var allProperties = Object.keys(schemaValidatorPrivate.__get__('getPolicySchema')().properties);

    var validInstanceCountSettings = [
      { instance_min_count: 1, instance_max_count: 2 },
      { instance_min_count: 1000, instance_max_count: 2000 },
    ];
    var invalidInstanceCountSettings = [
      { instance_min_count: 0, instance_max_count: 1, error: "outofrange" },
      { instance_min_count: -1, instance_max_count: 1, error: "outofrange" },
      { instance_min_count: 1, instance_max_count: 1, error: "incompatible" },
      { instance_min_count: 5, instance_max_count: 1, error: "incompatible" },
    ];

    context('should validate policy schema successfully', function () {
      policyTypes.forEach(function (type) {
        it('succeed, policy type ' + type, function (done) {
          buildTestPolicy(fakePolicy, type);
          request(app)
            .put('/v1/apps/fakeID/policy', validationMiddleware)
            .send(fakePolicy)
            .end(function (error, result) {
              expect(result.statusCode).to.equal(201);
              done();
            });
        });

        validInstanceCountSettings.forEach(function (setting) {
          var instance_min_count = setting.instance_min_count;
          var instance_max_count = setting.instance_max_count;
          it('succeed with instance_min_count =' + instance_min_count + ',instance_max_count=' + instance_max_count + ', policy type ' + type, function (done) {
            buildTestPolicy(fakePolicy, type);
            fakePolicy.instance_min_count = instance_min_count;
            fakePolicy.instance_max_count = instance_max_count;
            request(app)
              .put('/v1/apps/fakeID/policy', validationMiddleware)
              .send(fakePolicy)
              .end(function (error, result) {
                expect(result.statusCode).to.equal(201);
                done();
              });
          });
        });
      });
    });

    context('should fail to validate policy', function () {

      it('failed with the absence of neither scaling_rules nor schedules  ', function (done) {
        delete fakePolicy.scaling_rules;
        delete fakePolicy.schedules;
        request(app)
          .put('/v1/apps/fakeID/policy', validationMiddleware)
          .send(fakePolicy)
          .end(function (error, result) {
            expect(result.statusCode).to.equal(400);
            expect(result.body.error).to.not.be.null;
            expect(result.body.error[0].message).to.equal('is not any of [subschema 0],[subschema 1]');
            expect(result.body.error[0].stack).to.equal('instance is not any of [subschema 0],[subschema 1]');
            expect(result.body.error[0].schema).to.equal('/policySchema');
            done();
          });
      });

      policyTypes.forEach(function (type) {
        allProperties.forEach(function (property) {
          it('failed with property instance.' + property + ' set to NULL, policy type :' + type, function (done) {
            buildTestPolicy(fakePolicy, type);
            fakePolicy[property] = null;
            request(app)
              .put('/v1/apps/fakeID/policy', validationMiddleware)
              .send(fakePolicy)
              .end(function (error, result) {
                expect(result.statusCode).to.equal(400);
                expect(result.body.error).to.not.be.null;
                expect(result.body.error[0].message).to.contains('is not of a type(s)');
                expect(result.body.error[0].stack).to.contains('instance.' + property + ' is not of a type(s)');
                done();
              });
          });
        });

        requiredProperties.forEach(function (required) {
          it('failed with the absence of required property ' + required + ', policy type :' + type, function (done) {
            buildTestPolicy(fakePolicy, type);
            delete fakePolicy[required];
            request(app)
              .put('/v1/apps/fakeID/policy', validationMiddleware)
              .send(fakePolicy)
              .end(function (error, result) {
                expect(result.statusCode).to.equal(400);
                expect(result.body.error).to.not.be.null;
                expect(result.body.error[0].message).to.equal('requires property "' + required + '"');
                expect(result.body.error[0].stack).to.equal('instance requires property "' + required + '"');
                expect(result.body.error[0].schema).to.equal('/policySchema');
                done();
              });
          });
        });

        invalidInstanceCountSettings.forEach(function (setting) {
          var instance_min_count = setting.instance_min_count;
          var instance_max_count = setting.instance_max_count;
          it('failed with invalid setting instance_min_count=' + instance_min_count + ',instance_max_count=' + instance_max_count + ', policy type ' + type, function (done) {
            buildTestPolicy(fakePolicy, type);
            fakePolicy.instance_min_count = instance_min_count;
            fakePolicy.instance_max_count = instance_max_count;
            request(app)
              .put('/v1/apps/fakeID/policy', validationMiddleware)
              .send(fakePolicy)
              .end(function (error, result) {
                expect(result.statusCode).to.equal(400);
                expect(result.body.error).to.not.be.null;
                if (setting.error === "outofrange") {
                  expect(result.body.error[0].message).to.equal('must have a minimum value of 1');
                  expect(result.body.error[0].stack).to.equal('instance.instance_min_count must have a minimum value of 1');
                }
                if (setting.error === "notCompatible") {
                  expect(result.body.error[0].message).to.equal('instance_min_count and instance_max_count values are not compatible');
                  expect(result.body.error[0].stack).to.equal('instance_min_count ' + instance_min_count + ' is higher or equal to instance_max_count ' + instance_max_count + ' in policy_json');
                }
                done();
              });
          });
        });

      });
    });
  });


  context('scaling rule schema', function () {

    var requiredProperties = schemaValidatorPrivate.__get__('getScalingRuleSchema')(settings).required;
    var allProperties = Object.keys(schemaValidatorPrivate.__get__('getScalingRuleSchema')(settings).properties);
    var validMetricTypes = ['memoryutil','memoryused','responsetime','throughput','custom_metric'];
    var validMetricThresholdSettings = [
      { metric_type: "memoryutil", thresholds: [30, 100] },
      { metric_type: "memoryused", thresholds: [30, 100, 1000] },
      { metric_type: "responsetime", thresholds: [30, 100, 1000] },
      { metric_type: "throughput", thresholds: [30, 100, 1000] },
    ];
    var validDurationSettings = [
      { duration_type: "breach_duration_secs", values: [30, 1800, 3600] },
      { duration_type: "cool_down_secs", values: [30, 1800, 3600] }
    ];
    var validOperator = schemaValidatorPrivate.__get__('getValidOperators')();
    var validAdjustmentPattern = ['-1', '+1', '-19', '+19', '-10000', '+99999','-10%','+10%'];

    var invalidMetricThresholdSettings = [
      { metric_type: "memoryutil", thresholds: [-1, 0, 1000] },
      { metric_type: "memoryused", thresholds: [-1, 0] },
      { metric_type: "responsetime", thresholds: [-1, 0] },
      { metric_type: "throughput", thresholds: [-1, 0] },
    ];
    var invalidDurationSettings = [
      { duration_type: "breach_duration_secs", values: [1, 29, 3601] },
      { duration_type: "cool_down_secs", values: [1, 29, 3601] }
    ];
    var invalidAdjustmentPattern = ['0', '1', '+09', '|10', '-8%%'];

    policyTypes.forEach(function (type) {
      if (type === "schedules_only") {
        return;
      }
      context('should succeeded', function () {
        it('succeed with the absence of all optional fields, policy type ' + type, function (done) {
          buildTestPolicy(fakePolicy, type);
          for (let i = 0; i < fakePolicy.scaling_rules.length; i++) {
            delete fakePolicy.scaling_rules[i].breach_duration_secs
            delete fakePolicy.scaling_rules[i].cool_down_secs
          }
          request(app)
            .put('/v1/apps/fakeID/policy', validationMiddleware)
            .send(fakePolicy)
            .end(function (error, result) {
              expect(result.statusCode).to.equal(201);
              done();
            });
        });

        validMetricTypes.forEach(function (metricType) {
          it('succeed with valid metric type ' + metricType + ', policy type ' + type, function (done) {
            buildTestPolicy(fakePolicy, type);
            for (let i = 0; i < fakePolicy.scaling_rules.length; i++) {
              fakePolicy.scaling_rules[i].metric_type = metricType;
            }
            request(app)
              .put('/v1/apps/fakeID/policy', validationMiddleware)
              .send(fakePolicy)
              .end(function (error, result) {
                expect(result.statusCode).to.equal(201);
                done();
              });
          });
        });

        validMetricThresholdSettings.forEach(function (setting) {
          var metricType = setting.metric_type;
          setting.thresholds.forEach(function (threshold) {
            it('succeed with valid metric type ' + metricType + ', threshold ' + threshold + ', policy type ' + type, function (done) {
              buildTestPolicy(fakePolicy, type);
              fakePolicy.scaling_rules[0].metric_type = metricType;
              fakePolicy.scaling_rules[0].threshold = threshold;
              request(app)
                .put('/v1/apps/fakeID/policy', validationMiddleware)
                .send(fakePolicy)
                .end(function (error, result) {
                  expect(result.statusCode).to.equal(201);
                  done();
                });
            });
          });
        });
        validDurationSettings.forEach(function (setting) {
          var durationType = setting.duration_type;
          setting.values.forEach(function (value) {
            it('succeed with ' + durationType + '=' + value + ', policy type ' + type, function (done) {
              buildTestPolicy(fakePolicy, type);
              fakePolicy.scaling_rules[0][durationType] = value;
              request(app)
                .put('/v1/apps/fakeID/policy', validationMiddleware)
                .send(fakePolicy)
                .end(function (error, result) {
                  expect(result.statusCode).to.equal(201);
                  done();
                });
            });
          });
        });

        validOperator.forEach(function (operator) {
          it('succeed with valid operator ' + operator + ', policy type ' + type, function (done) {
            buildTestPolicy(fakePolicy, type);
            fakePolicy.scaling_rules[0].operator = operator;
            request(app)
              .put('/v1/apps/fakeID/policy', validationMiddleware)
              .send(fakePolicy)
              .end(function (error, result) {
                expect(result.statusCode).to.equal(201);
                done();
              });
          });
        });

        validAdjustmentPattern.forEach(function (adjustment) {
          it('succeed with valid adjustment ' + adjustment + ', policy type ' + type, function (done) {
            buildTestPolicy(fakePolicy, type);
            fakePolicy.scaling_rules[0].adjustment = adjustment;
            request(app)
              .put('/v1/apps/fakeID/policy', validationMiddleware)
              .send(fakePolicy)
              .end(function (error, result) {
                expect(result.statusCode).to.equal(201);
                done();
              });
          });
        });
      });

      context('should fail to validate', function () {

        allProperties.forEach(function (property) {
          it('failed with property instance.scaling_rules[0].' + property + ' set to NULL, policy type :' + type, function (done) {
            buildTestPolicy(fakePolicy, type);
            fakePolicy.scaling_rules[0][property] = null;
            request(app)
              .put('/v1/apps/fakeID/policy', validationMiddleware)
              .send(fakePolicy)
              .end(function (error, result) {
                expect(result.statusCode).to.equal(400);
                expect(result.body.error).to.not.be.null;
                expect(result.body.error[0].message).to.contains('is not of a type(s)');
                expect(result.body.error[0].stack).to.contains('instance.scaling_rules[0].' + property + ' is not of a type(s)');
                done();
              });
          });
        });

        requiredProperties.forEach(function (required) {
          it('failed with the absence of required property scaling_rules[0].' + required + ', policy type :' + type, function (done) {
            buildTestPolicy(fakePolicy, type);
            delete fakePolicy.scaling_rules[0][required];
            request(app)
              .put('/v1/apps/fakeID/policy', validationMiddleware)
              .send(fakePolicy)
              .end(function (error, result) {
                expect(result.statusCode).to.equal(400);
                expect(result.body.error).to.not.be.null;
                expect(result.body.error[0].message).to.equal('requires property "' + required + '"');
                expect(result.body.error[0].stack).to.equal('instance.scaling_rules[0] requires property "' + required + '"');
                expect(result.body.error[0].schema).to.equal('/scaling_rules');
                done();
              });
          });
        });
        it('should fail with custom metric type. policy type ' + type, function (done) {
          buildTestPolicy(fakePolicy, type);
          fakePolicy.scaling_rules[0].metric_type = 'my metric $';
          request(app)
            .put('/v1/apps/fakeID/policy', validationMiddleware)
            .send(fakePolicy)
            .end(function (error, result) {
              expect(result.statusCode).to.equal(400);
              expect(result.body.error).to.not.be.null;
              expect(result.body.error[0].message).to.equal('does not match pattern "^[a-zA-Z0-9_]+$"');
              expect(result.body.error[0].stack).to.equal('instance.scaling_rules[0].metric_type does not match pattern "^[a-zA-Z0-9_]+$"');
              done();
            });
        });

        it('should not fail with custom metric type. policy type ' + type, function (done) {
          buildTestPolicy(fakePolicy, type);
          fakePolicy.scaling_rules[0].metric_type = 'custom_metric';
          request(app)
            .put('/v1/apps/fakeID/policy', validationMiddleware)
            .send(fakePolicy)
            .end(function (error, result) {
              expect(result.statusCode).to.equal(201);
              done();
            });
        });

        invalidMetricThresholdSettings.forEach(function (setting) {
          var metricType = setting.metric_type;
          setting.thresholds.forEach(function (threshold) {
            it('failed with metric type ' + metricType + ', threshold ' + threshold + ', policy type ' + type, function (done) {
              buildTestPolicy(fakePolicy, type);
              fakePolicy.scaling_rules[0].metric_type = metricType;
              fakePolicy.scaling_rules[0].threshold = threshold;
              request(app)
                .put('/v1/apps/fakeID/policy', validationMiddleware)
                .send(fakePolicy)
                .end(function (error, result) {
                  expect(result.statusCode).to.equal(400);
                  expect(result.body.error).to.not.be.null;
                  expect(result.body.error[0].property).to.equal('scaling_rules[0].threshold');
                  if (metricType != "memoryutil") {
                    expect(result.body.error[0].message).to.equal('threshold value for metric_type ' + metricType + ' should be greater than 0');
                    expect(result.body.error[0].stack).to.equal('scaling_rules[0].threshold value should be greater than 0');
                  } else {
                    expect(result.body.error[0].message).to.equal('threshold value for metric_type memoryutil should be greater than 0 and less than equal to 100');
                    expect(result.body.error[0].stack).to.equal('scaling_rules[0].threshold value should be greater than 0 and less than equal to 100');
                  }
                  done();
                });
            });
          });
        });

        invalidDurationSettings.forEach(function (setting) {
          var durationType = setting.duration_type;
          setting.values.forEach(function (value) {
            it('failed with ' + durationType + '=' + value + ', policy type ' + type, function (done) {
              buildTestPolicy(fakePolicy, type);
              fakePolicy.scaling_rules[0][durationType] = value;
              request(app)
                .put('/v1/apps/fakeID/policy', validationMiddleware)
                .send(fakePolicy)
                .end(function (error, result) {
                  expect(result.statusCode).to.equal(400);
                  expect(result.body.error).to.not.be.null;
                  expect(result.body.error[0].property).to.equal('instance.scaling_rules[0].' + durationType);
                  expect(result.body.error[0].message === 'must have a minimum value of 30' ||
                    result.body.error[0].message === 'must have a maximum value of 3600').to.equal(true);
                  expect(result.body.error[0].stack === 'instance.scaling_rules[0].' + durationType + ' must have a minimum value of 30' ||
                    result.body.error[0].stack === 'instance.scaling_rules[0].' + durationType + ' must have a maximum value of 3600').to.equal(true);
                  done();
                });
            });
          });
        });

        it('failed with invalid Operator. policy type ' + type, function (done) {
          buildTestPolicy(fakePolicy, type);
          fakePolicy.scaling_rules[0].operator = '<>';
          request(app)
            .put('/v1/apps/fakeID/policy', validationMiddleware)
            .send(fakePolicy)
            .end(function (error, result) {
              expect(result.statusCode).to.equal(400);
              expect(result.body.error).to.not.be.null;
              expect(result.body.error[0].property).to.equal('instance.scaling_rules[0].operator');
              expect(result.body.error[0].message).to.equal('is not one of enum values: <,>,<=,>=');
              expect(result.body.error[0].stack).to.equal('instance.scaling_rules[0].operator is not one of enum values: <,>,<=,>=');
              done();
            });
        });

        invalidAdjustmentPattern.forEach(function (adjustment) {
          it('failed with invalid adjustment ' + adjustment + ', policy type ' + type, function (done) {
            buildTestPolicy(fakePolicy, type);
            fakePolicy.scaling_rules[0].adjustment = adjustment;
            request(app)
              .put('/v1/apps/fakeID/policy', validationMiddleware)
              .send(fakePolicy)
              .end(function (error, result) {
                expect(result.statusCode).to.equal(400);
                expect(result.body.error).to.not.be.null;
                expect(result.body.error[0].property).to.equal('instance.scaling_rules[0].adjustment');
                expect(result.body.error[0].message).to.equal('does not match pattern "^[-+][1-9]+[0-9]*[%]?$"');
                expect(result.body.error[0].stack).to.equal('instance.scaling_rules[0].adjustment does not match pattern "^[-+][1-9]+[0-9]*[%]?$"');
                done();
              });
          });
        });

      });
    });
  });


  context('scheduler schema', function () {
    var requiredProperties = schemaValidatorPrivate.__get__('getScheduleSchema')().required;
    var allProperties = Object.keys(schemaValidatorPrivate.__get__('getScheduleSchema')().properties);
    var validTimezone = [
      "Asia/Shanghai",
      "Etc/UTC",
      " E t c / U T C ",
    ]
    var invalidTimezone = [
      "(GMT+8:00)Asia/Shanghai",
      "Shanghai",
    ]

    policyTypes.forEach(function (type) {
      if (type === "scaling_rules_only") {
        return;
      }
      context('should validate schedule schema successfully', function () {
        scheduleTypes.forEach(function (scheduleType) {
          it('succeed, policy type ' + type + ', schedule type ' + scheduleType, function (done) {
            buildTestPolicy(fakePolicy, type);
            buildTestSchedule(fakePolicy, scheduleType);
            request(app)
              .put('/v1/apps/fakeID/policy', validationMiddleware)
              .send(fakePolicy)
              .end(function (error, result) {
                expect(result.statusCode).to.equal(201);
                done();
              });
          });

          validTimezone.forEach(function (timezone) {
            it('succeed with timezone=' + timezone + ' policy type ' + type + ', schedule type ' + scheduleType, function (done) {
              buildTestPolicy(fakePolicy, type);
              buildTestSchedule(fakePolicy, scheduleType);
              fakePolicy.schedules.timezone = timezone;
              request(app)
                .put('/v1/apps/fakeID/policy', validationMiddleware)
                .send(fakePolicy)
                .end(function (error, result) {
                  expect(result.statusCode).to.equal(201);
                  done();
                });
            });
          });
        });
      });

      context('should fail to validate', function () {
        it('failed with the absence of neither recurring_schedule nor specific_date ', function (done) {
          delete fakePolicy.schedules.recurring_schedule;
          delete fakePolicy.schedules.specific_date;
          request(app)
            .put('/v1/apps/fakeID/policy', validationMiddleware)
            .send(fakePolicy)
            .end(function (error, result) {
              expect(result.statusCode).to.equal(400);
              expect(result.body.error).to.not.be.null;
              expect(result.body.error[0].message).to.equal('is not any of [subschema 0],[subschema 1]');
              expect(result.body.error[0].stack).to.equal('instance.schedules is not any of [subschema 0],[subschema 1]');
              expect(result.body.error[0].schema).to.equal('/schedules');
              done();
            });
        });

        scheduleTypes.forEach(function (scheduleType) {

          allProperties.forEach(function (property) {
            it('failed with property instance.schedules.' + property + ' set to NULL, policy type :' + type, function (done) {
              buildTestPolicy(fakePolicy, type);
              fakePolicy.schedules[property] = null;
              request(app)
                .put('/v1/apps/fakeID/policy', validationMiddleware)
                .send(fakePolicy)
                .end(function (error, result) {
                  expect(result.statusCode).to.equal(400);
                  expect(result.body.error).to.not.be.null;
                  expect(result.body.error[0].message).to.contains('is not of a type(s)');
                  expect(result.body.error[0].stack).to.contains('instance.schedules.' + property + ' is not of a type(s)');
                  done();
                });
            });
          });

          requiredProperties.forEach(function (required) {
            it('failed with the absence of required property schedules.' + required + ', policy type :' + type, function (done) {
              buildTestPolicy(fakePolicy, type);
              buildTestSchedule(fakePolicy, scheduleType);
              delete fakePolicy.schedules[required];
              request(app)
                .put('/v1/apps/fakeID/policy', validationMiddleware)
                .send(fakePolicy)
                .end(function (error, result) {
                  expect(result.statusCode).to.equal(400);
                  expect(result.body.error).to.not.be.null;
                  done();
                });
            });
          });

          invalidTimezone.forEach(function (timezone) {
            it('failed with invalid timezone=' + timezone + ' policy type ' + type + ', schedule type ' + scheduleType, function (done) {
              buildTestPolicy(fakePolicy, type);
              buildTestSchedule(fakePolicy, scheduleType);
              fakePolicy.schedules.timezone = timezone;
              request(app)
                .put('/v1/apps/fakeID/policy', validationMiddleware)
                .send(fakePolicy)
                .end(function (error, result) {
                  expect(result.statusCode).to.equal(400);
                  expect(result.body.error).to.not.be.null;
                  expect(result.body.error[0].message).to.equal('does not conform to the "timeZoneFormat" format');
                  expect(result.body.error[0].property).to.equal('instance.schedules.timezone');
                  expect(result.body.error[0].stack).to.equal('instance.schedules.timezone does not conform to the "timeZoneFormat" format');
                  done();
                });
            });
          });

        });
      });
    });
  });

  context('recurring schedule schema', function () {
    var requiredProperties = schemaValidatorPrivate.__get__('getRecurringSchema')().required;
    var allProperties = Object.keys(schemaValidatorPrivate.__get__('getRecurringSchema')().properties);
    var validTimeSettings = [
      { start_time: "00:00", end_time: "00:01" },
      { start_time: "23:00", end_time: "24:00" },
      { start_time: "09:00", end_time: "10:00" },
    ]
    var validDateSettings = [
      { start_date: "2018-01-01", end_date: "2018-12-31" },
      { start_date: "2018-01-01", end_date: "2018-01-01" },
    ]
    var validInstanceCountSettings = [
      { instance_min_count: 1, instance_max_count: 5, initial_min_instance_count: 3 },
      { instance_min_count: 10000, instance_max_count: 50000, initial_min_instance_count: 20000 },
      { instance_min_count: 3, instance_max_count: 5, initial_min_instance_count: 5 },
      { instance_min_count: 3, instance_max_count: 5, initial_min_instance_count: 3 },
    ];
    var validDayOfWeekSetting = [
      [1],
      [1, 3, 5, 7],
      [1, 2, 3, 4, 5, 6, 7],
      [7],
    ]
    var validDayOfMonthSetting = [
      [1],
      [1, 2, 3, 4, 5],
      [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31],
      [31],
    ]
    var validMultiRecurringScheduleSetting = [
      {
        schedule: [
          { start_time: "00:00", end_time: "01:00", days_of_week: [1, 2, 3] },
          { start_time: "00:00", end_time: "01:00", days_of_week: [4, 5, 6, 7] },
        ],
        description: "same time slot on different weekdays"
      },
      {
        schedule: [
          { start_time: "00:00", end_time: "01:00", days_of_week: [1, 2, 3] },
          { start_time: "01:01", end_time: "02:00", days_of_week: [1, 2, 3] },
        ],
        description: "different time slot on same weekdays"
      },
      {
        schedule: [
          { start_time: "00:00", end_time: "01:00", days_of_week: [1, 2, 3], start_date: "2018-01-01", end_date: "2018-02-01" },
          { start_time: "00:00", end_time: "01:00", days_of_week: [1, 2, 3], start_date: "2018-02-02", end_date: "2018-03-01" },
        ],
        description: "same time slot and weekdays with different date range"
      },
      {
        schedule: [
          { start_time: "00:00", end_time: "01:00", days_of_week: [1, 2, 3], start_date: "2018-01-01", end_date: "2018-02-01" },
          { start_time: "00:00", end_time: "01:00", days_of_week: [1, 2, 3], start_date: "2018-02-02" },
        ],
        description: "same time slot and weekdays with different date range and no end_date provided."
      },
      {
        schedule: [
          { start_time: "00:00", end_time: "01:00", days_of_week: [1, 2, 3], start_date: "2018-01-01", end_date: "2018-02-01" },
          { start_time: "00:00", end_time: "01:00", days_of_week: [1, 2, 3], start_date: "2018-02-02" },
          { start_time: "00:00", end_time: "01:00", days_of_week: [1, 2, 3], end_date: "2017-12-31" },
        ],
        description: "same time slot and weekdays with different date range and no start_date provided."
      },
      {
        schedule: [
          { start_time: "00:00", end_time: "01:00", days_of_month: [1, 2, 3], },
          { start_time: "00:00", end_time: "01:00", days_of_week: [1, 2, 3], },
        ],
        description: "same time slot with different days_of_month and days_of_week"
      },
      {
        schedule: [
          { start_time: "00:00", end_time: "01:00", days_of_month: [1, 2, 3], },
          { start_time: "00:00", end_time: "01:00", days_of_month: [29, 30, 31], },
        ],
        description: "same time slot with different days_of_month"
      },
      {
        schedule: [
          { start_time: "00:00", end_time: "01:00", days_of_month: [1, 2, 3], },
          { start_time: "01:01", end_time: "02:00", days_of_month: [1, 2, 3], },
        ],
        description: "differnt time slot with same days_of_month"
      },
      {
        schedule: [
          { start_time: "00:00", end_time: "01:00", days_of_month: [1, 2, 3], start_date: "2018-01-01", end_date: "2018-02-01" },
          { start_time: "00:00", end_time: "01:00", days_of_month: [1, 2, 3], start_date: "2018-02-02", end_date: "2018-03-01" },
          { start_time: "00:00", end_time: "01:00", days_of_month: [1, 2, 3], start_date: "2018-03-02" },
          { start_time: "00:00", end_time: "01:00", days_of_month: [1, 2, 3], end_date: "2017-12-31" },
        ],
        description: "same time slot and days_of_month with different time range"
      },
    ]

    var invalidTimeSettings = [
      { start_time: "+00:00", end_time: "23:59", error: "incorrectFormat" },
      { start_time: "-10:00", end_time: "23:59", error: "incorrectFormat" },
      { start_time: "25:61", end_time: "23:59", error: "incorrectFormat" },
      { start_time: "11:00 PM", end_time: "23:59", error: "incorrectFormat" },
      { start_time: "23:00", end_time: "01:00", error: "notCompatible" },
      { start_time: "02:00", end_time: "02:00", error: "notCompatible" },
      { start_time: "02:00", end_time: "01:59", error: "notCompatible" },
    ]
    var invalidDateSettings = [
      { start_date: "2018-1-1", end_date: "2018-12-31", error: "incorrectFormat" },
      { start_date: "20180101", end_date: "2018-12-31", error: "incorrectFormat" },
      { start_date: "2018-01-01T10:00", end_date: "2018-12-31", error: "incorrectFormat" },
      { start_date: "2018-02-29", end_date: "2018-12-31", error: "incorrectFormat" },
      { start_date: "2018-01-32", end_date: "2018-12-31", error: "incorrectFormat" },
      { start_date: "31-01-2018", end_date: "2018-12-31", error: "incorrectFormat" },
      { start_date: "2018-01-02", end_date: "2018-01-01", error: "notCompatible" },
    ]
    var invalidInstanceCountSettings = [
      { instance_min_count: 0, instance_max_count: 1, initial_min_instance_count: 5, error: "outofrange" },
      { instance_min_count: -1, instance_max_count: 1, initial_min_instance_count: 5, error: "outofrange" },
      { instance_min_count: 3, instance_max_count: 5, initial_min_instance_count: 2, error: "incompatible" },
      { instance_min_count: 3, instance_max_count: 5, initial_min_instance_count: 6, error: "incompatible" },
    ];
    var invalidDayOfWeekSetting = [
      { value: [0], error: "outofrange" },
      { value: [1, 2, 3, 8], error: "outofrange" },
      { value: [1, 2, 2, 3], error: "duplicate" },
      { value: "invalid", error: "invalidType" },
      { value: 1, error: "invalidType" },
    ]

    var invalidDayOfMonthSetting = [
      { value: [0, 1, 2, 3], error: "outofrange" },
      { value: [32], error: "outofrange" },
      { value: [1, 30, 30, 31], error: "duplicate" },
      { value: "invalid", error: "invalidType" },
      { value: 1, error: "invalidType" },
    ]

    var invalidMultiRecurringScheduleSetting = [
      {
        schedule: [
          { start_time: "00:00", end_time: "01:00", days_of_week: [1, 2, 3] },
          { start_time: "00:00", end_time: "01:00", days_of_week: [3, 4, 5, 6, 7] },
        ],
        description: "same time slot on overlapped weekdays"
      },
      {
        schedule: [
          { start_time: "00:00", end_time: "01:00", days_of_week: [1, 2, 3] },
          { start_time: "01:00", end_time: "02:00", days_of_week: [1, 2, 3] },
        ],
        description: "overlapped time slot when recurring_schedule[0].end_time == recurring_schedule[1].start_time on the same weekdays"
      },
      {
        schedule: [
          { start_time: "00:00", end_time: "01:00", days_of_week: [1, 2, 3], start_date: "2018-01-01", end_date: "2018-02-01" },
          { start_time: "00:00", end_time: "01:00", days_of_week: [1, 2, 3], start_date: "2018-02-01", end_date: "2018-03-01" },
        ],
        description: "same time slot and weekdays with overlapped date range when recurring_schedule[0].end_date == recurring_schedule[1].start_date"
      },
      {
        schedule: [
          { start_time: "00:00", end_time: "01:00", days_of_week: [1, 2, 3], start_date: "2018-01-01", end_date: "2018-02-01" },
          { start_time: "00:00", end_time: "01:00", days_of_week: [1, 2, 3], start_date: "2018-01-01" },
        ],
        description: "same time slot and weekdays with overlapped data range & one of the end_date is not defined"
      },
      {
        schedule: [
          { start_time: "00:00", end_time: "01:00", days_of_week: [1, 2, 3], start_date: "2018-01-01", end_date: "2018-02-01" },
          { start_time: "00:00", end_time: "01:00", days_of_week: [1, 2, 3], end_date: "2018-01-15" },
        ],
        description: "same time slot and weekdays with overlapped data range & one of the start_date is not defined"
      },
      {
        schedule: [
          { start_time: "00:00", end_time: "01:00", days_of_month: [1, 2, 3, 29], },
          { start_time: "00:00", end_time: "01:00", days_of_month: [29, 30, 31], },
        ],
        description: " same time slot with overlapped days_of_month"
      },
      {
        schedule: [
          { start_time: "00:00", end_time: "01:00", days_of_month: [1, 2, 3], },
          { start_time: "01:00", end_time: "02:00", days_of_month: [1, 2, 3], },
        ],
        description: "overlapped time slot with same days_of_month"
      },
      {
        schedule: [
          { start_time: "00:00", end_time: "01:00", days_of_month: [1, 2, 3], start_date: "2018-01-01", end_date: "2018-02-01" },
          { start_time: "00:00", end_time: "01:00", days_of_month: [1, 2, 3], start_date: "2018-02-01", end_date: "2018-03-01" },
          { start_time: "00:00", end_time: "01:00", days_of_month: [1, 2, 3], start_date: "2018-03-01" },
          { start_time: "00:00", end_time: "01:00", days_of_month: [1, 2, 3], end_date: "2018-12-31" },
        ],
        description: "same time slot and days_of_month with overlapped time range"
      },
    ]


    policyTypes.forEach(function (type) {
      if (type === "scaling_rules_only") {
        return;
      }
      scheduleTypes.forEach(function (scheduleType) {
        if (scheduleType === "specific_date_only") {
          return;
        }

        context('should validate recurring schedule schema successfully', function () {

          it('succeed with the absence of all optional fields, policy type ' + type + ', schedule type ' + scheduleType, function (done) {
            buildTestPolicy(fakePolicy, type);
            buildTestSchedule(fakePolicy, scheduleType);
            for (let i = 0; i < fakePolicy.schedules.recurring_schedule.length; i++) {
              delete fakePolicy.schedules.recurring_schedule[i].start_date;
              delete fakePolicy.schedules.recurring_schedule[i].end_date;
              delete fakePolicy.schedules.recurring_schedule[i].initial_min_instance_count;
            }
            request(app)
              .put('/v1/apps/fakeID/policy', validationMiddleware)
              .send(fakePolicy)
              .end(function (error, result) {
                expect(result.statusCode).to.equal(201);
                done();
              });
          });

          validTimeSettings.forEach(function (setting) {
            it('succeed with schedules.recurring_schedule[0].start_time=' + setting.start_time + ', schedules.recurring_schedule[0].end_time=' + setting.end_time + ', policy type ' + type + ', schedule type ' + scheduleType, function (done) {
              buildTestPolicy(fakePolicy, type);
              buildTestSchedule(fakePolicy, scheduleType);
              fakePolicy.schedules.recurring_schedule[0].start_time = setting.start_time;
              fakePolicy.schedules.recurring_schedule[0].end_time = setting.end_time;
              request(app)
                .put('/v1/apps/fakeID/policy', validationMiddleware)
                .send(fakePolicy)
                .end(function (error, result) {
                  expect(result.statusCode).to.equal(201);
                  done();
                });
            });
          });
          validDateSettings.forEach(function (setting) {
            it('succeed with schedules.recurring_schedule[0].start_date=' + setting.start_date + ', schedules.recurring_schedule[0].end_date=' + setting.end_date + ', policy type ' + type + ', schedule type ' + scheduleType, function (done) {
              buildTestPolicy(fakePolicy, type);
              buildTestSchedule(fakePolicy, scheduleType);
              fakePolicy.schedules.recurring_schedule[0].start_date = setting.start_date;
              fakePolicy.schedules.recurring_schedule[0].end_date = setting.end_date;
              request(app)
                .put('/v1/apps/fakeID/policy', validationMiddleware)
                .send(fakePolicy)
                .end(function (error, result) {
                  expect(result.statusCode).to.equal(201);
                  done();
                });
            });
          });

          validInstanceCountSettings.forEach(function (setting) {
            var instance_min_count = setting.instance_min_count;
            var instance_max_count = setting.instance_max_count;
            var initial_min_instance_count = setting.initial_min_instance_count;
            it('succeed with instance_min_count=' + instance_min_count + ',initial_min_instance_count=' + initial_min_instance_count
              + ',instance_max_count=' + instance_max_count + ', policy type ' + type + ', schedule type ' + scheduleType, function (done) {
                buildTestPolicy(fakePolicy, type);
                buildTestSchedule(fakePolicy, scheduleType);
                fakePolicy.schedules.recurring_schedule[0].instance_min_count = instance_min_count;
                fakePolicy.schedules.recurring_schedule[0].initial_min_instance_count = initial_min_instance_count;
                fakePolicy.schedules.recurring_schedule[0].instance_max_count = instance_max_count;
                request(app)
                  .put('/v1/apps/fakeID/policy', validationMiddleware)
                  .send(fakePolicy)
                  .end(function (error, result) {
                    expect(result.statusCode).to.equal(201);
                    done();
                  });
              });
          });

          validDayOfWeekSetting.forEach(function (setting) {
            it('succeed with schedules.recurring_schedule[0].days_of_week=' + setting + ', policy type ' + type + ', schedule type ' + scheduleType, function (done) {
              buildTestPolicy(fakePolicy, type);
              buildTestSchedule(fakePolicy, scheduleType);
              fakePolicy.schedules.recurring_schedule = fakePolicy.schedules.recurring_schedule.slice(0, 1);
              delete fakePolicy.schedules.recurring_schedule[0].days_of_week;
              delete fakePolicy.schedules.recurring_schedule[0].days_of_month;
              fakePolicy.schedules.recurring_schedule[0].days_of_week = setting;
              request(app)
                .put('/v1/apps/fakeID/policy', validationMiddleware)
                .send(fakePolicy)
                .end(function (error, result) {
                  expect(result.statusCode).to.equal(201);
                  done();
                });
            });
          });

          validDayOfMonthSetting.forEach(function (setting) {
            it('succeed with schedules.recurring_schedule[0].days_of_month=' + setting + ', policy type ' + type + ', schedule type ' + scheduleType, function (done) {
              buildTestPolicy(fakePolicy, type);
              buildTestSchedule(fakePolicy, scheduleType);
              fakePolicy.schedules.recurring_schedule = fakePolicy.schedules.recurring_schedule.slice(0, 1);
              delete fakePolicy.schedules.recurring_schedule[0].days_of_week;
              delete fakePolicy.schedules.recurring_schedule[0].days_of_month;
              fakePolicy.schedules.recurring_schedule[0].days_of_month = setting;
              request(app)
                .put('/v1/apps/fakeID/policy', validationMiddleware)
                .send(fakePolicy)
                .end(function (error, result) {
                  expect(result.statusCode).to.equal(201);
                  done();
                });
            });
          });

          validMultiRecurringScheduleSetting.forEach(function (setting) {
            it('succeed with multiple schedules.recurring_schedule which defines ' + setting.description + ', policy type ' + type + ', schedule type ' + scheduleType, function (done) {
              buildTestPolicy(fakePolicy, type);
              buildTestSchedule(fakePolicy, scheduleType);
              var schedule = setting.schedule;
              if (fakePolicy.schedules.recurring_schedule.length > schedule.length) {
                fakePolicy.schedules.recurring_schedule = fakePolicy.schedules.recurring_schedule.slice(0, schedule.length);
              } else {
                for (let i = fakePolicy.schedules.recurring_schedule.length; i < schedule.length; i++) {
                  fakePolicy.schedules.recurring_schedule.push(_.clone(fakePolicy.schedules.recurring_schedule[0]));
                }
              }
              for (let i = 0; i < fakePolicy.schedules.recurring_schedule.length; i++) {
                delete fakePolicy.schedules.recurring_schedule[i].days_of_week;
                delete fakePolicy.schedules.recurring_schedule[i].days_of_month;
                delete fakePolicy.schedules.recurring_schedule[i].start_date;
                delete fakePolicy.schedules.recurring_schedule[i].end_date;
                for (var key in schedule[i]) {
                  fakePolicy.schedules.recurring_schedule[i][key] = schedule[i][key];
                }
              }
              request(app)
                .put('/v1/apps/fakeID/policy', validationMiddleware)
                .send(fakePolicy)
                .end(function (error, result) {
                  expect(result.statusCode).to.equal(201);
                  done();
                });
            });
          });

        });

        context('should fail to validate', function () {
          allProperties.forEach(function (property) {
            it('failed with property instance.schedules.recurring_schedule[0].' + property + ' set to NULL, policy type :' + type, function (done) {
              buildTestPolicy(fakePolicy, type);
              fakePolicy.schedules.recurring_schedule[0][property] = null;
              request(app)
                .put('/v1/apps/fakeID/policy', validationMiddleware)
                .send(fakePolicy)
                .end(function (error, result) {
                  expect(result.statusCode).to.equal(400);
                  expect(result.body.error).to.not.be.null;
                  if (property === "start_date" || property === "end_date") {
                    expect(result.body.error[0].message).to.contains('is not any of [subschema 0],[subschema 1]');
                    expect(result.body.error[0].stack).to.contains('instance.schedules.recurring_schedule[0].' + property + ' is not any of [subschema 0],[subschema 1]');
                  } else {
                    expect(result.body.error[0].message).to.contains('is not of a type(s)');
                    expect(result.body.error[0].stack).to.contains('instance.schedules.recurring_schedule[0].' + property + ' is not of a type(s)');
                  }
                  done();
                });
            });
          });

          requiredProperties.forEach(function (required) {
            it('failed with the absence of required property schedules.recurring_schedule.' + required + ', policy type :' + type + ', schedule type ' + scheduleType, function (done) {
              buildTestPolicy(fakePolicy, type);
              buildTestSchedule(fakePolicy, scheduleType);
              delete fakePolicy.schedules.recurring_schedule[0][required];
              request(app)
                .put('/v1/apps/fakeID/policy', validationMiddleware)
                .send(fakePolicy)
                .end(function (error, result) {
                  expect(result.statusCode).to.equal(400);
                  expect(result.body.error).to.not.be.null;
                  done();
                });
            });
          });

          it('failed when both days_of_week and days_of_month are presented , policy type :' + type + ', schedule type ' + scheduleType, function (done) {
            buildTestPolicy(fakePolicy, type);
            buildTestSchedule(fakePolicy, scheduleType);
            if (fakePolicy.schedules.recurring_schedule[0].days_of_week === undefined) {
              fakePolicy.schedules.recurring_schedule[0].days_of_week = [1, 2, 3];
            } else {
              fakePolicy.schedules.recurring_schedule[0].days_of_month = [1, 2, 3];
            }
            request(app)
              .put('/v1/apps/fakeID/policy', validationMiddleware)
              .send(fakePolicy)
              .end(function (error, result) {
                expect(result.statusCode).to.equal(400);
                expect(result.body.error).to.not.be.null;
                expect(result.body.error[0].message).to.equal('is not exactly one from [subschema 0],[subschema 1]');
                expect(result.body.error[0].property).to.equal('instance.schedules.recurring_schedule[0]');
                expect(result.body.error[0].stack).to.equal('instance.schedules.recurring_schedule[0] is not exactly one from [subschema 0],[subschema 1]');
                done();
              });
          });

          it('failed when neither days_of_week nor days_of_month is presented , policy type :' + type + ', schedule type ' + scheduleType, function (done) {
            buildTestPolicy(fakePolicy, type);
            buildTestSchedule(fakePolicy, scheduleType);
            delete fakePolicy.schedules.recurring_schedule[0].days_of_week;
            delete fakePolicy.schedules.recurring_schedule[0].days_of_month;
            request(app)
              .put('/v1/apps/fakeID/policy', validationMiddleware)
              .send(fakePolicy)
              .end(function (error, result) {
                expect(result.statusCode).to.equal(400);
                expect(result.body.error).to.not.be.null;
                expect(result.body.error[0].message).to.equal('is not exactly one from [subschema 0],[subschema 1]');
                expect(result.body.error[0].property).to.equal('instance.schedules.recurring_schedule[0]');
                expect(result.body.error[0].stack).to.equal('instance.schedules.recurring_schedule[0] is not exactly one from [subschema 0],[subschema 1]');
                done();
              });
          });

          invalidTimeSettings.forEach(function (setting) {
            it('failed with schedules.recurring_schedule[0].start_time=' + setting.start_time + ', schedules.recurring_schedule[0].end_time=' + setting.end_time + ', policy type ' + type + ', schedule type ' + scheduleType, function (done) {
              buildTestPolicy(fakePolicy, type);
              buildTestSchedule(fakePolicy, scheduleType);
              fakePolicy.schedules.recurring_schedule[0].start_time = setting.start_time;
              fakePolicy.schedules.recurring_schedule[0].end_time = setting.end_time;
              request(app)
                .put('/v1/apps/fakeID/policy', validationMiddleware)
                .send(fakePolicy)
                .end(function (error, result) {
                  expect(result.statusCode).to.equal(400);
                  expect(result.body.error).to.not.be.null;
                  if (setting.error === "incorrectFormat") {
                    expect(result.body.error[0].message).to.equal('does not conform to the "timeFormat" format');
                    expect(result.body.error[0].property).to.equal('instance.schedules.recurring_schedule[0].start_time');
                    expect(result.body.error[0].stack).to.equal('instance.schedules.recurring_schedule[0].start_time does not conform to the "timeFormat" format');
                  }
                  if (setting.error === "notCompatible") {
                    expect(result.body.error[0].message).to.equal('recurring_schedule.start_time and recurring_schedule.end_time values are not compatible');
                    expect(result.body.error[0].property).to.equal('recurring_schedule.start_time');
                    expect(result.body.error[0].stack).to.equal('start_time ' + setting.start_time + ' is same or after end_time ' + setting.end_time + ' in recurring_schedule :[0]');
                  }
                  done();
                });
            });
          });

          invalidDateSettings.forEach(function (setting) {
            it('failed with schedules.recurring_schedule[0].start_date=' + setting.start_date + ', schedules.recurring_schedule[0].end_date=' + setting.end_date + ', policy type ' + type + ', schedule type ' + scheduleType, function (done) {
              buildTestPolicy(fakePolicy, type);
              buildTestSchedule(fakePolicy, scheduleType);
              fakePolicy.schedules.recurring_schedule[0].start_date = setting.start_date;
              fakePolicy.schedules.recurring_schedule[0].end_date = setting.end_date;
              request(app)
                .put('/v1/apps/fakeID/policy', validationMiddleware)
                .send(fakePolicy)
                .end(function (error, result) {
                  expect(result.statusCode).to.equal(400);
                  expect(result.body.error).to.not.be.null;
                  if (setting.error === "incorrectFormat") {
                    expect(result.body.error[0].message).to.equal('is not any of [subschema 0],[subschema 1]');
                    expect(result.body.error[0].property).to.equal('instance.schedules.recurring_schedule[0].start_date');
                    expect(result.body.error[0].stack).to.equal('instance.schedules.recurring_schedule[0].start_date is not any of [subschema 0],[subschema 1]');
                  }
                  if (setting.error === "notCompatible") {
                    expect(result.body.error[0].message).to.equal('recurring_schedule.start_date and recurring_schedule.end_date values are not compatible');
                    expect(result.body.error[0].property).to.equal('recurring_schedule.start_date');
                    expect(result.body.error[0].stack).to.equal('start_date ' + setting.start_date + ' is after end_date ' + setting.end_date + ' in recurring_schedule :[0]');
                  }
                  done();
                });
            });
          });

          invalidInstanceCountSettings.forEach(function (setting) {
            var instance_min_count = setting.instance_min_count;
            var instance_max_count = setting.instance_max_count;
            var initial_min_instance_count = setting.initial_min_instance_count;
            it('failed with instance_min_count=' + instance_min_count + ',initial_min_instance_count=' + initial_min_instance_count
              + ',instance_max_count=' + instance_max_count + ', policy type ' + type + ', schedule type ' + scheduleType, function (done) {
                buildTestPolicy(fakePolicy, type);
                buildTestSchedule(fakePolicy, scheduleType);
                fakePolicy.schedules.recurring_schedule[0].instance_min_count = instance_min_count;
                fakePolicy.schedules.recurring_schedule[0].initial_min_instance_count = initial_min_instance_count;
                fakePolicy.schedules.recurring_schedule[0].instance_max_count = instance_max_count;
                request(app)
                  .put('/v1/apps/fakeID/policy', validationMiddleware)
                  .send(fakePolicy)
                  .end(function (error, result) {
                    expect(result.statusCode).to.equal(400);
                    expect(result.body.error).to.not.be.null;
                    if (setting.error === "outofrange") {
                      expect(result.body.error[0].message).to.equal('must have a minimum value of 1');
                      expect(result.body.error[0].property).to.equal('instance.schedules.recurring_schedule[0].instance_min_count');
                      expect(result.body.error[0].stack).to.equal('instance.schedules.recurring_schedule[0].instance_min_count must have a minimum value of 1');
                    }
                    if (setting.error === "notCompatible") {
                      expect(result.body.error[0].message).to.equal('recurring_schedule.initial_min_instance_count and recurring_schedule.instance_min_count values are not compatible');
                      expect(result.body.error[0].property).to.equal('recurring_schedule.start_date');
                    }
                    done();
                  });
              });
          });

          invalidDayOfWeekSetting.forEach(function (setting) {
            it('failed with schedules.recurring_schedule[0].days_of_week=' + setting.value + ', policy type ' + type + ', schedule type ' + scheduleType, function (done) {
              buildTestPolicy(fakePolicy, type);
              buildTestSchedule(fakePolicy, scheduleType);
              fakePolicy.schedules.recurring_schedule = fakePolicy.schedules.recurring_schedule.slice(0, 1);
              delete fakePolicy.schedules.recurring_schedule[0].days_of_week;
              delete fakePolicy.schedules.recurring_schedule[0].days_of_month;
              fakePolicy.schedules.recurring_schedule[0].days_of_week = setting.value;
              request(app)
                .put('/v1/apps/fakeID/policy', validationMiddleware)
                .send(fakePolicy)
                .end(function (error, result) {
                  expect(result.statusCode).to.equal(400);
                  expect(result.body.error).to.not.be.null;
                  if (setting.error === "invalidType") {
                    expect(result.body.error[0].message).to.equal('is not of a type(s) array');
                    expect(result.body.error[0].property).to.equal('instance.schedules.recurring_schedule[0].days_of_week');
                    expect(result.body.error[0].stack).to.equal('instance.schedules.recurring_schedule[0].days_of_week is not of a type(s) array');
                  }
                  if (setting.error === "outofrange") {
                    expect(result.body.error[0].message).to.equal('is not one of enum values: 1,2,3,4,5,6,7');
                    expect(result.body.error[0].property).to.contains('instance.schedules.recurring_schedule[0].days_of_week');
                    expect(result.body.error[0].stack).to.contains('not one of enum values: 1,2,3,4,5,6,7');
                  }
                  if (setting.error === "duplicate") {
                    expect(result.body.error[0].message).to.equal('contains duplicate item');
                    expect(result.body.error[0].property).to.equal('instance.schedules.recurring_schedule[0].days_of_week');
                    expect(result.body.error[0].stack).to.equal('instance.schedules.recurring_schedule[0].days_of_week contains duplicate item');
                  }
                  done();
                });
            });
          });

          invalidDayOfMonthSetting.forEach(function (setting) {
            it('failed with schedules.recurring_schedule[0].days_of_month=' + setting.value + ', policy type ' + type + ', schedule type ' + scheduleType, function (done) {
              buildTestPolicy(fakePolicy, type);
              buildTestSchedule(fakePolicy, scheduleType);
              fakePolicy.schedules.recurring_schedule = fakePolicy.schedules.recurring_schedule.slice(0, 1);
              delete fakePolicy.schedules.recurring_schedule[0].days_of_week;
              delete fakePolicy.schedules.recurring_schedule[0].days_of_month;
              fakePolicy.schedules.recurring_schedule[0].days_of_month = setting.value;
              request(app)
                .put('/v1/apps/fakeID/policy', validationMiddleware)
                .send(fakePolicy)
                .end(function (error, result) {
                  expect(result.statusCode).to.equal(400);
                  expect(result.body.error).to.not.be.null;
                  if (setting.error === "invalidType") {
                    expect(result.body.error[0].message).to.equal('is not of a type(s) array');
                    expect(result.body.error[0].property).to.equal('instance.schedules.recurring_schedule[0].days_of_month');
                    expect(result.body.error[0].stack).to.equal('instance.schedules.recurring_schedule[0].days_of_month is not of a type(s) array');
                  }
                  if (setting.error === "outofrange") {
                    expect(result.body.error[0].message).to.equal('is not one of enum values: 1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30,31');
                    expect(result.body.error[0].property).to.contains('instance.schedules.recurring_schedule[0].days_of_month');
                    expect(result.body.error[0].stack).to.contains('is not one of enum values: 1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30,31');
                  }
                  if (setting.error === "duplicate") {
                    expect(result.body.error[0].message).to.equal('contains duplicate item');
                    expect(result.body.error[0].property).to.equal('instance.schedules.recurring_schedule[0].days_of_month');
                    expect(result.body.error[0].stack).to.equal('instance.schedules.recurring_schedule[0].days_of_month contains duplicate item');
                  }
                  done();
                });
            });
          });

          invalidMultiRecurringScheduleSetting.forEach(function (setting) {
            it('failed with invalid multiple schedules.recurring_schedule which defines ' + setting.description + ', policy type ' + type + ', schedule type ' + scheduleType, function (done) {
              buildTestPolicy(fakePolicy, type);
              buildTestSchedule(fakePolicy, scheduleType);
              var schedule = setting.schedule;
              if (fakePolicy.schedules.recurring_schedule.length > schedule.length) {
                fakePolicy.schedules.recurring_schedule = fakePolicy.schedules.recurring_schedule.slice(0, schedule.length);
              } else {
                for (let i = fakePolicy.schedules.recurring_schedule.length; i < schedule.length; i++) {
                  fakePolicy.schedules.recurring_schedule.push(_.clone(fakePolicy.schedules.recurring_schedule[0]));
                }
              }
              for (let i = 0; i < fakePolicy.schedules.recurring_schedule.length; i++) {
                delete fakePolicy.schedules.recurring_schedule[i].days_of_week;
                delete fakePolicy.schedules.recurring_schedule[i].days_of_month;
                delete fakePolicy.schedules.recurring_schedule[i].start_date;
                delete fakePolicy.schedules.recurring_schedule[i].end_date;
                for (var key in schedule[i]) {
                  fakePolicy.schedules.recurring_schedule[i][key] = schedule[i][key];
                }
              }
              request(app)
                .put('/v1/apps/fakeID/policy', validationMiddleware)
                .send(fakePolicy)
                .end(function (error, result) {
                  expect(result.statusCode).to.equal(400);
                  expect(result.body.error).to.not.be.null;
                  expect(result.body.error[0].stack).to.contains("overlapped with time range");
                  done();
                });
            });
          });


        });
      });
    });
  });


  context('specific date schema', function () {
    var requiredProperties = schemaValidatorPrivate.__get__('getSpecificDateSchema')().required;
    var allProperties = Object.keys(schemaValidatorPrivate.__get__('getSpecificDateSchema')().properties);
    var validDateTimeSettings = [
      { start_date_time: "2017-12-31T00:00", end_date_time: "2017-12-31T24:00" },
      { start_date_time: "2017-12-31T23:59", end_date_time: "2018-01-01T00:01" },
    ]
    var validInstanceCountSettings = [
      { instance_min_count: 1, instance_max_count: 5, initial_min_instance_count: 3 },
      { instance_min_count: 10000, instance_max_count: 50000, initial_min_instance_count: 20000 },
      { instance_min_count: 3, instance_max_count: 5, initial_min_instance_count: 5 },
      { instance_min_count: 3, instance_max_count: 5, initial_min_instance_count: 3 },
    ];
    var validMultiSpecificDateSetting = [
      {
        schedule: [
          { start_date_time: "2017-12-31T00:00", end_date_time: "2017-12-31T23:58" },
          { start_date_time: "2017-12-31T23:59", end_date_time: "2018-01-01T00:00" },
          { start_date_time: "2018-01-01T00:01", end_date_time: "2018-01-02T23:59" },
        ],
        description: "different timeslot"
      },
    ]

    var invalidDateTimeSettings = [
      { start_date_time: "2017-12-3100:00", end_date_time: "2017-12-3123:59", error: "incorrectFormat" },
      { start_date_time: "2017-12-31 00:00", end_date_time: "2017-12-31 23:59", error: "incorrectFormat" },
      { start_date_time: "2017/12/31 00:00", end_date_time: "2017/12/31 23:59", error: "incorrectFormat" },
      { start_date_time: "2017-12-31T00:00+8:00", end_date_time: "2017-12-31T23:59+8:00", error: "incorrectFormat" },
      { start_date_time: "2017-1-1T00:00", end_date_time: "2017-12-31T01:00", error: "incorrectFormat" },
      { start_date_time: "2017-13-31T00:00", end_date_time: "2017-12-31T01:00", error: "incorrectFormat" },
      { start_date_time: "2017-12-32T00:00", end_date_time: "2017-12-31T01:00", error: "incorrectFormat" },
      { start_date_time: "0000-00-00T00:00", end_date_time: "2017-12-31T01:00", error: "incorrectFormat" },
      { start_date_time: "2018-02-29T00:00", end_date_time: "2018-02-29T01:00", error: "incorrectFormat" },
      { start_date_time: "2017-12-31T01:00", end_date_time: "2017-12-31T01:00", error: "notCompatible" },
      { start_date_time: "2017-12-31T02:00", end_date_time: "2017-12-31T01:00", error: "notCompatible" },
      { start_date_time: "2017-12-31T02:00", end_date_time: "2017-12-30T02:00", error: "notCompatible" },
    ]
    var invalidInstanceCountSettings = [
      { instance_min_count: 0, instance_max_count: 1, initial_min_instance_count: 5, error: "outofrange" },
      { instance_min_count: -1, instance_max_count: 1, initial_min_instance_count: 5, error: "outofrange" },
      { instance_min_count: 3, instance_max_count: 5, initial_min_instance_count: 2, error: "incompatible" },
      { instance_min_count: 3, instance_max_count: 5, initial_min_instance_count: 6, error: "incompatible" },
    ];

    var invalidMultiSpecificDateSetting = [
      {
        schedule: [
          { start_date_time: "2017-12-31T00:00", end_date_time: "2017-12-31T23:59" },
          { start_date_time: "2017-12-31T23:59", end_date_time: "2018-01-01T00:00" },
          { start_date_time: "2017-12-31T00:00", end_date_time: "2018-12-31T23:59" },
        ],
        description: "overlapped timeslot"
      },
    ]

    policyTypes.forEach(function (type) {
      if (type === "scaling_rules_only") {
        return;
      }
      scheduleTypes.forEach(function (scheduleType) {
        if (scheduleType === "recurring_only") {
          return;
        }

        context('should validate specific schedule schema successfully', function () {

          it('succeed with the absence of all optional fields, policy type ' + type + ', schedule type ' + scheduleType, function (done) {
            buildTestPolicy(fakePolicy, type);
            buildTestSchedule(fakePolicy, scheduleType);
            for (let i = 0; i < fakePolicy.schedules.specific_date.length; i++) {
              delete fakePolicy.schedules.specific_date[i].initial_min_instance_count;
            }
            request(app)
              .put('/v1/apps/fakeID/policy', validationMiddleware)
              .send(fakePolicy)
              .end(function (error, result) {
                expect(result.statusCode).to.equal(201);
                done();
              });
          });

          validDateTimeSettings.forEach(function (setting) {
            it('succeed with schedules.specific_date[0].start_date_time=' + setting.start_date_time + ', schedules.specific_date[0].end_date_time=' + setting.end_date_time + ', policy type ' + type + ', schedule type ' + scheduleType, function (done) {
              buildTestPolicy(fakePolicy, type);
              buildTestSchedule(fakePolicy, scheduleType);
              fakePolicy.schedules.specific_date[0].start_time = setting.start_date_time;
              fakePolicy.schedules.specific_date[0].end_date_time = setting.end_date_time;
              request(app)
                .put('/v1/apps/fakeID/policy', validationMiddleware)
                .send(fakePolicy)
                .end(function (error, result) {
                  expect(result.statusCode).to.equal(201);
                  done();
                });
            });
          });

          validInstanceCountSettings.forEach(function (setting) {
            var instance_min_count = setting.instance_min_count;
            var instance_max_count = setting.instance_max_count;
            var initial_min_instance_count = setting.initial_min_instance_count;
            it('succeed with instance_min_count=' + instance_min_count + ',initial_min_instance_count=' + initial_min_instance_count
              + ',instance_max_count=' + instance_max_count + ', policy type ' + type + ', schedule type ' + scheduleType, function (done) {
                buildTestPolicy(fakePolicy, type);
                buildTestSchedule(fakePolicy, scheduleType);
                fakePolicy.schedules.specific_date[0].instance_min_count = instance_min_count;
                fakePolicy.schedules.specific_date[0].initial_min_instance_count = initial_min_instance_count;
                fakePolicy.schedules.specific_date[0].instance_max_count = instance_max_count;
                request(app)
                  .put('/v1/apps/fakeID/policy', validationMiddleware)
                  .send(fakePolicy)
                  .end(function (error, result) {
                    expect(result.statusCode).to.equal(201);
                    done();
                  });
              });
          });

          validMultiSpecificDateSetting.forEach(function (setting) {
            it('succeed with multiple schedules.specific_date which defines ' + setting.description + ', policy type ' + type + ', schedule type ' + scheduleType, function (done) {
              buildTestPolicy(fakePolicy, type);
              buildTestSchedule(fakePolicy, scheduleType);
              var schedule = setting.schedule;
              if (fakePolicy.schedules.specific_date.length > schedule.length) {
                fakePolicy.schedules.specific_date = fakePolicy.schedules.specific_date.slice(0, schedule.length);
              } else {
                for (let i = fakePolicy.schedules.specific_date.length; i < schedule.length; i++) {
                  fakePolicy.schedules.specific_date.push(_.clone(fakePolicy.schedules.specific_date[0]));
                }
              }
              for (let i = 0; i < fakePolicy.schedules.specific_date.length; i++) {
                delete fakePolicy.schedules.specific_date[i].start_date_time;
                delete fakePolicy.schedules.specific_date[i].end_date_time;
                for (var key in schedule[i]) {
                  fakePolicy.schedules.specific_date[i][key] = schedule[i][key];
                }
              }
              request(app)
                .put('/v1/apps/fakeID/policy', validationMiddleware)
                .send(fakePolicy)
                .end(function (error, result) {
                  expect(result.statusCode).to.equal(201);
                  done();
                });
            });
          });

        });

        context('should fail to validate', function () {
          allProperties.forEach(function (property) {
            it('failed with property instance.schedules.specific_date[0].' + property + ' set to NULL, policy type :' + type, function (done) {
              buildTestPolicy(fakePolicy, type);
              fakePolicy.schedules.specific_date[0][property] = null;
              request(app)
                .put('/v1/apps/fakeID/policy', validationMiddleware)
                .send(fakePolicy)
                .end(function (error, result) {
                  expect(result.statusCode).to.equal(400);
                  expect(result.body.error).to.not.be.null;
                  expect(result.body.error[0].message).to.contains('is not of a type(s)');
                  expect(result.body.error[0].stack).to.contains('instance.schedules.specific_date[0].' + property + ' is not of a type(s)');
                  done();
                });
            });
          });

          requiredProperties.forEach(function (required) {
            it('failed with the absence of required property schedules.specific_date.' + required + ', policy type :' + type + ', schedule type ' + scheduleType, function (done) {
              buildTestPolicy(fakePolicy, type);
              buildTestSchedule(fakePolicy, scheduleType);
              delete fakePolicy.schedules.specific_date[0][required];
              request(app)
                .put('/v1/apps/fakeID/policy', validationMiddleware)
                .send(fakePolicy)
                .end(function (error, result) {
                  expect(result.statusCode).to.equal(400);
                  expect(result.body.error).to.not.be.null;
                  done();
                });
            });
          });

          invalidDateTimeSettings.forEach(function (setting) {
            it('failed with schedules.specific_date[0].start_date_time=' + setting.start_date_time + ', schedules.specific_date[0].end_date_time=' + setting.end_date_time + ', policy type ' + type + ', schedule type ' + scheduleType, function (done) {
              buildTestPolicy(fakePolicy, type);
              buildTestSchedule(fakePolicy, scheduleType);
              fakePolicy.schedules.specific_date[0].start_date_time = setting.start_date_time;
              fakePolicy.schedules.specific_date[0].end_date_time = setting.end_date_time;
              request(app)
                .put('/v1/apps/fakeID/policy', validationMiddleware)
                .send(fakePolicy)
                .end(function (error, result) {
                  expect(result.statusCode).to.equal(400);
                  expect(result.body.error).to.not.be.null;
                  if (setting.error === "incorrectFormat") {
                    expect(result.body.error[0].message).to.equal('does not conform to the "dateTimeFormat" format');
                    expect(result.body.error[0].property).to.equal('instance.schedules.specific_date[0].start_date_time');
                    expect(result.body.error[0].stack).to.equal('instance.schedules.specific_date[0].start_date_time does not conform to the "dateTimeFormat" format');
                  }
                  if (setting.error === "notCompatible") {
                    expect(result.body.error[0].message).to.equal('specific_date.start_date_time and specific_date.end_date_time values are not compatible');
                    expect(result.body.error[0].property).to.equal('specific_date.start_date_time');
                    expect(result.body.error[0].stack).to.equal('start_date_time ' + setting.start_date_time + ' is same or after end_date_time ' + setting.end_date_time + ' in specific_date :[0]');
                  }
                  done();
                });
            });
          });

          invalidInstanceCountSettings.forEach(function (setting) {
            var instance_min_count = setting.instance_min_count;
            var instance_max_count = setting.instance_max_count;
            var initial_min_instance_count = setting.initial_min_instance_count;
            it('failed with instance_min_count=' + instance_min_count + ',initial_min_instance_count=' + initial_min_instance_count
              + ',instance_max_count=' + instance_max_count + ', policy type ' + type + ', schedule type ' + scheduleType, function (done) {
                buildTestPolicy(fakePolicy, type);
                buildTestSchedule(fakePolicy, scheduleType);
                fakePolicy.schedules.specific_date[0].instance_min_count = instance_min_count;
                fakePolicy.schedules.specific_date[0].initial_min_instance_count = initial_min_instance_count;
                fakePolicy.schedules.specific_date[0].instance_max_count = instance_max_count;
                request(app)
                  .put('/v1/apps/fakeID/policy', validationMiddleware)
                  .send(fakePolicy)
                  .end(function (error, result) {
                    expect(result.statusCode).to.equal(400);
                    expect(result.body.error).to.not.be.null;
                    if (setting.error === "outofrange") {
                      expect(result.body.error[0].message).to.equal('must have a minimum value of 1');
                      expect(result.body.error[0].property).to.equal('instance.schedules.specific_date[0].instance_min_count');
                      expect(result.body.error[0].stack).to.equal('instance.schedules.specific_date[0].instance_min_count must have a minimum value of 1');
                    }
                    if (setting.error === "notCompatible") {
                      expect(result.body.error[0].message).to.equal('specific_date.initial_min_instance_count and specific_date.instance_min_count values are not compatible');
                      expect(result.body.error[0].property).to.equal('specific_date.start_date');
                    }
                    done();
                  });
              });
          });

          invalidMultiSpecificDateSetting.forEach(function (setting) {
            it('succeed with multiple schedules.specific_date which defines ' + setting.description + ', policy type ' + type + ', schedule type ' + scheduleType, function (done) {
              buildTestPolicy(fakePolicy, type);
              buildTestSchedule(fakePolicy, scheduleType);
              var schedule = setting.schedule;
              if (fakePolicy.schedules.specific_date.length > schedule.length) {
                fakePolicy.schedules.specific_date = fakePolicy.schedules.specific_date.slice(0, schedule.length);
              } else {
                for (let i = fakePolicy.schedules.specific_date.length; i < schedule.length; i++) {
                  fakePolicy.schedules.specific_date.push(_.clone(fakePolicy.schedules.specific_date[0]));
                }
              }
              for (let i = 0; i < fakePolicy.schedules.specific_date.length; i++) {
                delete fakePolicy.schedules.specific_date[i].start_date_time;
                delete fakePolicy.schedules.specific_date[i].end_date_time;
                for (var key in schedule[i]) {
                  fakePolicy.schedules.specific_date[i][key] = schedule[i][key];
                }
              }
              request(app)
                .put('/v1/apps/fakeID/policy', validationMiddleware)
                .send(fakePolicy)
                .end(function (error, result) {
                  expect(result.statusCode).to.equal(400);
                  expect(result.body.error).to.not.be.null;
                  expect(result.body.error[0].stack).to.contains("overlapped with date range");
                  done();
                });
            });
          });

        });
      });
    });
  });



});
