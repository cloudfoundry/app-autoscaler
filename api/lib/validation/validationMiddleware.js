'use strict'
var logger = require('../log/logger');
var _ = require('underscore');
var schemaStructureValidator = require('./schemaValidator');
var attributeValidator = require('./attributeValidator');
module.exports = function(req, res, next) {  
  schemaStructureValidator.validatePolicy(req.body, function(schemaValidationResult) {
    if(!_.isEmpty(schemaValidationResult)) {
      logger.error('Input policy JSON schema structure is not valid for application id : ' + 
        req.params.app_id, schemaValidationResult);
      next (schemaValidationResult);
    }
    else{
      logger.info('Input policy JSON schema structure is valid for application id : ' + 
        req.params.app_id);
      attributeValidator.validatePolicy(req.body, function(valueValidationResult) {
        if(!_.isEmpty(valueValidationResult)) {
          logger.error('Input policy JSON has incompatible values for application id : '
            + req.params.app_id , valueValidationResult);  
          next (valueValidationResult);
        }
        else{
          logger.info('Input policy JSON validated successfully.' +
           ' Creating policy for application id : ' + req.params.app_id);
          next();
        }
      });
    }
  });
};
