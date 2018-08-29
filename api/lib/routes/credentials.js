'use strict';
module.exports = function(models, credentialCache, cacheTTL) {

  var express = require('express');
  var router = express.Router();
  var logger = require('../log/logger');
  var credHelper = require('./credentialHelper')(models, credentialCache, cacheTTL);

  router.post('/:app_id/credentials', function(req, resp) {
    var appId = req.params.app_id;
    logger.info('Request for credentials creation received', {
      'app_id': appId
    });
    credHelper.createOrUpdateCredentials(req, function(err, result) {
      var responseBody = {};
      var statusCode;
      if (err) {
        statusCode = err.statusCode;
        responseBody = {
          'error': err.message
        };
      }
      else {
        statusCode = result.statusCode;
        responseBody = {
          'username': result.username,
          'password': result.password
        }
      }
      resp.status(statusCode).json(responseBody)
    });
  });

  router.delete('/:app_id/credentials', function(req, res) {
    logger.info('Request for credentials deletion received', {
      'app_id': req.params.app_id
    });
    credHelper.deleteCredentials(req, function(err, result) {
      var responseBody = {};
      var statusCode;
      if (err) {
        statusCode = err.statusCode;
        responseBody = {
          'error': err.message
        };
      }
      else {
        statusCode = result.statusCode;
      }
      res.status(statusCode).json(responseBody);
    });
  });

  router.post('/:app_id/credentials/validate', function(req, resp) {
    var appId = req.params.app_id;
    logger.info('Request for credential validation received', {
      'app_id': appId
    });
    credHelper.validateCredentials(req, function(err, result) {
      var responseBody = {};
      var statusCode;
      if (err) {
        statusCode = err.statusCode;
        responseBody = {
          'error': err.message
        };
      }
      else {
        statusCode = result.statusCode;
        responseBody = {
          'isValid': result.isValid
        }
      }
      resp.status(statusCode).json(responseBody)
    });
  });
  return router;
}
