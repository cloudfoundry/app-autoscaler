'use strict';

var expect = require("chai").expect;
var path = require('path');
var fs = require('fs');
var settings = require(path.join(__dirname, '../../../lib/config/setting.js'))((JSON.parse(
  fs.readFileSync(path.join(__dirname, '../../../config/settings.json'), 'utf8'))));
var rewire = require('rewire');
var schemaValidatorPrivate = rewire('../../../lib/validation/schemaValidator');

var fs = require('fs');
var Ajv = require('ajv');
var validator = new Ajv({allErrors: true});
validator.addMetaSchema(require('ajv/lib/refs/json-schema-draft-06.json'));
var fakePolicy, total_schema, specific_date_schema, recurring_schedule_schema, schedules_schema, scaling_rules_schema;

describe('Validating Policy JSON schema construction',function(){
  
  it('Should validate the getSpecificDateSchema successfully',function(){
    var schema = schemaValidatorPrivate.__get__('getSpecificDateSchema')();  
    expect(schema.id).to.equal('/specific_date');
    expect(schema.properties.start_date_time).to.deep.equal({'type':'string','format':'dateTimeFormat'});
    expect(schema.properties.end_date_time).to.deep.equal({ 'type':'string','format':'dateTimeFormat' });
    expect(schema.properties.instance_min_count).to.deep.equal({ 'type':'integer','minimum':1 });
    expect(schema.properties.instance_max_count).to.deep.equal({ 'type':'integer' ,'minimum':1});
    expect(schema.properties.initial_min_instance_count).to.deep.equal({ 'type':'integer','minimum':1 });
    expect(schema.required).to.deep.equal( ['start_date_time','end_date_time','instance_min_count','instance_max_count']);
    
  });
  
  it('Should validate the getRecurringSchema successfully',function(){
    var schema = schemaValidatorPrivate.__get__('getRecurringSchema')();  
    var weekEnum = schemaValidatorPrivate.__get__('getDaysInWeeksInISOFormat')();
    var monthEnum =  schemaValidatorPrivate.__get__('getDaysInMonthInISOFormat')(); 
    expect(schema.id).to.equal('/recurring_schedule');
    expect(schema.properties.start_time).to.deep.equal({ 'type':'string','format':'timeFormat' });
    expect(schema.properties.end_time).to.deep.equal({ 'type':'string','format':'timeFormat' });
    expect(schema.properties.instance_min_count).to.deep.equal({ 'type':'integer','minimum':1});
    expect(schema.properties.instance_max_count).to.deep.equal({ 'type':'integer','minimum':1});
    expect(schema.properties.initial_min_instance_count).to.deep.equal({ 'type':'integer','minimum':1 });
    expect(schema.properties.days_of_week).to.deep.equal({ 'type':'array','uniqueItems': true, 'minItems': 1,
          'items':{ 'type':'number','enum':weekEnum } });
    expect(schema.properties.days_of_month).to.deep.equal({ 'type':'array','uniqueItems': true, 'minItems': 1,
          'items':{ 'type':'number','enum':monthEnum } });
    expect(schema.properties.start_date).to.deep.equal({'anyOf': [{ 'type':'string', 'format':'dateFormat' },{ 'type':'string', 'enum':[''] } ]});
    expect(schema.properties.end_date).to.deep.equal({'anyOf': [ { 'type':'string', 'format':'dateFormat' },{ 'type':'string', 'enum':[''] }]});
    expect(schema.required).to.deep.equal(['start_time','end_time','instance_min_count','instance_max_count']);
    expect(schema.oneOf).to.deep.equal([ {'required':['days_of_week']}, {'required':['days_of_month']} ]);;
  });
  
  it('should validate the getScheduleSchema successfully',function(){
    var schema = schemaValidatorPrivate.__get__('getScheduleSchema')();
    var timezoneEnum = schemaValidatorPrivate.__get__('getTimeZones')();
    expect(schema.id).to.equal('/schedules');
    expect(schema.properties.timezone).to.deep.equal({ 'type':'string','format': 'timeZoneFormat' });
    expect(schema.properties.recurring_schedule.type).to.equal('array');
    expect(schema.properties.recurring_schedule.items).to.deep.equal({ '$ref': '/recurring_schedule' });
    expect(schema.properties.recurring_schedule.minItems).to.equal(1);
    expect(schema.properties.specific_date.type).to.equal('array');
    expect(schema.properties.specific_date.items).to.deep.equal({ '$ref':'/specific_date' });
    expect(schema.properties.specific_date.minItems).to.equal(1);
    expect(schema.required).to.deep.equal(['timezone']);
    expect(schema.anyOf).to.deep.equal([ {'required':["recurring_schedule"]}, {'required':["specific_date"]} ]);
  });
  
  it('should validate the getScalingRuleSchema successfully',function(){
    var schema = schemaValidatorPrivate.__get__('getScalingRuleSchema')(settings);
    var validOperator = schemaValidatorPrivate.__get__('getValidOperators')();
    var adjustmentPattern = schemaValidatorPrivate.__get__('getAdjustmentPattern')();
    expect(schema.id).to.equal('/scaling_rules');
    expect(schema.properties.metric_type).to.deep.equal({ 'type':'string','pattern': '^[a-zA-Z0-9_]+$' });
    expect(schema.properties.breach_duration_secs).to.deep.equal({ 'type':'integer','minimum': 30,'maximum': 3600 });
    expect(schema.properties.threshold).to.deep.equal({ 'type':'integer'});
    expect(schema.properties.operator).to.deep.equal({ 'type':'string','enum':validOperator });
    expect(schema.properties.cool_down_secs).to.deep.equal({ 'type':'integer','minimum': 30,'maximum': 3600 });
    expect(schema.properties.adjustment).to.deep.equal({ 'type':'string','pattern':adjustmentPattern });
    expect(schema.required).to.deep.equal(['metric_type','threshold','operator','adjustment']);
  });
  
  it('should validate the getPolicySchema successfully',function(){
    var schema = schemaValidatorPrivate.__get__('getPolicySchema')(); 
    expect(schema.id).to.equal('/policySchema');
    expect(schema.properties.instance_min_count).to.deep.equal( { 'type':'integer','minimum':1});
    expect(schema.properties.instance_min_count).to.deep.equal( { 'type':'integer','minimum':1 });
    expect(schema.properties.scaling_rules.type).to.equal('array');
    expect(schema.properties.scaling_rules.items).to.deep.equal({ '$ref': '/scaling_rules' });
    expect(schema.properties.schedules).to.deep.equal({ '$ref':'/schedules' });
    expect(schema.required).to.deep.equal(['instance_min_count','instance_max_count']);
    expect(schema.anyOf).to.deep.equal([{'required':['scaling_rules']},{'required':['schedules']}]);
  });
  
  it('should validate the getValidOperators successfully',function(){
    var validOperators = schemaValidatorPrivate.__get__('getValidOperators')();
    expect(validOperators).to.not.be.null;
    expect(validOperators).to.have.members(['<','>','<=','>=']);
  });
  
    it('should validate the getAdjustmentPattern successfully',function(){
    var adjustmentPattern = schemaValidatorPrivate.__get__('getAdjustmentPattern')();
    expect(adjustmentPattern).to.not.be.null;
    expect(adjustmentPattern).to.equal('^[-+][1-9]+[0-9]*[%]?$');
  });
    
    it('should validate the getDaysInWeeksInISOFormat successfully',function(){
    var daysInWeekInISO = schemaValidatorPrivate.__get__('getDaysInWeeksInISOFormat')();
    expect(daysInWeekInISO).to.not.be.null;
    expect(daysInWeekInISO).to.have.members([1,2,3,4,5,6,7]);
  });
    
    it('should validate the getDaysInMonthInISOFormat successfully',function(){
      var daysInMonthInISO = schemaValidatorPrivate.__get__('getDaysInMonthInISOFormat')(); 
      expect(daysInMonthInISO).to.not.be.null;
      expect(daysInMonthInISO).to.have.members([1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,
                                              18,19,20,21,22,23,24,25,26,27,28,29,30,31]);
    });  
});

