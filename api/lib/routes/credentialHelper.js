'use strict';
module.exports = function(models) {
  var logger = require('../log/logger');
  var HttpStatus = require('http-status-codes');
  var uuidv4 = require('uuid/v4');
  var bcrypt = require('bcrypt-nodejs');
  var credhelper = {};

  function generateHash(input) {
    return bcrypt.hashSync(input, bcrypt.genSaltSync(8));
  }

  function validateHash(input, hash) {
    return bcrypt.compareSync(input, hash);
  }

  credhelper.createOrUpdateCredentials = function(req, callback) {
    var username = uuidv4();
    var password = uuidv4();
    models.credentials.upsert({
      id: req.params.app_id,
      username: generateHash(username),
      password: generateHash(password)
    }).then(function(updatedData) {
      if (updatedData) {
        logger.info('New credentials hasbeen generated successfully', {
          'app_id': req.params.app_id
        });
        callback(null, {
          'statusCode': HttpStatus.CREATED,
          'username': username,
          'password': password
        });
      }
      else {
        logger.info('Existing credentials hasbeen updated successfully', {
          'app_id': req.params.app_id
        });
        callback(null, {
          'statusCode': HttpStatus.OK,
          'username': username,
          'password': password
        });
      }
    }).catch(function(error) {
      logger.error('Failed to create custom metrics credentials', {
        'app_id': req.params.app_id,
        'error': error
      });
      error.statusCode = HttpStatus.INTERNAL_SERVER_ERROR;
      callback(error);
    });
  }

  credhelper.deleteCredentials = function(req, callback) {
    var appId = req.params.app_id;
    models.credentials.destroy({
      where: {
        id: appId
      }
    }).then(function(result) {
      if (result > 0) {
        logger.info('Successfully deleted the custom metrics credentials for application', {
          'app id': appId
        });
        callback(null, {
          'statusCode': HttpStatus.OK
        });
      }
      else {
        var error = {
          message: 'No custom metrics credentials exists with application',
          statusCode: HttpStatus.NOT_FOUND
        }
        logger.error('No custom metrics credentials exists with application', {
          'app id': appId,
          error: error
        });
        callback(error);
      }
    }).catch(function(error) {
      logger.error('Internal Error while deleting custom metrics credentials', {
        'app id': appId,
        'error': error
      });
      error.statusCode = HttpStatus.INTERNAL_SERVER_ERROR;
      callback(error);
    });
  };

  credhelper.validateCredentials = function(req, callback) {
    var appId = req.params.app_id;
    var username = req.query["username"];
    var password = req.query["password"];
    if (!username || !password) {
      var insufficientQueryparamError = new Error();
      insufficientQueryparamError.statusCode = HttpStatus.INTERNAL_SERVER_ERROR;
      insufficientQueryparamError.message = 'insufficient query parameters';
      logger.error('Failed to validate custom metrics credentials due to insufficient query parameters', {
        'app_id': appId,
        'error': insufficientQueryparamError
      });
      callback(insufficientQueryparamError);
      return;
    }
    models.credentials.find({
      where: {
        id: appId
      }
    }).then(function(creds) {
      if (!creds) {
        var error = {
          message: 'No credentials found',
          statusCode: HttpStatus.NOT_FOUND
        }
        logger.info('No credentials found', {
          'app_id': appId,
          'error': error
        });
        callback(error);
      }
      else {
        logger.info('Credentials hasbeen found successfully in database', {
          'app_id': appId
        });
        var isUsernameValid = validateHash(username, creds.username);
        var isPasswordValid = validateHash(password, creds.password);
        var isValidCred = isUsernameValid && isPasswordValid;
        callback(null, {
          'statusCode': HttpStatus.OK,
          'isValid': isValidCred
        });
      }
    }).catch(function(err) {
      logger.error('Failed to validate custom metrics credentials', {
        'app_id': appId,
        'error': err
      });
      err.statusCode = HttpStatus.INTERNAL_SERVER_ERROR;
      callback(err);
    });
  }
  return credhelper;
}
