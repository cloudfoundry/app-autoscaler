'use strict';

exports.parseParameter = function(req) {
  var routeHelper = require("./routeHelper");
  var metricType = req.params.metric_type;
  if (!metricType) {
    return { valid: false, message: "metric_type is required" };
  }
  var parseResult = routeHelper.parseParameter(req);
  if (parseResult.valid) {
    parseResult.parameters.metricType = metricType;
  }
  return parseResult;
}