'use-strict'
var moment = require('moment-timezone');
var JSONValidator = require('jsonschema').Validator;
var validator = new JSONValidator();
var logger = require('../log/logger');
var _ = require('underscore');

JSONValidator.prototype.customFormats.dateTimeFormat = function(inputDate) {
  return moment(inputDate, ['YYYY-MM-DDTHH:mm'], true).isValid();
};

JSONValidator.prototype.customFormats.dateFormat = function(inputDate) {
  return moment(inputDate, ['YYYY-MM-DD'], true).isValid();
};
JSONValidator.prototype.customFormats.timeFormat = function(inputTime) {
  return moment(inputTime, 'HH:mm',true).isValid();
};

//  Getting the list of the Valid Time zone Using moment from latest IANA  data 

var getTimeZones = function() {
  var timezoneList = moment.tz.names();
  return timezoneList;
}

JSONValidator.prototype.customFormats.timeZoneFormat = function(inputTimeZone) {
  if (inputTimeZone === "undefined" || inputTimeZone === null) {
    return false;
  }
  var trimmedTimeZone = inputTimeZone.replace(/\s+/g, '');
  var timeZoneList = getTimeZones();
  if(_.contains(timeZoneList,trimmedTimeZone)) {
    return true;
  }
  return false;
};



var getValidOperators = function() {
  var validOperators = ['<','>','<=','>='];
  return validOperators;
};

var getAdjustmentPattern = function() {
  var adjustmentPattern = '^[-+][1-9]+[0-9]*[%]?$';
  return adjustmentPattern;
};

var getDaysInWeeksInISOFormat = function() {
  var weekEnum = [1,2,3,4,5,6,7];
  return weekEnum;
};

var getDaysInMonthInISOFormat = function() {
  var monthEnum = [1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,
                 18,19,20,21,22,23,24,25,26,27,28,29,30,31];
  return monthEnum;
};

var getPolicySchema = function() {
  var schema = {
    'type': 'object',
    'id':'/policySchema',
    'properties' :{
      'instance_min_count': { 'type':'integer','minimum':1 },
      'instance_max_count': { 'type':'integer','minimum':1 },
      'scaling_rules': {
        'type':'array',
        'minItems': 1,
        'items': { '$ref': '/scaling_rules' }
      },
      'schedules': { '$ref':'/schedules' }
    },
    'required' : ['instance_min_count','instance_max_count'],
    'anyOf':[ { 'required' : ['scaling_rules'] },{ 'required' : ['schedules'] } ]
  };
  return schema;
};


var getScalingRuleSchema = function(settings) {
  var validOperators = getValidOperators();
  var adjustmentPattern = getAdjustmentPattern();
  var schema = {
    'type': 'object',
    'id':'/scaling_rules',
    'properties' : {
      'metric_type':{ 'type':'string', 'pattern':'^[a-zA-Z0-9_]+$' },
      'breach_duration_secs':{ 'type':'integer','minimum': settings.minBreachDurationSecs,'maximum': 3600 },
      'threshold':{ 'type':'integer'},
      'operator':{ 'type':'string','enum': validOperators },
      'cool_down_secs':{ 'type':'integer','minimum': settings.minCoolDownSecs,'maximum': 3600 },
      'adjustment':{ 'type':'string','pattern': adjustmentPattern }
    },
    'required' : ['metric_type','threshold','operator','adjustment']
  };  
  return schema;
};


var getScheduleSchema = function() {
  var schema = {
    'type': 'object',
    'id':'/schedules',
    'properties' : {
      'timezone':{ 'type':'string','format': 'timeZoneFormat' },
      'recurring_schedule': {
        'type':'array',
        'items': { '$ref': '/recurring_schedule' },
        'minItems': 1
      },
      'specific_date': {
        'type':'array', 
        'items': { '$ref':'/specific_date' },
        'minItems': 1
      }
    },
    'required' : ['timezone'],
    'anyOf' : [ { 'required' : ['recurring_schedule'] }, { 'required' : ['specific_date'] }]
  };  
  return schema;
};

var getRecurringSchema = function() {
  var weekEnum = getDaysInWeeksInISOFormat();
  var monthEnum = getDaysInMonthInISOFormat();
  var schema = {
    'type': 'object',
    'id':'/recurring_schedule',
    'properties' : {
      'start_date': {
        'anyOf': [{ 'type':'string', 'format':'dateFormat' },{ 'type':'string', 'enum':[''] } ]
      },
      'start_time':{ 'type':'string','format':'timeFormat' },
      'end_date': {
        'anyOf': [ { 'type':'string', 'format':'dateFormat' },{ 'type':'string', 'enum':[''] }]
      },
      'end_time':{ 'type':'string','format':'timeFormat' },
      'instance_min_count':{ 'type':'integer','minimum':1 },
      'instance_max_count':{ 'type':'integer','minimum':1 },
      'initial_min_instance_count':{ 'type':'integer','minimum':1 },
      'days_of_week':{ 'type':'array','uniqueItems': true, 'minItems': 1,
        'items':{ 'type':'number','enum':weekEnum } },
      'days_of_month':{ 'type':'array','uniqueItems': true, 'minItems': 1,
        'items':{ 'type':'number','enum':monthEnum } }
    },
    'required' : ['start_time','end_time','instance_min_count','instance_max_count'],
    'oneOf' : [ { 'required' : ['days_of_week'] }, { 'required' : ['days_of_month'] }]
  };
  return schema;
};

var getSpecificDateSchema = function() {
  var schema = {
    'type': 'object',
    'id':'/specific_date',
    'properties' : {
      'start_date_time':{ 'type':'string','format':'dateTimeFormat' },
      'end_date_time':{ 'type':'string','format':'dateTimeFormat' },
      'instance_min_count':{ 'type':'integer','minimum':1 },
      'instance_max_count':{ 'type':'integer' ,'minimum':1 },
      'initial_min_instance_count':{ 'type':'integer','minimum':1 }
    },
    'required' : ['start_date_time','end_date_time','instance_min_count','instance_max_count']
  };
  return schema;
}

var initSchema = function(settings) {
  validator.addSchema(getSpecificDateSchema(), '/specific_date');
  validator.addSchema(getRecurringSchema(),'/recurring_schedule');
  validator.addSchema(getScheduleSchema(),'/schedules');
  validator.addSchema(getScalingRuleSchema(settings),'/scaling_rules');
  return getPolicySchema();
}

exports.validatePolicy = function validatePolicy(inputJson, settings ,callback) {
  if(callback) {
    var errors = validator.validate(inputJson, initSchema(settings)).errors;
    callback(errors);
  }
  else{
    logger.error('No callback function specified!', {});
    return;
  }
}

 
