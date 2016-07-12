'use strict'
var logger = require('../log/logger');
var _ = require('underscore');
var schemaStructureValidator = require('./schemaValidator');
var attributeValidator = require('./attributeValidator');
module.exports = function(req, res, next) {  
  schemaStructureValidator.validatePolicy(req.body, function(validationResult) {
    if(!_.isEmpty(validationResult)) {
      logger.error('Input policy json schema is not valid', validationResult);
      next (validationResult);
    }
    else{
      logger.log('Input Policy JSON schema structure is valid');
      attributeValidator.validatePolicy(req.body, function(valueValidationResult) {
        if(!_.isEmpty(valueValidationResult)) {
          logger.error('Input policy json has incompatible values', valueValidationResult);  
          next (valueValidationResult);
        }
        else{
          logger.log('Policy JSON validated successfully. Creating policy for application id : ' +
          req.params.app_id);
          next();
        }
      });
    }
  });
};
