"use strict";
module.exports = function(scalingEngineSettings) {
  var request = require("request");
  var logger = require("../log/logger");
  var HttpStatus = require("http-status-codes");
  var fs = require("fs");
  var scalingEngineUtilObj = {};
  var getOptions = function(appGuid, startTime, endTime, order) {
    var options = {
      url: scalingEngineSettings.uri + "/v1/apps/" + appGuid + "/scaling_histories",
      qs: { "start": startTime, "end": endTime, "order": order },
      method: "GET",
      json: true,
      timeout: 10000,
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
  scalingEngineUtilObj.getScalingHistory = function getScalingHistory(req, callback) {
    var appGuid = req.params.guid;
    var startTime = req.query["start-time"];
    var endTime = req.query["end-time"];
    var order = req.query["order"];
    logger.info("Get scaling histories", { "appGuid": appGuid, "startTime": startTime, "endTime": endTime, "order": order });
    var options = getOptions(appGuid, startTime, endTime, order);
    request(options, function(error, response, body) {
      if (error) {
        logger.error("Error occurred during get scaling histories ", { "appGuid": appGuid, "startTime": startTime, "endTime": endTime, "error": error });
        error.statusCode = HttpStatus.INTERNAL_SERVER_ERROR;
        callback(error, null);
      } else if (response.statusCode === HttpStatus.OK) {
        logger.info("Get scaling histories successfully", { "appGuid": appGuid, "startTime": startTime, "endTime": endTime });
        callback(null, { "statusCode": HttpStatus.OK, "body": body });
      } else {
        var errorObj = {
          "statusCode": response.statusCode,
          "message": body.message
        };
        logger.error("Error occurred during getting scaling histories ", { "appGuid": appGuid, "startTime": startTime, "endTime": endTime, "error": errorObj });
        callback(errorObj, null);
      }
    });

  };

  return scalingEngineUtilObj;
}
