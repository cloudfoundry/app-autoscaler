'use strict'

var moment = require('moment');
require('moment-range');
var intersect = require('intersect');
var logger = require('../log/logger');

var createErrorResponse = function(property,message,instance,stack) {
  var errorObject = { property:property,
  message: message,instance:instance,stack:stack }; 
  return errorObject;
}
var compareTimes = function(startTime,endTime) {
  return moment(startTime,'HH:mm').isSameOrAfter(moment(endTime,'HH:mm'));
}
var compareDates = function(startDate,endDate) {
  return moment(startDate,'YYYY-MM-DD').isAfter(moment(endDate,'YYYY-MM-DD'));
}
var compareDateTimes = function(startDateTime,endDateTime) {
  return moment(startDateTime,'YYYY-MM-DDTHH:mm')
    .isSameOrAfter(moment(endDateTime,'YYYY-MM-DDTHH:mm'));
}
var normalizeStartDate = function(object) {
  if ( !object.hasOwnProperty("start_date") ) {
    return '0001-01-01'
  }  
  return object.start_date;
}
var normalizeEndDate = function(object) {
  if ( !object.hasOwnProperty("end_date") ) {
    return '9999-12-31'
  }  
  return object.end_date;
}

var validateOverlappingDaysInRecurringSchedules = function(inputRecurringSchedules,property) {
  var errors = [];
  var errorCount = 0;
  for(let j = 0; j < inputRecurringSchedules.length - 1; j++) {
    for(let i = j + 1; i < inputRecurringSchedules.length; i++) {
      if(inputRecurringSchedules[i].hasOwnProperty(property) && 
          inputRecurringSchedules[j].hasOwnProperty(property)) {
        if(intersect(inputRecurringSchedules[i][property],
            inputRecurringSchedules[j][property]).length > 0) {
  
            var normalizeStartDate_I = normalizeStartDate(inputRecurringSchedules[i]);
            var normalizeEndDate_I = normalizeEndDate(inputRecurringSchedules[i]);
            var normalizeStartDate_J = normalizeStartDate(inputRecurringSchedules[j]);
            var normalizeEndDate_J = normalizeEndDate(inputRecurringSchedules[j]);

            if((inputRecurringSchedules[i].start_time <=inputRecurringSchedules[j].end_time) && 
             (inputRecurringSchedules[i].end_time >=inputRecurringSchedules[j].start_time) &&
             (normalizeStartDate_I <= normalizeEndDate_J) && 
             (normalizeEndDate_I >= normalizeStartDate_J)) {
            let error = createErrorResponse('recurring_schedule.start_time',
                'recurring_schedule.start_time and recurring_schedule.end_time ' + 
                'ranges are overlapping',inputRecurringSchedules[j],
                property + ' based time range of recurring_schedule[' + j + ']' + 
                ' is overlapped with time range of recurring_schedule[' + i + ']');
            errors[errorCount++] = error;
          }
        }
      }
    }
  }

  return errors;
}

var validateDaysOfWeekErrors = function(inputRecurringSchedules) {
  var daysOfWeekErrors = validateOverlappingDaysInRecurringSchedules(inputRecurringSchedules,
  'days_of_week');
  return daysOfWeekErrors;
}


var validateDaysOfMonthErrors = function(inputRecurringSchedules) {
  var daysOfMonthErrors = validateOverlappingDaysInRecurringSchedules(inputRecurringSchedules,
  'days_of_month')
  return daysOfMonthErrors;
}



var validateRecurringScheduleAttributeValue = function(inputRecurringSchedules) {
  var attributeValueErrors = [];
  var errorCount = 0;
  for(let i = 0; i < inputRecurringSchedules.length; i++) {
    if(inputRecurringSchedules[i].instance_min_count >= 
      inputRecurringSchedules[i].instance_max_count) {
      let error = createErrorResponse('recurring_schedule.instance_min_count',
          'recurring_schedule.instance_min_count and recurring_schedule.instance_max_count ' + 
          'values are not compatible',inputRecurringSchedules[i],'instance_min_count ' + 
          inputRecurringSchedules[i].instance_min_count + ' is higher or equal to ' + 
          'instance_max_count ' + inputRecurringSchedules[i].instance_max_count +
          ' in recurring_schedule :[' + i + ']');
      attributeValueErrors[errorCount++] = error;
    }
    if(compareTimes(inputRecurringSchedules[i].start_time,
        inputRecurringSchedules[i].end_time)) {
      let error = createErrorResponse('recurring_schedule.start_time',
          'recurring_schedule.start_time and recurring_schedule.end_time values are ' + 
          'not compatible',inputRecurringSchedules[i],'start_time ' + 
          inputRecurringSchedules[i].start_time + ' is same or after end_time ' + 
          inputRecurringSchedules[i].end_time + ' in recurring_schedule :[' + i + ']');
      attributeValueErrors[errorCount++] = error;
    }
    if(inputRecurringSchedules[i].start_date && inputRecurringSchedules[i].end_date) {
      if(compareDates(inputRecurringSchedules[i].start_date,
          inputRecurringSchedules[i].end_date)) {
        let error = createErrorResponse('recurring_schedule.start_date',
            'recurring_schedule.start_date and recurring_schedule.end_date values are ' + 
            'not compatible',inputRecurringSchedules[i],'start_date ' + 
            inputRecurringSchedules[i].start_date + ' is after end_date ' + 
            inputRecurringSchedules[i].end_date + ' in recurring_schedule :[' + i + ']');
        attributeValueErrors[errorCount++] = error;
      }

    }
  }
  return attributeValueErrors;
}

