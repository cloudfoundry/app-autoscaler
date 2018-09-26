"use strict";
module.exports = function(eventGeneratorSettings, httpClientTimeout) {
  var request = require("request");
  var logger = require("../log/logger");
  var HttpStatus = require("http-status-codes");
  var fs = require("fs");
  var eventGeneratorUtilObj = {};
  var getOptions = function(appId, metricType, startTime, endTime, order) {
    var options = {
      url: eventGeneratorSettings.uri + "/v1/apps/" + appId + "/aggregated_metric_histories/" + metricType,
      qs: { "start": startTime, "end": endTime, "order": order },
      method: "GET",
      json: true,
      timeout: httpClientTimeout,
    }
    if (eventGeneratorSettings.tls) {
      var eventGeneratorTLSOptions = {
        key: fs.readFileSync(eventGeneratorSettings.tls.keyFile),
        cert: fs.readFileSync(eventGeneratorSettings.tls.certFile),
        ca: fs.readFileSync(eventGeneratorSettings.tls.caCertFile)
      };
      options.cert = eventGeneratorTLSOptions.cert;
      options.key = eventGeneratorTLSOptions.key;
      options.ca = eventGeneratorTLSOptions.ca;
    }
    return options;
  }
  eventGeneratorUtilObj.getAggregatedMetricHistory = function getAggregatedMetricHistory(parameters, callback) {
    var appId = parameters.appId;
    var metricType = parameters.metricType;
    var startTime = parameters.startTime;
    var endTime = parameters.endTime;
    var order = parameters.order;
    logger.info("Get aggregated metric histories", { "appId": appId, "metricType": metricType, "startTime": startTime, "endTime": endTime, "order": order });
    var options = getOptions(appId, metricType, startTime, endTime, order);
    request(options, function(error, response, body) {
      if (error) {
        logger.error("Error occurred during get aggregated metric histories ", { "appId": appId, "metricType": metricType, "startTime": startTime, "endTime": endTime, "order": order, "error": error });
        error.statusCode = HttpStatus.INTERNAL_SERVER_ERROR;
        callback(error, null);
      } else if (response.statusCode === HttpStatus.OK) {
        logger.info("Get aggregated metric histories successfully", { "appId": appId, "metricType": metricType, "startTime": startTime, "endTime": endTime, "order": order });
        callback(null, { "statusCode": HttpStatus.OK, "body": body });
      } else {
        var errorObj = {
          "statusCode": response.statusCode,
          "message": body.message
        };
        logger.error("Error occurred during getting aggregated metric histories ", { "appId": appId, "metricType": metricType, "startTime": startTime, "endTime": endTime, "order": order, "error": errorObj });
        callback(errorObj, null);
      }
    });

  };

  return eventGeneratorUtilObj;
}
