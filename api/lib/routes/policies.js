var models = require('../models');
var express = require('express');
var router = express.Router();
var logger = require('../log/logger');

router.put('/:app_id/policy',function(req, res) {
  logger.info('Policy creation request received for app id : ' + req.params.app_id);
  models.policy_json.create({
    policy_json: req.body,
    app_id: req.params.app_id
  }).then(function(result) {
    var result = {
      'success': true,
      'error': null,
      'result': result
    };
    res.send(result);
  }).catch(function(error) {
    logger.error ('Policy creation failed for app id : ' + req.params.app_id, error);
    var error = {
      'success': false,
      'error': error,
      'result': null
    };
    res.send(error)
  });
});

module.exports = router;
