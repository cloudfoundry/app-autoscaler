var models = require('../models')();
var express = require('express');
var router = express.Router();
var logger = require('../log/logger');
var HttpStatus = require('http-status-codes');
var validationMiddleWare = require('../validation/validationMiddleware');
var routeHelper = require('./routeHelper');
var schedulerUtil = require('../utils/schedulerUtils');
var async = require('async');

router.put('/:app_id',validationMiddleWare,function(req, res) {
  logger.info('Policy creation request received',{ 'app id': req.params.app_id });
  async.waterfall([async.apply(schedulerUtil.createOrUpdateSchedule, req),
    async.apply(routeHelper.createOrUpdatePolicy, req)],
    function(error, result) {
      var responseDecorator = { };
      if(error) {
        responseDecorator = {
          'success': false,
          'error': error,
          'result': null
        };
      }
      else {
        if(result.statusCode === HttpStatus.CREATED) {
          res.set('Location', '/v1/policies/' + req.params.app_id);
        }
        responseDecorator = {
          'success': true,
          'error': null,
          'result': result.response    
        }
      }
      res.status(result.statusCode).json(responseDecorator);
    });
});

router.delete('/:app_id',function(req,res) {
  logger.info('Policy deletion request received', { 'app id': req.params.app_id });
  models.policy_json.destroy({ where: { app_id: req.params.app_id } }).then(function(result) {
    if(result > 0) {
      logger.info('Successfully deleted the policy',{ 'app id': req.params.app_id });
      res.status(HttpStatus.OK).json({});
    }
    else{
      logger.info('No policy found to delete',{ 'app id': req.params.app_id });
      res.status(HttpStatus.NOT_FOUND).json({});
    }

  }).catch(function(error) {
    logger.error ('Failed to delete policy', { 'app id': req.params.app_id,'error':error });
    res.status(HttpStatus.INTERNAL_SERVER_ERROR).json(error);
  });
});

router.get('/:app_id',function(req,res) {
  logger.info('Request for policy details received',{ 'app id': req.params.app_id });
  models.policy_json.findById(req.params.app_id).then (function(policyExists) {
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
    res.status(HttpStatus.INTERNAL_SERVER_ERROR).json(error);
  });
});

module.exports = router;
