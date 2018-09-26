"use strict";
module.exports = function(scalingEngineSettings, httpClientTimeout) {
  var request = require("request");
  var logger = require("../log/logger");
  var HttpStatus = require("http-status-codes");
  var fs = require("fs");
  var scalingEngineUtilObj = {};
  var getOptions = function(appId, startTime, endTime, order) {
    var options = {
      url: scalingEngineSettings.uri + "/v1/apps/" + appId + "/scaling_histories",
      qs: { "start": startTime, "end": endTime, "order": order },
      method: "GET",
      json: true,
      timeout: httpClientTimeout,
    }
    if (scalingEngineSettings.tls) {
      var scalingEngineTLSOptions = {
        key: fs.readFileSync(scalingEngineSettings.tls.keyFile),
        cert: fs.readFileSync(scalingEngineSettings.tls.certFile),
        ca: fs.readFileSync(scalingEngineSettings.tls.caCertFile)
      };
      options.cert = scalingEngineTLSOptions.cert;
      options.key = scalingEngineTLSOptions.key;
      options.ca = scalingEngineTLSOptions.ca;
    }
    return options;
  }
  scalingEngineUtilObj.getScalingHistory = function getScalingHistory(parameters, callback) {
    var appId = parameters.appId;
    var startTime = parameters.startTime;
    var endTime = parameters.endTime;
    var order = parameters.order;
    logger.info("Get scaling histories", { "appId": appId, "startTime": startTime, "endTime": endTime, "order": order });
    var options = getOptions(appId, startTime, endTime, order);
    request(options, function(error, response, body) {
      if (error) {
        logger.error("Error occurred during get scaling histories ", { "appId": appId, "startTime": startTime, "endTime": endTime, "error": error });
        error.statusCode = HttpStatus.INTERNAL_SERVER_ERROR;
        callback(error, null);
      } else if (response.statusCode === HttpStatus.OK) {
        logger.info("Get scaling histories successfully", { "appId": appId, "startTime": startTime, "endTime": endTime });
        callback(null, { "statusCode": HttpStatus.OK, "body": body });
      } else {
        var errorObj = {
          "statusCode": response.statusCode,
          "message": body.message
        };
        logger.error("Error occurred during getting scaling histories ", { "appId": appId, "startTime": startTime, "endTime": endTime, "error": errorObj });
        callback(errorObj, null);
      }
    });

  };

  return scalingEngineUtilObj;
}
