'use strict';
module.exports = function(schedulerSettings, httpClientTimeout) {
  var request = require('request') ;
  var logger = require('../log/logger');
  var HttpStatus = require('http-status-codes');
  var fs = require('fs');
  var schedulerUtilObj = {};

  var getOptions = function(appId, policy, policyGuid ){
    var options = { 
      json: true,
      timeout: httpClientTimeout
    };
    
    if(policy && policyGuid){
      options.body = policy;
      options.url = schedulerSettings.uri + '/v1/apps/' + appId + "/schedules" + "?guid=" + policyGuid;
    }else{
      options.url = schedulerSettings.uri + '/v1/apps/' + appId + "/schedules";
    }
    if(schedulerSettings.tls){
      var schedulerTLSOptions = {
        key: fs.readFileSync(schedulerSettings.tls.keyFile),
        cert: fs.readFileSync(schedulerSettings.tls.certFile),
        ca: fs.readFileSync(schedulerSettings.tls.caCertFile)
      };
      options.cert = schedulerTLSOptions.cert,
      options.key = schedulerTLSOptions.key,
      options.ca = schedulerTLSOptions.ca
    }
    return options;
  }

  
  schedulerUtilObj.createOrUpdateSchedule = function createOrUpdateSchedule(req,callback) {
    var appId = req.params.app_id;
    var policyGuid = req.query.policy_guid;
    var policy = req.body;

    var options = getOptions(appId, policy, policyGuid);
    request.put(options, function (error, response, body) {
      if (error) {
        logger.error('Error occurred during schedule creation/update ',
          { 'app id': appId, 'error': error.message });
        error.statusCode = HttpStatus.INTERNAL_SERVER_ERROR;
        callback(error);
      }
      else if (response.statusCode === HttpStatus.NO_CONTENT
        || response.statusCode === HttpStatus.OK) {
        logger.info('Schedules created/updated successfully', { 'app id': appId });
        callback(null);
      }
      else if (response.statusCode === HttpStatus.BAD_REQUEST) {
        /* Creating the error object for Validation Error in scheduler with status code 400 
        to call the main callback in the waterfall immediately after getting this error. */
        var validationError = {
          'message': 'Failed to create schedules due to validation' +
            ' error in scheduler, details:\n' + response.body
        };
        logger.error('Error occurred during creation/update of schedules ',
          { 'app id': appId, 'error': validationError });
        validationError.statusCode = HttpStatus.BAD_REQUEST;
        callback(validationError);
      }
      // For any other error response received from Scheduler
      else {
        var internalError = {
          'message': 'Failed to create schedules due to an internal' +
            ' error in scheduler, details:\n' + response.body
        };
        logger.error('Error occurred in scheduler module during creation/update ',
          { 'app id': appId, 'error': internalError });
        internalError.statusCode = HttpStatus.INTERNAL_SERVER_ERROR
        callback(internalError);
      }
    });
    

  };

  schedulerUtilObj.deleteSchedules = function deleteSchedules(req, callback) {
    var appId = req.params.app_id;
    logger.info('Deleting schedules for application',{ 'app id': appId });
    
    var options = getOptions(appId);
    request.delete(options, function(error, response, body) {
      if(error) {
        logger.error('Error occurred during schedule deletion ', { 'app id': appId,'error':error });
        error.statusCode = HttpStatus.INTERNAL_SERVER_ERROR;
        callback(error);
      }
      else if (response.statusCode === HttpStatus.NO_CONTENT || 
          response.statusCode === HttpStatus.OK) {
        logger.info('Schedules deleted successfully ',{ 'app id': appId });
        callback(null);
      }
      else if (response.statusCode === HttpStatus.NOT_FOUND) {
        logger.info('No schedules found for application',{ 'app id': appId });
        callback(null);
      }
      else {
        var internalError = { 'message':'Failed to delete schedules due to an internal' + 
              ' error in scheduler, details:\n' + response.body };
        logger.error('Error occurred in scheduler module during deletion ',
              { 'app id': appId,'error':internalError });
        internalError.statusCode = HttpStatus.INTERNAL_SERVER_ERROR;
        callback(internalError);
      }
    });

  };
  return schedulerUtilObj;
}