var validateInitialInstanceCount = function(inputSchedules,type) {
  var minInstCountError = [];
  var errorCount = 0;
  for(let i = 0; i < inputSchedules.length; i++) {
    if(inputSchedules[i].initial_min_instance_count) {
      if(inputSchedules[i].initial_min_instance_count <  
          inputSchedules[i].instance_min_count) {
        let error = createErrorResponse(type + '.initial_min_instance_count',
            type + '.initial_min_instance_count and ' + type + '.instance_min_count ' + 
            'values are not compatible',inputSchedules[i],'initial_min_instance_count ' + 
            inputSchedules[i].initial_min_instance_count + ' is lower than ' + 
            'instance_min_count ' + inputSchedules[i].instance_min_count +
            ' in ' + type + ' :[' + i + ']');
        minInstCountError[errorCount++] = error;
      }
      if(inputSchedules[i].initial_min_instance_count > 
      inputSchedules[i].instance_max_count) {
        let error = createErrorResponse(type + '.initial_min_instance_count',
            type + '.initial_min_instance_count and ' + type + '.instance_max_count ' + 
            'values are not compatible',inputSchedules[i],'initial_min_instance_count ' + 
            inputSchedules[i].initial_min_instance_count + ' is higher than ' + 
            'instance_max_count ' + inputSchedules[i].instance_max_count +
            ' in ' + type + ' :[' + i + ']');
        minInstCountError[errorCount++] = error;
      }
    }
  }
  return minInstCountError;
}
var validateRecurringRuleErrors = function(inputRecurringSchedules) {
  var errors = [];
  var attributeValueErrors = [];
  var daysOfMonthErrors = [];
  var daysOfWeekErrors = [];
  var initialInstanceCountErrors = [];
  if(inputRecurringSchedules.length > 0) {
    attributeValueErrors = validateRecurringScheduleAttributeValue(inputRecurringSchedules);
    daysOfMonthErrors = validateDaysOfMonthErrors(inputRecurringSchedules);
    daysOfWeekErrors = validateDaysOfWeekErrors(inputRecurringSchedules);
    initialInstanceCountErrors = validateInitialInstanceCount(inputRecurringSchedules,
                                 'recurring_schedule');
    errors = errors.concat(attributeValueErrors,daysOfMonthErrors,
             daysOfWeekErrors,initialInstanceCountErrors);
  }
  return errors;
}

var validateSpecificDateRange = function(inputSpecificDates) {
  var dateRangeErrors = [];
  var errorCount = 0;
  var dateRangeList = [];
  if(inputSpecificDates.length > 0) {
    for(let i = 0; i < inputSpecificDates.length; i++) {
      var start = moment(inputSpecificDates[i].start_date_time, 'YYYY-MM-DDTHH:mm');
      var end = moment(inputSpecificDates[i].end_date_time, 'YYYY-MM-DDTHH:mm');
      var range = moment.range(start, end);
      dateRangeList[i] = range;
    }
    for(let j = 0; j < dateRangeList.length; j++) {
      for(let i = j + 1; i < dateRangeList.length; i++) {
        if(dateRangeList[j].overlaps(dateRangeList[i])) {
          let error = createErrorResponse('specific_date.start_date_time',
              'specific_date.start_date_time and specific_date.end_date_time ' + 
              'ranges are overlapping', inputSpecificDates[i],
              'Date range of specific_date[' + i + '] is overlapped with date range of ' + 
              'specific_date[' + j + ']');
          dateRangeErrors[errorCount++] = error;
        }
      }
    }  
  }
  return dateRangeErrors;
}

