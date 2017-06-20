'use strict';

exports.parseParameter = function(req) {
  var routeHelper = require("./routeHelper");
  var parameters = {};
  var appId = req.params.app_id;
  var metricType = req.params.metric_type;
  var startTime = req.query["start-time"];
  var endTime = req.query["end-time"];
  var order = req.query["order"];
  var page = req.query["page"];
  var resultsPerPage = req.query["results-per-page"];
  var ASC = routeHelper.ASC;
  var DESC = routeHelper.DESC;
  if (!appId) {
    return { valid: false, message: "app_id is required" };
  } else {
    parameters.appId = appId;
  }
  if (!metricType) {
    return { valid: false, message: "metric_type is required" };
  } else {
    parameters.metricType = metricType;
  }
  if (routeHelper.isMissing(startTime)) {
    parameters.startTime = 0;
  } else {
    startTime = parseInt(startTime);
    if (!routeHelper.isInteger(startTime)) {
      return { valid: false, message: "start-time must be an integer" };
    } else {
      parameters.startTime = startTime;
    }
  }

  if (routeHelper.isMissing(endTime)) {
    parameters.endTime = -1;
  } else {
    endTime = parseInt(endTime);
    if (!routeHelper.isInteger(endTime)) {
      return { valid: false, message: "end-time must be an integer" };
    } else {
      parameters.endTime = endTime;
    }
  }
  if (routeHelper.isMissing(order)) {
    parameters.order = ASC;
  } else {
    if (!routeHelper.isString(order)) {
      return { valid: false, message: "order must be a string" };
    } else {
      order = order.toUpperCase();
      if (order !== DESC && order !== ASC) {
        return { valid: false, message: "order must be DESC or ASC" };
      } else {
        parameters.order = order;
      }
    }
  }
  if (routeHelper.isMissing(page)) {
    parameters.page = 1;
  } else {
    page = parseInt(page);
    if (!routeHelper.isInteger(page)) {
      return { valid: false, message: "page must be an integer" };
    }
    if (page <= 0) {
      return { valid: false, message: "page must be greater than 0" };
    }
    parameters.page = page;
  }

  if (routeHelper.isMissing(resultsPerPage)) {
    parameters.resultsPerPage = 50;
  } else {
    resultsPerPage = parseInt(resultsPerPage)
    if (!routeHelper.isInteger(resultsPerPage)) {
      return { valid: false, message: "results-per-page must be an integer" };
    }
    if (resultsPerPage <= 0) {
      return { valid: false, message: "results-per-page must be greater than 0" };
    }
    parameters.resultsPerPage = resultsPerPage;
  }


  return { valid: true, parameters: parameters }
}
