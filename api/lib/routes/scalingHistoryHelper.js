'use strict';
var DESC = "desc";
var ASC = "asc";

function isMissing(value) {
  return typeof(value) === "undefined" || value === null;
}
function isInteger(value) {
  return Number.isInteger(value);
}
var isString = function(value) {
  return typeof(value) === "string";
}
exports.parseParameter = function(req) {
  var parameters = {};
  var appId = req.params.app_id;
  var startTime = req.query["start-time"];
  var endTime = req.query["end-time"];
  var order = req.query["order"];
  var page = req.query["page"];
  var resultsPerPage = req.query["results-per-page"];
  if (!appId) {
    return { valid: false, message: "app_id is required" };
  } else {
    parameters.appId = appId;
  }
  if (isMissing(startTime)) {
    parameters.startTime = 0;
  } else {
    startTime = parseInt(startTime);
    if (!isInteger(startTime)) {
      return { valid: false, message: "start-time must be an integer" };
    } else {
      parameters.startTime = startTime;
    }
  }

  if (isMissing(endTime)) {
    parameters.endTime = -1;
  } else {
    endTime = parseInt(endTime);
    if (!isInteger(endTime)) {
      return { valid: false, message: "end-time must be an integer" };
    } else {
      parameters.endTime = endTime;
    }
  }
  if (isMissing(order)) {
    parameters.order = DESC;
  } else {
    if (!isString(order)) {
      return { valid: false, message: "order must be a string" };
    } else {
      order = order.toLowerCase();
      if (order !== DESC && order !== ASC) {
        return { valid: false, message: "order must be desc or asc" };
      }else{
        parameters.order = order;
      }
    }
  }
  if (isMissing(page)) {
    parameters.page = 1;
  } else {
    page = parseInt(page);
    if (!isInteger(page)) {
      return { valid: false, message: "page must be an integer" };
    }
    if (page <= 0) {
      return { valid: false, message: "page must be greater than 0" };
    }
    parameters.page = page;
  }

  if (isMissing(resultsPerPage)) {
    parameters.resultsPerPage = 10;
  } else {
    resultsPerPage = parseInt(resultsPerPage)
    if (!isInteger(resultsPerPage)) {
      return { valid: false, message: "results-per-page must be an integer" };
    }
    if (resultsPerPage <= 0) {
      return { valid: false, message: "results-per-page must be greater than 0" };
    }
    parameters.resultsPerPage = resultsPerPage;
  }


  return { valid: true, parameters: parameters }
}

exports.pagination = function(scalingHistoryList, page, resultsPerPage) {
  var result = {};
  var totalResults = scalingHistoryList.length;
  var totalPages = 0;
  var perPage = Number.parseInt(resultsPerPage);
  var pageNo = Number.parseInt(page);
  totalPages = (totalResults % perPage) === 0 ? (totalResults / perPage) : (totalResults / perPage + 1);
  var startIndex = (pageNo - 1) * perPage;
  var endIndex = startIndex + perPage;
  var resources = scalingHistoryList.slice(startIndex, endIndex);
  result.total_results = totalResults;
  result.total_pages = Number.parseInt(totalPages);
  result.page = Number.parseInt(pageNo);
  result.resources = resources;
  return result;
}