describe('Validating policy JSON schema against sample policy',function(){

  before(function(){
    total_schema = schemaValidatorPrivate.__get__('getPolicySchema')();
    specific_date_schema = schemaValidatorPrivate.__get__('getSpecificDateSchema')();
    recurring_schedule_schema = schemaValidatorPrivate.__get__('getRecurringSchema')();
    schedules_schema = schemaValidatorPrivate.__get__('getScheduleSchema')();
    scaling_rules_schema = schemaValidatorPrivate.__get__('getScalingRuleSchema')(settings);

    // Custom formats recreated through regular expression for testing purpose
    validator.addFormat('timeZoneFormat', /^[a-zA-Z0-9/]*$/i);
    validator.addFormat('dateFormat',/^2[0-9]{3}-(0[1-9]|1[0-2])-(0[1-9]|[1-2][0-9]|3[0-1])$/i);
    validator.addFormat('timeFormat',/^(2[0-3]|1[0-9]|0[0-9]):([0-5][0-9])$/i);
    validator.addFormat('dateTimeFormat',/^2[0-9]{3}-(0[1-9]|1[0-2])-(0[1-9]|[1-2][0-9]|3[0-1])T(2[0-3]|1[0-9]|0[0-9]):([0-5][0-9])$/i);

    validator.addSchema(specific_date_schema, '/specific_date');
    validator.addSchema(recurring_schedule_schema,'/recurring_schedule');
    validator.addSchema(schedules_schema,'/schedules');
    validator.addSchema(scaling_rules_schema,'/scaling_rules');
  });

  beforeEach(function(){
    fakePolicy = JSON.parse(fs.readFileSync(__dirname+'/../fakePolicy.json', 'utf8'));
  });
  it('Should validate the total schema',function() {
    expect(validator.validate(total_schema,fakePolicy)).to.be.true;
    delete fakePolicy.instance_max_count;
    expect(validator.validate(total_schema,fakePolicy)).to.be.false;
  });

  it('should validate schedules',function() {
    expect(validator.validate(schedules_schema,fakePolicy.schedules)).to.be.true;
    delete fakePolicy.schedules.timezone;
    expect(validator.validate(schedules_schema,fakePolicy.schedules)).to.be.false;
  });

  it('should validate scaling_rules',function() {
    expect(validator.validate(scaling_rules_schema,fakePolicy.scaling_rules[0])).to.be.true;
    delete fakePolicy.scaling_rules[0].cool_down_secs;
    expect(validator.validate(scaling_rules_schema,fakePolicy.scaling_rules[0])).to.be.true;
    delete fakePolicy.scaling_rules[0].threshold;
    expect(validator.validate(scaling_rules_schema,fakePolicy.scaling_rules[0])).to.be.false;
  });

  it('should validate empty scaling_rules',function(){
    expect(validator.validate(total_schema,fakePolicy)).to.be.true;
    fakePolicy.scaling_rules = [];
    expect(validator.validate(total_schema,fakePolicy)).to.be.false;
    expect(validator.errors[0].message).to.equal('should NOT have less than 1 items');
  });

  it('should validate scaling_rules with custom metrics',function(){
    expect(validator.validate(scaling_rules_schema,fakePolicy.scaling_rules[0])).to.be.true;
    fakePolicy.scaling_rules[0].metric_type = 'queueLength'
    expect(validator.validate(scaling_rules_schema,fakePolicy.scaling_rules[0])).to.be.true;
  });
  it('should validate specific_date schedules',function() {
    expect(validator.validate(specific_date_schema,fakePolicy.schedules.specific_date[0])).to.be.true;
    fakePolicy.schedules.specific_date[0].start_date_time = '2015-15-04T20:00';
    expect(validator.validate(specific_date_schema,fakePolicy.schedules.specific_date[0])).to.be.false;
  });

  it('should validate recurring_schedule',function() {
    expect(validator.validate(recurring_schedule_schema,fakePolicy.schedules.recurring_schedule[1])).to.be.true;
    fakePolicy.schedules.recurring_schedule[1].days_of_month = [];
    expect(validator.validate(recurring_schedule_schema,fakePolicy.schedules.recurring_schedule[1])).to.be.false;
    fakePolicy.schedules.recurring_schedule[1].days_of_month = [1,2,3];
    expect(validator.validate(recurring_schedule_schema,fakePolicy.schedules.recurring_schedule[0])).to.be.true;
    fakePolicy.schedules.recurring_schedule[0].days_of_week = [];
    expect(validator.validate(recurring_schedule_schema,fakePolicy.schedules.recurring_schedule[0])).to.be.false;
    fakePolicy.schedules.recurring_schedule[0].days_of_week = [1,2,3];
    expect(validator.validate(recurring_schedule_schema,fakePolicy.schedules.recurring_schedule[0])).to.be.true;
    fakePolicy.schedules.recurring_schedule[0].start_time = '24:12';
    expect(validator.validate(recurring_schedule_schema,fakePolicy.schedules.recurring_schedule[0])).to.be.false;
  });

  it('should fail to validate scaling_rules for non-integer threshold values',function() {
    expect(validator.validate(total_schema,fakePolicy)).to.be.true;
    fakePolicy.scaling_rules[0].threshold = 30.2;
    expect(validator.validate(total_schema,fakePolicy)).to.be.false;
    expect(validator.errors[0].message).to.equal('should be integer');
  });

  it('should fail to validate scaling_rules for non-integer breach_duration_secs values',function() {
    expect(validator.validate(total_schema,fakePolicy)).to.be.true;
    fakePolicy.scaling_rules[0].breach_duration_secs = 60.2;
    expect(validator.validate(total_schema,fakePolicy)).to.be.false;
    expect(validator.errors[0].message).to.equal('should be integer');
  });

  it('should fail to validate scaling_rules for non-integer cool_down_secs values',function() {
    expect(validator.validate(total_schema,fakePolicy)).to.be.true;
    fakePolicy.scaling_rules[0].cool_down_secs = 60.2;
    expect(validator.validate(total_schema,fakePolicy)).to.be.false;
    expect(validator.errors[0].message).to.equal('should be integer');
  });

});
