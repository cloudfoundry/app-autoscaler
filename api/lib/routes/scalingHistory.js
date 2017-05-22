'use strict'
module.exports = function(settings) {
  var express = require('express');
  var path = require('path');
  var request = require('request');
  var HttpStatus = require('http-status-codes');
  var router = express.Router();
  var logger = require('../log/logger');
  var scalingHistory = require(path.join(__dirname, '../models'))(settings.db).scalinghistory;

  function validateParameter(appId, startTime, endTime, page, resultsPerPage) {
    if (!appId) {
      return { valid: false, "description": "app guid is required" };
    }
    if (startTime == null || ((typeof startTime) === "undefined")) {
      return { valid: false, message: "start-time is required" };
    }
    if (!Number.isInteger(parseInt(startTime))) {
      return { valid: false, message: "start-time must be an integer" };
    }
    if (endTime == null || ((typeof endTime) === "undefined")) {
      return { valid: false, message: "end-time is required" };
    }
    if (!Number.isInteger(parseInt(endTime))) {
      return { valid: false, message: "end-time must be an integer" };
    }
    if (startTime > endTime) {
      return { valid: false, message: "start-time must be smaller than end-time" };
    }
    if (page == null && ((typeof page) === "undefined")) {
      return { valid: false, message: "page is required" };
    }
    if (!Number.isInteger(parseInt(page))) {
      return { valid: false, message: "page must be an integer" };
    }
    if (resultsPerPage == null && ((typeof resultsPerPage) === "undefined")) {
      return { valid: false, message: "results-per-page is required" };
    }
    if (!Number.isInteger(parseInt(resultsPerPage))) {
      return { valid: false, message: "results-per-page must be an integer" };
    }
    return { valid: true }
  }

  function getCondition(appId, startTime, endTime, page, resultsPerPage) {
    var condition = {};
    var whereObj = {};
    var timstampAndObj = {};
    whereObj.app_guid = appId;
    timstampAndObj.$gte = parseInt(startTime);
    timstampAndObj.$lte = parseInt(endTime);
    condition.limit = parseInt(resultsPerPage);
    condition.offset = parseInt(resultsPerPage) * parseInt(page);
    whereObj.timestamp = timstampAndObj;
    condition.where = whereObj;
    condition.order = [
      ["timestamp", "DESC"]
    ];
    return condition;
  }

  router.get('/:guid/scaling_history', function(req, resp) {
    var appId = req.params.guid;
    var startTime = req.query["start-time"];
    var endTime = req.query["end-time"];
    var page = req.query["page"];
    var resultsPerPage = req.query["results-per-page"];
    logger.error("Get scalinghistory", { "app_guid": appId, "start-time": startTime, "end-time": endTime, "page": page, "results-per-page": resultsPerPage });
    var validateResult = validateParameter(appId, startTime, endTime, page, resultsPerPage);
    if (!validateResult.valid) {
      resp.status(HttpStatus.BAD_REQUEST).json({ "description": validateResult.message });
      return;
    }
    var condition = getCondition(appId, startTime, endTime, page, resultsPerPage);
    scalingHistory.findAndCountAll(condition).then(function(queryResult) {
      var result = {};
      var totalPages = 0;
      var totalResults = queryResult.count;
      var rows = queryResult.rows;
      var perPage = Number.parseInt(resultsPerPage);
      totalPages = (totalResults % perPage) === 0 ? (totalResults / perPage) : (totalResults / perPage + 1);
      for (var i = 0; i < rows.length; i++) {
        //the timestamp of scalinghistory is defined as bigint in postgres
        //sequelize will treat bigint as string, so there convert timestamp(string) to integer
        rows[i].timestamp = Number.parseInt(rows[i].timestamp);
      }
      result.total_results = totalResults;
      result.total_pages = Number.parseInt(totalPages);
      result.page = Number.parseInt(page);
      result.resources = rows;
      resp.status(HttpStatus.OK).json(result);
    }).catch(function(err) {
      logger.error("Failed to get scalinghistory", { "app_guid": appId, "start-time": startTime, "end-time": endTime, "page": page, "results-per-page": resultsPerPage, "error": err });
      resp.status(HttpStatus.INTERNAL_SERVER_ERROR);
    });
  });
  return router;
}
