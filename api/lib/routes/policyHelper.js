'use strict';
module.exports = function(models){
  var logger = require('../log/logger');
  var HttpStatus = require('http-status-codes');
  var helper = {};
  helper.createOrUpdatePolicy = function(req, callback) {
  /*  Create policy will only be called in the async waterfall when we do not 
  get any error during the schedule creation/update. */
    models.policy_json.findOrCreate({ where:{ app_id: req.params.app_id },
      defaults: { app_id: req.params.app_id,
      policy_json: req.body, guid: req.query.policy_guid } })
      .spread(function(result, created) {
        if(created) {
          logger.info('No policy exists, creating policy..',{ 'app id': req.params.app_id });  
          callback(null, { 'statusCode':HttpStatus.CREATED,'response':result.policy_json });
        }
        else {
          logger.info('Updating the existing policy',{ 'app id': req.params.app_id });
          models.policy_json.update({
            app_id: req.params.app_id,
            policy_json: req.body,
            guid: req.query.policy_guid
          },{ where: { app_id: req.params.app_id } ,plain: true, returning:true }).then(function(result) {
            callback(null, { 'statusCode':HttpStatus.OK,'response':result[1].policy_json });
          }).catch(function(error) {
            logger.error ('Failed to update policy',
               { 'app id': req.params.app_id,'error':error });
            error.statusCode = HttpStatus.INTERNAL_SERVER_ERROR;
            callback(error);
          });
        }
      }).catch(function(error) {
        logger.error ('Failed to create policy', { 'app id': req.params.app_id,'error':error });
        error.statusCode = HttpStatus.INTERNAL_SERVER_ERROR;
        callback(error);
      });
  }

  helper.deletePolicy = function(req, callback) {
    var appId = req.params.app_id;
    models.policy_json.destroy({ where: { app_id: appId } }).then(function(result) {
      if(result > 0) {
        logger.info('Successfully deleted the policy for application',{ 'app id': appId });
        callback(null);
      }
      else {
        var error = {
          message: 'No policy bound with application',
          statusCode: HttpStatus.NOT_FOUND
        }
        logger.error('No policy bound with application',{ 'app id': appId, error: error });
        callback(error);
      }
    }).catch(function(error) {
      logger.error ('Internal Error while deleting policy', { 'app id': appId,'error':error });
      error.statusCode = HttpStatus.INTERNAL_SERVER_ERROR;
      callback(error);
    });
  };
  return helper;
}

