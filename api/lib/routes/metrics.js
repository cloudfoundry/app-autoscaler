'use strict';
module.exports = function(settings) {
  var express = require("express");
  var router = express.Router();
  var logger = require("../log/logger");
  var HttpStatus = require("http-status-codes");
  var metricsCollectorUtil = require("../utils/metricsCollectorUtils")(settings.metricsCollector);
  var metricHelper = require("./metricHelper");
  var routeHelper = require("./routeHelper");

  router.get("/:app_id/metric_histories/:metric_type", function(req, resp) {
    var appId = req.params.app_id;
    var metricType = req.params.metric_type;
    var startTime = req.query["start-time"];
    var endTime = req.query["end-time"];
    var order = req.query["order"];
    var page = req.query["page"];
    var resultsPerPage = req.query["results-per-page"];
    logger.info("Get metrics", { "app_id": appId, "metric_type":metricType, "start-time": startTime, "end-time": endTime, "order": order, "page": page, "results-per-page": resultsPerPage });
    var parseResult = metricHelper.parseParameter(req);
    if (!parseResult.valid) {
      logger.error("Failed to get metrics", { "app_id": appId, "metric_type":metricType, "start-time": startTime, "end-time": endTime, "order": order, "page": page, "results-per-page": resultsPerPage, "message": parseResult.message });
      resp.status(HttpStatus.BAD_REQUEST).json({ "description": parseResult.message });
      return;
    }
    var parameters = parseResult.parameters;
    metricsCollectorUtil.getMetricHistory(parameters, function(err, result) {
      var responseBody = {};
      var statusCode;
      if (err) {
        statusCode = err.statusCode;
        responseBody.description = err.message;
      } else {
        statusCode = result.statusCode;
        if (statusCode === HttpStatus.OK) {
          var page = parameters.page;
          var resultsPerPage = parameters.resultsPerPage;
          responseBody = routeHelper.pagination(result.body, page, resultsPerPage);
        } else {
          responseBody.description = result.message;
        }
      }
      resp.status(statusCode).json(responseBody);
    });
  });
  return router;
}
