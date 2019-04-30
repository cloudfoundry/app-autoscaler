'use strict';
module.exports = function(models, credentialCache, cacheTTL) {
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

  function validateCredentialDetails(username,usernamehash,password, passwordhash){
    var isUsernameValid = validateHash(username, usernamehash);
    var isPasswordValid = validateHash(password, passwordhash);
    var isValidCred = isUsernameValid && isPasswordValid;
    return isValidCred;
  }

  credhelper.createOrUpdateCredentials = function(req, callback) {
    var username = uuidv4();
    var password = uuidv4();
    var appId = req.params.app_id;
    models.credentials.upsert({
      id: appId,
      username: generateHash(username),
      password: generateHash(password)
    }).then(function(createdData) {
      if (createdData) {
        logger.info('New credentials has been generated successfully', {
          'app_id': appId
        });
        callback(null, {
          'statusCode': HttpStatus.CREATED,
          'username': username,
          'password': password
        });
      }
      else {
        logger.info('Existing credentials has been updated successfully', {
          'app_id': appId
        });
        var deleted = credentialCache.del(appId);
        if (deleted != 1) {
          logger.info('Cache invalidation failed', {
            'app_id': appId
          });
        }
        callback(null, {
          'statusCode': HttpStatus.OK,
          'username': username,
          'password': password
        });
      }
    }).catch(function(error) {
      logger.error('Failed to create custom metrics credentials', {
        'app_id': appId,
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
        var deleted = credentialCache.del(appId);
        if (deleted != 1) {
          logger.info('Cache invalidation failed', {
            'app_id': appId
          });
        }
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
    var creds,isValidCred,cachedCred;
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
    // Try to find credentials in cache
    try{
      creds = credentialCache.get(appId, true);
      isValidCred = validateCredentialDetails(username, creds.username, password, creds.password);
      // If cache contains old or invalid credentials
      if (!isValidCred){
        logger.info('Credentials not valid', {
          'app_id': appId,
          'isValid': isValidCred
        });
        throw new Error('Invalid or old credentials found in cache');
      }
      else {
        logger.info('valid credentials hasbeen found successfully in cache', {
          'app_id': appId,
          'isValid': isValidCred
        });
        callback(null, {
          'statusCode': HttpStatus.OK,
          'isValid': isValidCred
        });
      }
    }
    catch(err){
    // Did not find credentials in cache, lets find in database.
    models.credentials.findOne({
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
        isValidCred = validateCredentialDetails(username, creds.username, password, creds.password);
        logger.info('Credentials hasbeen found successfully in database', {
          'app_id': appId,
          'isValid': isValidCred
        });
        cachedCred = {
            'username': creds.username,
            'password': creds.password
        };
        var isCached = credentialCache.set(appId, cachedCred, cacheTTL);
        logger.info('Credential cached',{ 'app_id':appId, 'isCached':isCached });
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
  }
  return credhelper;
}
