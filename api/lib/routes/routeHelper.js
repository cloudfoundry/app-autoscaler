'use strict';

var DESC = exports.DESC = "DESC";
var ASC = exports.ASC = "ASC";

var isMissing = exports.isMissing = function(value) {
  return typeof(value) === "undefined" || value === null;
}
var isInteger = exports.isInteger = function(value) {
  return Number.isInteger(value);
}
var isString = exports.isString = function(value) {
  return typeof(value) === "string";
}

exports.parseParameter = function(req) {
  var parameters = {};
  var appId = req.params.app_id;
  var startTime = req.query["start-time"];
  var endTime = req.query["end-time"];
  var orderDirection = req.query["order-direction"] ? req.query["order-direction"] : req.query["order"];
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
  if (isMissing(orderDirection)) {
    parameters["order"] = DESC;
  } else {
    if (!isString(orderDirection)) {
      return { valid: false, message: "order-direction must be a string" };
    } else {
      orderDirection = orderDirection.toUpperCase();
      if (orderDirection !== DESC && orderDirection !== ASC) {
        return { valid: false, message: "order-direction must be DESC or ASC" };
      } else {
        parameters["order"] = orderDirection;
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
    parameters.resultsPerPage = 50;
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

exports.pagination = function(resultList, page, resultsPerPage, req) {
  var result = {};
  var totalResults = resultList.length;
  var totalPages = 0;
  var perPage = Number.parseInt(resultsPerPage);
  var pageNo = Number.parseInt(page);
  var prevUrl = null;
  var nextUrl = null;
  totalPages = (totalResults % perPage) === 0 ? (totalResults / perPage) : (Number.parseInt(totalResults / perPage) + 1);
  var startIndex = (pageNo - 1) * perPage;
  var endIndex = startIndex + perPage;
  var resources = resultList.slice(startIndex, endIndex);
  var originalUrl = req.originalUrl
  if (page > 1 && page <= totalPages + 1) {
    var prevPage = page - 1;
    prevUrl = getPageUrl(req, prevPage);
  }
  if (page < totalPages) {
    var nextPage = page + 1;
    nextUrl = getPageUrl(req, nextPage);
  }
  result.total_results = totalResults;
  result.total_pages = Number.parseInt(totalPages);
  result.page = Number.parseInt(pageNo);
  result.prev_url = prevUrl;
  result.next_url = nextUrl;
  result.resources = resources;
  return result;
}

var getPageUrl = exports.getPageUrl = function(req, targetPageNo) {
  var queries = req.query;
  if (!queries || !queries["page"]) {
    return null;
  }
  queries["page"] = targetPageNo;
  var pageStr = req.baseUrl + req.path + "?";
  for (var key in queries) {
    pageStr += "&" + key + "=" + queries[key];
  }
  pageStr = pageStr.replace("&", "");
  return pageStr;
}
