'use strict';
module.exports = function(settings){
  var request = require('request') ;
  var logger = require('../log/logger');
  var HttpStatus = require('http-status-codes');
  var schedulerUtilObj = {};
  schedulerUtilObj.createOrUpdateSchedule = function createOrUpdateSchedule(req,callback) {
    var schedulerURI = settings.schedulerUri;
    if(!req.body.schedules) {
      logger.info('Policy does not have schedule info ',{ 'app id':req.params.app_id });
      callback(null);
    }
    else{
      logger.info('Policy has schedules, creating/updating schedules ..',
      { 'app id':req.params.app_id });
      var options = { 
        url: schedulerURI + '/v2/schedules/' + req.params.app_id,
        method: 'PUT',
        body: req.body,
        json: true  
      };
      request(options, function(error, response, body) {
        if(error) {
          logger.error('Error occurred during schedule creation/update ',
                { 'app id': req.params.app_id,'error':error });
          error.statusCode = HttpStatus.INTERNAL_SERVER_ERROR;
          callback(error); 
        }
        else if(response.statusCode === HttpStatus.NO_CONTENT 
            || response.statusCode === HttpStatus.OK) { 
          logger.info('Schedules created/updated successfully',{ 'app id': req.params.app_id });
          callback(null);
        }
        else if(response.statusCode === HttpStatus.BAD_REQUEST) {
          /* Creating the error object for Validation Error in scheduler with status code 400 
          to call the main callback in the waterfall immediately after getting this error. */
          var validationError = { 'message':'Failed to create schedules due to validation' + 
              ' error in scheduler','details':response.body };
          logger.error('Error occurred during creation/update of schedules ',
              { 'app id': req.params.app_id,'error':validationError });
          validationError.statusCode = HttpStatus.BAD_REQUEST;
          callback(validationError);
        }
        // For any other error response received from Scheduler
        else {
          var internalError = { 'message':'Failed to create schedules due to an internal' + 
                  ' error in scheduler','details':response.body };
          logger.error('Error occurred in scheduler module during creation/update ',
              { 'app id': req.params.app_id,'error':internalError });
          internalError.statusCode = HttpStatus.INTERNAL_SERVER_ERROR
          callback(internalError);
        }
      });
    }

  };

  schedulerUtilObj.deleteSchedules = function deleteSchedules(req, callback) {
    logger.info('Deleting schedules for application',{ 'app id': req.params.app_id });
    var appId = req.params.app_id;
    var schedulerURI = settings.schedulerUri;
    var options = { 
      url: schedulerURI + '/v2/schedules/' + appId,
      method: 'DELETE'
    };
    
    request(options, function(error, response, body) {
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
              ' error in scheduler','details':response.body };
        logger.error('Error occurred in scheduler module during deletion ',
              { 'app id': appId,'error':internalError });
        internalError.statusCode = HttpStatus.INTERNAL_SERVER_ERROR;
        callback(internalError);
      }
    });

  };
  return schedulerUtilObj;
}

