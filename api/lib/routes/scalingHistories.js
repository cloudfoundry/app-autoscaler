"use strict"
module.exports = function(settings) {
  var express = require("express");
  var router = express.Router();
  var logger = require("../log/logger");
  var HttpStatus = require("http-status-codes");
  var scalingEngineUtil = require("../utils/scalingEngineUtils")(settings.scalingEngine);
  var scalingHistoryHelper = require("./scalingHistoryHelper");

  router.get("/:guid/scaling_histories", function(req, resp) {
    var guid = req.params.guid;
    var startTime = req.query["start-time"];
    var endTime = req.query["end-time"];
    var order = req.query["order"];
    var page = req.query["page"];
    var resultsPerPage = req.query["results-per-page"];
    logger.info("Get scalinghistory", { "app_guid": guid, "start-time": startTime, "end-time": endTime, "order": order, "page": page, "results-per-page": resultsPerPage });
    var validateResult = scalingHistoryHelper.validateParameter(req);
    if (!validateResult.valid) {
      logger.error("Failed to get scaling history", { "app_guid": guid, "start-time": startTime, "end-time": endTime, "order": order, "page": page, "results-per-page": resultsPerPage, "message": validateResult.message });
      resp.status(HttpStatus.BAD_REQUEST).json({ "description": validateResult.message });
      return;
    }
    scalingEngineUtil.getScalingHistory(req, function(err, result) {
      var responseBody = {};
      var statusCode = HttpStatus.OK;
      if (err) {
        statusCode = err.statusCode;
        responseBody.description = err.message;
      } else {
        statusCode = result.statusCode;
        if (result.statusCode === HttpStatus.OK) {
          var page = req.query["page"];
          var resultsPerPage = req.query["results-per-page"];
          responseBody = scalingHistoryHelper.pagination(result.body, page, resultsPerPage);
        } else {
          responseBody.description = result.message;
        }
      }
      resp.status(statusCode).json(responseBody);
    });
  });
  return router;
}
