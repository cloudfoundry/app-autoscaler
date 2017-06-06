'use strict';

function isMissing(value) {
  return typeof(value) === "undefined" || value === null;
}

function isInteger(value) {
  return Number.isInteger(value);
}
exports.validateParameter = function(req) {
  var appId = req.params.guid;
  var startTime = req.query["start-time"];
  var endTime = req.query["end-time"];
  var order = req.query["order"];
  var page = req.query["page"];
  var resultsPerPage = req.query["results-per-page"];
  if (!appId) {
    return { valid: false, message: "app guid is required" };
  }

  if (isMissing(startTime)) {
    return { valid: false, message: "start-time is required" };
  }
  startTime = parseInt(startTime);
  if (!isInteger(startTime)) {
    return { valid: false, message: "start-time must be an integer" };
  }

  if (isMissing(endTime)) {
    return { valid: false, message: "end-time is required" };
  }
  endTime = parseInt(endTime);
  if (!isInteger(endTime)) {
    return { valid: false, message: "end-time must be an integer" };
  }

  if (startTime > endTime) {
    return { valid: false, message: "start-time must be smaller than end-time" };
  }

  if (isMissing(page)) {
    return { valid: false, message: "page is required" };
  }
  page = parseInt(page);
  if (!isInteger(page)) {
    return { valid: false, message: "page must be an integer" };
  }
  if (page < 0) {
    return { valid: false, message: "page must be greater or equal than 0" };
  }

  if (isMissing(resultsPerPage)) {
    return { valid: false, message: "results-per-page is required" };
  }
  resultsPerPage = parseInt(resultsPerPage)
  if (!isInteger(resultsPerPage)) {
    return { valid: false, message: "results-per-page must be an integer" };
  }
  if (resultsPerPage < 0) {
    return { valid: false, message: "results-per-page must be greater or equal than 0" };
  }

  return { valid: true }
}

exports.pagination = function(list, page, resultsPerPage) {
  var result = {};
  var totalResults = list.length;
  var totalPages = 0;
  var perPage = Number.parseInt(resultsPerPage);
  var pageNo = Number.parseInt(page);
  totalPages = (totalResults % perPage) === 0 ? (totalResults / perPage) : (totalResults / perPage + 1);
  var startIndex = 0 + pageNo * perPage;
  var endIndex = startIndex + perPage;
  var resources = list.slice(startIndex, endIndex);
  result.total_results = totalResults;
  result.total_pages = Number.parseInt(totalPages);
  result.page = Number.parseInt(pageNo);
  result.resources = resources;
  return result;
}
