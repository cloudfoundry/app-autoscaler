"use strict";
module.exports = function(serviceBorkerSettings) {
  var request = require("request");
  var logger = require("../log/logger");
  var HttpStatus = require("http-status-codes");
  var fs = require("fs");
  var serviceBorkerUtilObj = {};
  var getOptions = function(appId) {
    var options = {
      url: serviceBorkerSettings.uri + "/v1/apps/" + appId + "/service_bindings",
      method: "GET",
      json: true,
      timeout: 10000,
    }
    if (serviceBorkerSettings.tls) {
      var serviceBorkerTLSOptions = {
        key: fs.readFileSync(serviceBorkerSettings.tls.keyFile),
        cert: fs.readFileSync(serviceBorkerSettings.tls.certFile),
        ca: fs.readFileSync(serviceBorkerSettings.tls.caCertFile)
      };
      options.cert = serviceBorkerTLSOptions.cert;
      options.key = serviceBorkerTLSOptions.key;
      options.ca = serviceBorkerTLSOptions.ca;
    }
    return options;
  }
  serviceBorkerUtilObj.checkBinding = function checkBinding(parameters, callback) {
    var appId = parameters.app_id;
    logger.info("Checking binding", { "appId": appId });
    var options = getOptions(appId);
    request(options, function(error, response, body) {
      if (error) {
        logger.error("Error occurred during check binding ", { "appId": appId, "error": error });
        error.statusCode = HttpStatus.INTERNAL_SERVER_ERROR;
        callback(error, null);
      } else if (response.statusCode === HttpStatus.OK) {
        logger.info("Binding info exists", { "appId": appId });
        callback(null, { "statusCode": HttpStatus.OK });
      } else if (response.statusCode === HttpStatus.NOT_FOUND) {
        if (body && body.message && body.message === "binding_info_not_found") {
          logger.info("Binding info does not exist", { "appId": appId });
          callback(null, { "statusCode": HttpStatus.NOT_FOUND});
        } else {
          logger.info("Error occurred during check binding, service broker can not be reached", { "appId": appId,"response":body });
          callback({ "statusCode": HttpStatus.INTERNAL_SERVER_ERROR }, null);
        }

      } else {
        var errorObj = {
          "statusCode": response.statusCode,
        };
        logger.error("Error occurred during checking binding ", { "appId": appId, "error": errorObj });
        callback(errorObj, null);
      }
    });

  };

  return serviceBorkerUtilObj;
}