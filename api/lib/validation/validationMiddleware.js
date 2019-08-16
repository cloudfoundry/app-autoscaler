'use strict'
module.exports = function (settings) {
  var logger = require('../log/logger');
  var _ = require('underscore');
  var schemaStructureValidator = require('./schemaValidator');
  var attributeValidator = require('./attributeValidator');
  var validationMiddleware = {};
  validationMiddleware.Validate = function (req, res, next) {
    schemaStructureValidator.validatePolicy(req.body, settings, function (schemaValidationResult) {
      if (!_.isEmpty(schemaValidationResult)) {
        logger.error('Input policy JSON schema structure is not valid',
          { 'app id': req.params.app_id, 'error': schemaValidationResult });
        next(schemaValidationResult);
      }
      else {
        logger.info('Input policy JSON schema structure is valid', { 'app id': req.params.app_id });
        attributeValidator.validatePolicy(req.body, function (valueValidationResult) {
          if (!_.isEmpty(valueValidationResult)) {
            logger.error('Input policy JSON has incompatible values',
              { 'app id': req.params.app_id, 'error': valueValidationResult });
            next(valueValidationResult);
          }
          else {
            logger.info('Input policy JSON validated successfully. Creating policy..',
              { 'app id': req.params.app_id });
            next();
          }
        });
      }
    });
  };
  return validationMiddleware;
};