var validateSpecificDateAttributeValue = function(inputSpecificDates) {
  var attributeValueErrors = [];
  var errorCount = 0; 
  for(let i = 0; i < inputSpecificDates.length; i++) {
    if(inputSpecificDates[i].instance_min_count >=
      inputSpecificDates[i].instance_max_count) {
      let error = createErrorResponse('specific_date.instance_min_count',
          'specific_date.instance_min_count and specific_date.instance_max_count' + 
          ' values are not compatible',inputSpecificDates[i],'instance_min_count ' + 
          inputSpecificDates[i].instance_min_count + ' is higher or equal to ' + 
          'instance_max_count ' + inputSpecificDates[i].instance_max_count + 
          ' in specific_date :[' + i + ']');
      attributeValueErrors[errorCount++] = error;
    }
    if(compareDateTimes(inputSpecificDates[i].start_date_time,
        inputSpecificDates[i].end_date_time)) {
      let error = createErrorResponse('specific_date.start_date_time',
          'specific_date.start_date_time and specific_date.end_date_time ' +
          'values are not compatible',inputSpecificDates[i],'start_date_time ' + 
          inputSpecificDates[i].start_date_time + ' is same or after end_date_time ' + 
          inputSpecificDates[i].end_date_time + ' in specific_date :[' + i + ']');
      attributeValueErrors[errorCount++] = error;
    }
  }
  return attributeValueErrors;
}
var validateSpecificDateErrors = function(inputSpecificDates) {
  var errors = [];
  var attributeValueErrors = [];
  var dateRangeErrors = [];
  var initialInstanceCountErrors = [];
  if(inputSpecificDates.length > 0) {
    attributeValueErrors = validateSpecificDateAttributeValue(inputSpecificDates);
    dateRangeErrors = validateSpecificDateRange(inputSpecificDates);
    initialInstanceCountErrors = validateInitialInstanceCount(inputSpecificDates,'specific_date');
    errors = attributeValueErrors.concat(dateRangeErrors,initialInstanceCountErrors);
  }
  return errors;
}

var validateScalingRuleThresholdValue = function(scalingRules){
  var errors = [];
  var errorCount = 0;
  for(let i=0; i < scalingRules.length; i++){
    switch(scalingRules[i]['metric_type']){
      case 'memoryused':
        if(scalingRules[i]['threshold'] <= 0){
          errors[errorCount++] = createErrorResponse('scaling_rules['+i+'].threshold',
            'threshold value for metric_type memoryused should be greater than 0',scalingRules[i],
            'scaling_rules['+i+'].threshold value should be greater than 0');
        }
        break;
      case 'memoryutil':
        if(scalingRules[i]['threshold'] <= 0 || scalingRules[i]['threshold'] > 100){
          errors[errorCount++] = createErrorResponse('scaling_rules['+i+'].threshold',
            'threshold value for metric_type memoryutil should be greater than 0 and less than equal to 100',scalingRules[i],
            'scaling_rules['+i+'].threshold value should be greater than 0 and less than equal to 100');
        }
        break;
      case 'responsetime':
        if(scalingRules[i]['threshold'] <= 0){
          errors[errorCount++] = createErrorResponse('scaling_rules['+i+'].threshold',
            'threshold value for metric_type responsetime should be greater than 0',scalingRules[i],
            'scaling_rules['+i+'].threshold value should be greater than 0');
        }
        break;
      case 'throughput':
        if(scalingRules[i]['threshold'] <= 0){
          errors[errorCount++] = createErrorResponse('scaling_rules['+i+'].threshold',
            'threshold value for metric_type throughput should be greater than 0',scalingRules[i],
            'scaling_rules['+i+'].threshold value should be greater than 0');
        }
        break;
      default:
        break;
    }
  }
  return errors;
}


var validatePolicyJSONValues = function(policyJson) {
  var errorCount = 0;
  var errors = [];

  if(policyJson.instance_min_count >= policyJson.instance_max_count) {
    errors[errorCount++] = createErrorResponse('instance_min_count','instance_min_count and ' + 
        'instance_max_count values are not compatible', policyJson,'instance_min_count ' + 
        policyJson.instance_min_count + ' is higher or equal to instance_max_count ' + 
        policyJson.instance_max_count + ' in policy_json');
  }
  
  if(policyJson.schedules) {
    var inputRecurringSchedules = policyJson.schedules.recurring_schedule;
    var inputSpecificDates = policyJson.schedules.specific_date;

    var specificDateErrors = [];
    var recurringScheduleErrors = [];
    if(inputSpecificDates) {
      specificDateErrors = validateSpecificDateErrors(inputSpecificDates);
    }
    if(inputRecurringSchedules) {
      recurringScheduleErrors = validateRecurringRuleErrors(inputRecurringSchedules); 
    }
    errors = errors.concat(specificDateErrors,recurringScheduleErrors);
  }
  if(policyJson.scaling_rules){
    var scalingRulesErrors = [];
    scalingRulesErrors = validateScalingRuleThresholdValue(policyJson.scaling_rules);
    errors = errors.concat(scalingRulesErrors);
  }
  return errors;
}



exports.validatePolicy = function(inputPolicy,callback) {
  if(callback) {
    var errors = validatePolicyJSONValues(inputPolicy); 
    callback(errors);
  }
  else{
    logger.error('No callback function specified!', {});
    return;
  }
}

