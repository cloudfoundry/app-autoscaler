'use strict';

var expect = require("chai").expect;
var fs = require('fs');
var rewire = require('rewire');
var logger = require('../../../lib/log/logger');
var attributeValidator = require('../../../lib/validation/attributeValidator');
var attributeValidatorPrivate = rewire('../../../lib/validation/attributeValidator');

describe('Validating Policy JSON properties',function(){
    
    var fakePolicy;
    beforeEach(function(){
      fakePolicy = JSON.parse(fs.readFileSync(__dirname+'/../fakePolicy.json', 'utf8'));
    });

    it('Should validate the policy JSON successfully',function(){
      attributeValidator.validatePolicy(fakePolicy,function(result){
        expect(result).to.be.empty;
      });
    });

    context('Validating instance_min/max_count properties',function(){
      beforeEach(function(){
        fakePolicy.instance_min_count = 10;
        fakePolicy.instance_max_count = 5;
      });

      it('Should fail to validate the policy as instance_min_count is greater than instance_max_count with schedule defined',function(){
          attributeValidator.validatePolicy(fakePolicy,function(result){
            expect(result[0]).to.have.property('stack').and.equal('instance_min_count 10 is higher or equal to instance_max_count 5 in policy_json');
            expect(result[0]).to.have.property('message').and.equal('instance_min_count and instance_max_count values are not compatible');
            expect(result[0]).to.have.property('property').and.equal('instance_min_count');
          });
        });

        it('Should fail to validate the policy as instance_min_count is greater than instance_max_count without schedule defined',function(){
          delete fakePolicy.schedules
          attributeValidator.validatePolicy(fakePolicy,function(result){
            expect(result[0]).to.have.property('stack').and.equal('instance_min_count 10 is higher or equal to instance_max_count 5 in policy_json');
            expect(result[0]).to.have.property('message').and.equal('instance_min_count and instance_max_count values are not compatible');
            expect(result[0]).to.have.property('property').and.equal('instance_min_count');
          });
        });

        it('Should fail to validate the policy as instance_min_count is greater than instance_max_count without scaling_rule defined',function(){
          delete fakePolicy.scaling_rules
          attributeValidator.validatePolicy(fakePolicy,function(result){
            expect(result[0]).to.have.property('stack').and.equal('instance_min_count 10 is higher or equal to instance_max_count 5 in policy_json');
            expect(result[0]).to.have.property('message').and.equal('instance_min_count and instance_max_count values are not compatible');
            expect(result[0]).to.have.property('property').and.equal('instance_min_count');
          });
        });

    });

    it('Should fail to validate the policy as threshold <=0 ',function(){
        fakePolicy.scaling_rules[0].threshold = 0
        attributeValidator.validatePolicy(fakePolicy,function(result){
          expect(result[0]).to.have.property('stack').and.equal('scaling_rules[0].threshold value should be greater than 0');
          expect(result[0]).to.have.property('message').and.equal('threshold value for metric_type memoryused should be greater than 0');
          expect(result[0]).to.have.property('property').and.equal('scaling_rules[0].threshold');
        });
    });
  
    it('Should fail to validate the policy as end_date is before start_date',function(){
      fakePolicy.schedules.specific_date[0].start_date_time = '2016-06-19T10:30';
      fakePolicy.schedules.specific_date[0].end_date_time = '2014-06-19T13:30';
      attributeValidator.validatePolicy(fakePolicy,function(result){
        expect(result[0].property).to.equal('specific_date.start_date_time');
        expect(result[0].message).to.equal('specific_date.start_date_time and specific_date.end_date_time values are not compatible');
        expect(result[0].stack).to.equal('start_date_time 2016-06-19T10:30 is same or after end_date_time 2014-06-19T13:30 in specific_date :[0]');  
      });
    });
    it('Should fail to validate the policy as instance_min_count is greater than instance_max_count in specific_date',function(){
      fakePolicy.schedules.specific_date[0].instance_min_count = 10;
      fakePolicy.schedules.specific_date[0].instance_max_count = 2;
      attributeValidator.validatePolicy(fakePolicy,function(result){
        expect(result[0].property).to.equal('specific_date.instance_min_count');
        expect(result[0].message).to.equal('specific_date.instance_min_count and specific_date.instance_max_count values are not compatible');
        expect(result[0].stack).to.equal('instance_min_count 10 is higher or equal to instance_max_count 2 in specific_date :[0]');  
      });
    });
    it('Should fail to validate the policy as initial_min_instance_count is greater than instance_max_count in specific_date',function(){
        fakePolicy.schedules.specific_date[0].initial_min_instance_count = 5;
        fakePolicy.schedules.specific_date[0].instance_max_count = 4;
        attributeValidator.validatePolicy(fakePolicy,function(result){
          expect(result[0].property).to.equal('specific_date.initial_min_instance_count');
          expect(result[0].message).to.equal('specific_date.initial_min_instance_count and specific_date.instance_max_count values are not compatible');
          expect(result[0].stack).to.equal('initial_min_instance_count 5 is higher than instance_max_count 4 in specific_date :[0]');  
        });
    });
    it('Should fail to validate the policy as initial_min_instance_count is less than instance_min_count in specific_date',function(){
        fakePolicy.schedules.specific_date[0].initial_min_instance_count = 1;
        fakePolicy.schedules.specific_date[0].instance_min_count = 2;
        attributeValidator.validatePolicy(fakePolicy,function(result){
          expect(result[0].property).to.equal('specific_date.initial_min_instance_count');
          expect(result[0].message).to.equal('specific_date.initial_min_instance_count and specific_date.instance_min_count values are not compatible');
          expect(result[0].stack).to.equal('initial_min_instance_count 1 is lower than instance_min_count 2 in specific_date :[0]');  
        });
    });
    it('Should fail to validate the Policy with overlapping time range in specific date',function(){
      fakePolicy.schedules.specific_date[1].start_date_time = '2015-06-04T10:53';
      fakePolicy.schedules.specific_date[1].end_date_time = '2015-06-21T23:45';
      attributeValidator.validatePolicy(fakePolicy,function(result){
        expect(result[0].property).to.equal('specific_date.start_date_time');
        expect(result[0].stack).to.equal('Date range of specific_date[1] is overlapped with date range of specific_date[0]');
        expect(result[0].message).to.equals('specific_date.start_date_time and specific_date.end_date_time ranges are overlapping');
      });
    });
    it('should fail to validate policy with overlapping time range in days of week in recurring schedule',function(){
      fakePolicy.schedules.recurring_schedule[2].days_of_week = [4,5,2];
      attributeValidator.validatePolicy(fakePolicy,function(result){
        expect(result[0].property).to.equal('recurring_schedule.start_time');
        expect(result[0].message).to.equal('recurring_schedule.start_time and recurring_schedule.end_time ranges are overlapping');
        expect(result[0].stack).to.equal('days_of_week based time range of recurring_schedule[0] is overlapped with time range of recurring_schedule[2]');
      });
    });
    it('should fail to validate policy with overlapping time range in days of month in recurring schedule',function(){
      fakePolicy.schedules.recurring_schedule[1].days_of_month = [12,20,24];
      attributeValidator.validatePolicy(fakePolicy,function(result){
        expect(result[0].property).to.equal('recurring_schedule.start_time');
        expect(result[0].message).to.equal('recurring_schedule.start_time and recurring_schedule.end_time ranges are overlapping');
        expect(result[0].stack).to.equal('days_of_month based time range of recurring_schedule[1] is overlapped with time range of recurring_schedule[3]');
      });
    });
    it('should validate the policy successfully if days of month in recurring schedule is overlapping but start_time and end_time in overlapped date are non-overlapping',function(){
      fakePolicy.schedules.recurring_schedule[1].days_of_week = [4,5,2];
      fakePolicy.schedules.recurring_schedule[1].start_time = '19:00'
      fakePolicy.schedules.recurring_schedule[1].end_time = '23:15';
      attributeValidator.validatePolicy(fakePolicy,function(result){
        expect(result).to.be.empty;
      });
    });
    it('should fail to validate the policy if start_time is after end_time in recurring schedule',function(){
      fakePolicy.schedules.recurring_schedule[1].start_time = '23:00'
      fakePolicy.schedules.recurring_schedule[1].end_time = '13:15';
      attributeValidator.validatePolicy(fakePolicy,function(result){
        expect(result[0].property).to.equal('recurring_schedule.start_time');
        expect(result[0].message).to.equal('recurring_schedule.start_time and recurring_schedule.end_time values are not compatible');
        expect(result[0].stack).to.equal('start_time 23:00 is same or after end_time 13:15 in recurring_schedule :[1]');
      });
    });
    it('should fail to validate the policy if start_date is after end_date in recurring schedule',function(){
        fakePolicy.schedules.recurring_schedule[1].start_date = '2016-06-12'
        fakePolicy.schedules.recurring_schedule[1].end_date = '2016-06-02';
        attributeValidator.validatePolicy(fakePolicy,function(result){
          expect(result[0].property).to.equal('recurring_schedule.start_date');
          expect(result[0].message).to.equal('recurring_schedule.start_date and recurring_schedule.end_date values are not compatible');
          expect(result[0].stack).to.equal('start_date 2016-06-12 is after end_date 2016-06-02 in recurring_schedule :[1]');
        });
    });
    it('Should fail to validate the policy as instance_min_count is greater than instance_max_count in recurring_schedule',function(){
      fakePolicy.schedules.recurring_schedule[0].instance_min_count = 10;
      fakePolicy.schedules.recurring_schedule[0].instance_max_count = 2;
      attributeValidator.validatePolicy(fakePolicy,function(result){
        expect(result[0].property).to.equal('recurring_schedule.instance_min_count');
        expect(result[0].message).to.equal('recurring_schedule.instance_min_count and recurring_schedule.instance_max_count values are not compatible');
        expect(result[0].stack).to.equal('instance_min_count 10 is higher or equal to instance_max_count 2 in recurring_schedule :[0]');
      });
    });
    it('Should fail to validate the policy as initial_min_instance_count is greater than instance_max_count in recurring_schedule',function(){
        fakePolicy.schedules.recurring_schedule[0].initial_min_instance_count = 5;
        fakePolicy.schedules.recurring_schedule[0].instance_max_count = 4;
        attributeValidator.validatePolicy(fakePolicy,function(result){
          expect(result[0].property).to.equal('recurring_schedule.initial_min_instance_count');
          expect(result[0].message).to.equal('recurring_schedule.initial_min_instance_count and recurring_schedule.instance_max_count values are not compatible');
          expect(result[0].stack).to.equal('initial_min_instance_count 5 is higher than instance_max_count 4 in recurring_schedule :[0]');  
        });
    });
    it('Should fail to validate the policy as initial_min_instance_count is less than instance_min_count in recurring_schedule',function(){
        fakePolicy.schedules.recurring_schedule[0].initial_min_instance_count = 1;
        fakePolicy.schedules.recurring_schedule[0].instance_min_count = 2;
        attributeValidator.validatePolicy(fakePolicy,function(result){
          expect(result[0].property).to.equal('recurring_schedule.initial_min_instance_count');
          expect(result[0].message).to.equal('recurring_schedule.initial_min_instance_count and recurring_schedule.instance_min_count values are not compatible');
          expect(result[0].stack).to.equal('initial_min_instance_count 1 is lower than instance_min_count 2 in recurring_schedule :[0]');  
        });
    });
    
   
  });
  
