var request = require('request') ;
var logger = require('../log/logger');
var HttpStatus = require('http-status-codes');

exports.createOrUpdateSchedule = function createOrUpdateSchedule(req,callback) {
  var schedulerURI = process.env.SCHEDULER_URI;
  if(!req.body.schedules) {
    logger.info('Policy does not have schedule info ',{ 'app id':req.params.app_id });
    callback(null,{ 'statusCode':HttpStatus.OK });
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
        logger.error('Error occurs during schedule creation/update ',
        { 'app id': req.params.app_id,'error':error });
        callback(error,{ 'statusCode':HttpStatus.INTERNAL_SERVER_ERROR });
      }
      else if(response.statusCode === HttpStatus.NO_CONTENT 
          || response.statusCode === HttpStatus.OK) { 
        logger.info('Schedules created/updated successfully',{ 'app id': req.params.app_id });
        callback(null,{ 'statusCode':HttpStatus.OK });
      }
      else { 
        /* Creating the error object for Validation Error in scheduler with status code 400 
        to call the main callback in the waterfall immediately after getting this error. */
        var internalError = { 'message':'Failed to create schedules due to an internal' + 
            ' error in scheduler','details':response.body };
        logger.error('Error occurs in scheduler module during creation/update ',
        { 'app id': req.params.app_id,'error':internalError });
        callback(internalError,{ 'statusCode':response.statusCode });
      }
    });
  }

};


