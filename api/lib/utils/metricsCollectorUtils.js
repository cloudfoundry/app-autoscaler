"use strict";
module.exports = function(metricsCollectorSettings, httpClientTimeout) {
  var request = require("request");
  var logger = require("../log/logger");
  var HttpStatus = require("http-status-codes");
  var fs = require("fs");
  var metricsCollectorUtilObj = {};
  var getOptions = function(appId, metricType, instanceIndex, startTime, endTime, order) {
    var options = {
      url: metricsCollectorSettings.uri + "/v1/apps/" + appId + "/metric_histories/" + metricType,
      qs: { "instanceindex": instanceIndex, "start": startTime, "end": endTime, "order": order },
      method: "GET",
      json: true,
      timeout: httpClientTimeout,
    }
    if (metricsCollectorSettings.tls) {
      var metricsCollectorTLSOptions = {
        key: fs.readFileSync(metricsCollectorSettings.tls.keyFile),
        cert: fs.readFileSync(metricsCollectorSettings.tls.certFile),
        ca: fs.readFileSync(metricsCollectorSettings.tls.caCertFile)
      };
      options.cert = metricsCollectorTLSOptions.cert;
      options.key = metricsCollectorTLSOptions.key;
      options.ca = metricsCollectorTLSOptions.ca;
    }
    return options;
  }
  metricsCollectorUtilObj.getMetricHistory = function getMetricHistory(parameters, callback) {
    var appId = parameters.appId;
    var metricType = parameters.metricType;
    var instanceIndex = parameters.instanceIndex;
    var startTime = parameters.startTime;
    var endTime = parameters.endTime;
    var order = parameters.order;
    logger.info("Get metric histories", { "appId": appId, "metricType": metricType, "instanceIndex": instanceIndex, "startTime": startTime, "endTime": endTime, "order": order });
    var options = getOptions(appId, metricType, instanceIndex, startTime, endTime, order);
    request(options, function(error, response, body) {
      if (error) {
        logger.error("Error occurred during get metric histories ", { "appId": appId, "metricType": metricType, "instanceIndex": instanceIndex, "startTime": startTime, "endTime": endTime, "order": order, "error": error });
        error.statusCode = HttpStatus.INTERNAL_SERVER_ERROR;
        callback(error, null);
      } else if (response.statusCode === HttpStatus.OK) {
        logger.info("Get metric histories successfully", { "appId": appId, "metricType": metricType, "instanceIndex": instanceIndex, "startTime": startTime, "endTime": endTime, "order": order });
        callback(null, { "statusCode": HttpStatus.OK, "body": body });
      } else {
        var errorObj = {
          "statusCode": response.statusCode,
          "message": body.message
        };
        logger.error("Error occurred during getting metric histories ", { "appId": appId, "metricType": metricType, "instanceIndex": instanceIndex, "startTime": startTime, "endTime": endTime, "order": order, "error": errorObj });
        callback(errorObj, null);
      }
    });

  };

  return metricsCollectorUtilObj;
}
