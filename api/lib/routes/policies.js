'use strict';
module.exports = function(settings, models){
  
  var express = require('express');
  var router = express.Router();
  var logger = require('../log/logger');
  var HttpStatus = require('http-status-codes');
  var validationMiddleWare = require('../validation/validationMiddleware')(settings);
  var policyHelper = require('./policyHelper')(models);
  var schedulerUtil = require('../utils/schedulerUtils')(settings.scheduler, settings.httpClientTimeout);
  var async = require('async');
  var uuidV4 = require('uuid/v4');

  router.put('/:app_id/policy',validationMiddleWare.Validate,function(req, res) {
    logger.info('Policy creation request received',{ 'app id': req.params.app_id });
    req.query.policy_guid = uuidV4();
    logger.info('Policy guid ' + req.query.policy_guid);
    async.waterfall([async.apply(schedulerUtil.createOrUpdateSchedule, req),
      async.apply(policyHelper.createOrUpdatePolicy, req)],
      function(error, result) {
        var responseBody = {};
        var statusCode = HttpStatus.OK;
        if(error) {
          statusCode = error.statusCode;
          responseBody = {
            'error': error.message,
          };
        }
        else {
          statusCode = result.statusCode;
          if(result.statusCode === HttpStatus.CREATED) {
            res.set('Location', '/v1/apps/' + req.params.app_id + '/policy');
          }
          responseBody = result.response;
        }
        res.status(statusCode).json(responseBody);
      });
  });

  router.delete('/:app_id/policy',function(req,res) {
    logger.info('Policy deletion request received for application', { 'app id': req.params.app_id });
    async.waterfall([async.apply(policyHelper.deletePolicy, req),
                     async.apply(schedulerUtil.deleteSchedules, req)],
    function(error, result) {
      var responseBody = {};
      var status = HttpStatus.OK;
      if(error) {
        status = error.statusCode;
        responseBody = {
          'error': error.message,
        };
      } 
      else {
        status = HttpStatus.OK;
      }
      res.status(status).json(responseBody);
    });
  });

  router.get('/:app_id/policy',function(req,res) {
    logger.info('Request for policy details received',{ 'app id': req.params.app_id });
    models.policy_json.findByPk(req.params.app_id).then (function(policyExists) {
      if(policyExists) {
        logger.info('Policy details retrieved ', { 'app id': req.params.app_id });
        res.status(HttpStatus.OK).json(policyExists.policy_json);
      } 
      else{
        logger.info('No policy found',{ 'app id': req.params.app_id });
        res.status(HttpStatus.NOT_FOUND).json({});
      }
    }).catch(function(error) {
      logger.error ('Failed to retrieve policy details',
          { 'app id': req.params.app_id,'error':error });
      var responseBody = {
            'error': error.message,
      };
      res.status(HttpStatus.INTERNAL_SERVER_ERROR).json(responseBody);
    });
  });

  return router;
}

