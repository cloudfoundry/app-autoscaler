var models = require('../models')();
var express = require('express');
var router = express.Router();
var logger = require('../log/logger');
var HttpStatus = require('http-status-codes');
var validationMiddleWare = require('../validation/validationMiddleware');

router.put('/:app_id/policy',validationMiddleWare,function(req, res) {
  logger.info('Policy creation request received for app id : ' + req.params.app_id);
  models.policy_json.create({
    policy_json: req.body,
    app_id: req.params.app_id
  }).then(function(result) {
    var successResponse = {
      'success': true,
      'error': null,
      'result': result
    };
    res.set('Location', '/v1/apps/' + req.params.app_id + '/policy');
    res.status(HttpStatus.CREATED).json(successResponse);
  }).catch(function(error) {
    logger.error ('Policy creation failed for app id : ' + req.params.app_id, error);
    var errorResponse = {
      'success': false,
      'error': error,
      'result': null
    };
    res.status(HttpStatus.BAD_REQUEST).json(errorResponse)
  });
});

module.exports = router;
