'use strict';
module.exports = function(models,credentialCache, cacheTTL) {

  var express = require('express');
  var router = express.Router();
  var logger = require('../log/logger');
  var HttpStatus = require('http-status-codes');
  var credHelper = require('./credentialHelper')(models);

  router.get('/:app_id/creds', function(req, resp) {
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

  router.delete('/:app_id/creds', function(req, res) {
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

  router.get('/:app_id/creds/validate', function(req, resp) {
    var appId = req.params.app_id;
    logger.info('Request for credential validation received', {
      'app_id': appId
    });
    try {
      var value = credentialCache.get(appId, true);
      resp.status(HttpStatus.OK).json({
        'isValid': value.isValid
      });
      return;
    } 
    catch (err) {
      logger.info('Credentials not found in cache', {
        'app_id': appId, 'err':err
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
          var isCached = credentialCache.set(appId, responseBody, cacheTTL);
          logger.info('Credential cached',{ 'app_id':appId, 'isCached':isCached });
        }
        resp.status(statusCode).json(responseBody)
      });
    }
  });
  return router;
}
