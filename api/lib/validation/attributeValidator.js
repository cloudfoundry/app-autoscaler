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
  var result = true;
  if(moment(startTime,'HH:mm').isAfter(moment(endTime,'HH:mm'))) {
    result = false;
  }
  return result;
}

var compareDateTimes = function(startDateTime,endDateTime) {
  var result = true;
  if(moment(startDateTime,'YYYY-MM-DDTHH:mm')
    .isAfter(moment(endDateTime,'YYYY-MM-DDTHH:mm'))) {
    result = false;
  }
  return result;
}

var validateDaysOfWeekErrors = function(inputRecurringSchedules) {
  var daysOfWeekErrors = [];
  var errorCount = 0;
  for(let j = 0; j < inputRecurringSchedules.length; j++) {
    for(let i = j + 1; i < inputRecurringSchedules.length; i++) {
      if(inputRecurringSchedules[i].hasOwnProperty('days_of_week') && 
        inputRecurringSchedules[j].hasOwnProperty('days_of_week')) {
        if(intersect(inputRecurringSchedules[i].days_of_week,
         inputRecurringSchedules[j].days_of_week).length > 0) {
          if(inputRecurringSchedules[i].start_time <=
          inputRecurringSchedules[j].end_time && 
          inputRecurringSchedules[i].end_time >=
          inputRecurringSchedules[j].start_time) {
            let error = createErrorResponse('recurring_schedule.start_time',
            'recurring_schedule.start_time and recurring_schedule.end_time ' + 
            'ranges are overlapping',inputRecurringSchedules[j],
            'Days_of_week based time range of recurring_schedule[' + j + ']' + 
            ' is overlapped with time range of recurring_schedule[' + i + ']');
            daysOfWeekErrors[errorCount++] = error;
          }
        }
      }
    }
  }
  return daysOfWeekErrors;
}


var validateDaysOfMonthErrors = function(inputRecurringSchedules) {
  var daysOfMonthErrors = [];
  var errorCount = 0;
  for(let j = 0; j < inputRecurringSchedules.length; j++) {
    for(let i = j + 1;i < inputRecurringSchedules.length; i++) {
      if(inputRecurringSchedules[i].hasOwnProperty('days_of_month') && 
            inputRecurringSchedules[j].hasOwnProperty('days_of_month')) {
        if(intersect(inputRecurringSchedules[i].days_of_month,
          inputRecurringSchedules[j].days_of_month).length > 0) {
          if(inputRecurringSchedules[i].start_time <= 
          inputRecurringSchedules[j].end_time && 
          inputRecurringSchedules[i].end_time >= 
          inputRecurringSchedules[j].start_time) {
            let error = createErrorResponse('recurring_schedule.start_time',
            'recurring_schedule.start_time and recurring_schedule.end_time ranges ' + 
            'are overlapping',inputRecurringSchedules[j],'Days_of_month based time range ' + 
            'of recurring_schedule[' + j + '] is overlapped with time range ' + 
            'of recurring_schedule[' + i + ']');
            daysOfMonthErrors[errorCount++] = error;
          }
        }
      }
    }
  }
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
    if(!compareTimes(inputRecurringSchedules[i].start_time,
      inputRecurringSchedules[i].end_time)) {
      let error = createErrorResponse('recurring_schedule.start_time',
      'recurring_schedule.start_time and recurring_schedule.end_time values are ' + 
      'not compatible',inputRecurringSchedules[i],'start_time ' + 
      inputRecurringSchedules[i].start_time + ' is after end_time ' + 
      inputRecurringSchedules[i].end_time + ' in recurring_schedule :[' + i + ']');
      attributeValueErrors[errorCount++] = error;
    }
  }
  return attributeValueErrors;
}

var validateRecurringRuleErrors = function(inputRecurringSchedules) {
  var errors = [];
  var attributeValueErrors = [];
  var daysOfMonthErrors = [];
  var daysOfWeekErrors = [];
  if(inputRecurringSchedules.length > 0) {
    attributeValueErrors = validateRecurringScheduleAttributeValue(inputRecurringSchedules);
    daysOfMonthErrors = validateDaysOfMonthErrors(inputRecurringSchedules);
    daysOfWeekErrors = validateDaysOfWeekErrors(inputRecurringSchedules);
    errors = errors.concat(attributeValueErrors,daysOfMonthErrors,daysOfWeekErrors);
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
    if(!compareDateTimes(inputSpecificDates[i].start_date_time,
      inputSpecificDates[i].end_date_time)) {
      let error = createErrorResponse('specific_date.start_date_time',
      'specific_date.start_date_time and specific_date.end_date_time ' +
      'values are not compatible',inputSpecificDates[i],'start_date_time ' + 
      inputSpecificDates[i].start_date_time + ' is after end_date_time ' + 
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
  if(inputSpecificDates.length > 0) {
    attributeValueErrors = validateSpecificDateAttributeValue(inputSpecificDates);
    dateRangeErrors = validateSpecificDateRange(inputSpecificDates);
    errors = attributeValueErrors.concat(dateRangeErrors);
  }
  return errors;
}

var validatePolicyJSONValues = function(policyJson) {
  var errorCount = 0;
  var errors = [];
  if(policyJson.schedules) {
    var inputRecurringSchedules = policyJson.schedules.recurring_schedule;
    var inputSpecificDates = policyJson.schedules.specific_date;
    if(policyJson.instance_min_count >= policyJson.instance_max_count) {
      errors[errorCount++] = createErrorResponse('instance_min_count','instance_min_count and ' + 
      'instance_max_count values are not compatible', policyJson,'instance_min_count ' + 
      policyJson.instance_min_count + ' is higher or equal to instance_max_count ' + 
      policyJson.instance_max_count + ' in policy_json');
    }
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
  
  return errors;
}



exports.validatePolicy = function(inputPolicy,callback) {
  if(callback) {
    var result = validatePolicyJSONValues(inputPolicy); 
    callback(result);
  }
  else{
    logger.error('No callback function specified!');
    return;
  }
}

