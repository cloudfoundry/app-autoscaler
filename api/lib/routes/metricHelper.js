'use strict';

exports.parseParameter = function(req) {
  var routeHelper = require("./routeHelper");
  var parseResult = routeHelper.parseParameter(req);
  if (!parseResult.valid) {
    return parseResult;
  }
  var metricType = req.params.metric_type;
  var instanceIndex = req.query["instance-index"];
  if (!metricType) {
    return { valid: false, message: "metric_type is required" };
  } else {
    parseResult.parameters.metricType = metricType;
  }
  if (!routeHelper.isMissing(instanceIndex)) {
    instanceIndex = parseInt(instanceIndex);
    if (!routeHelper.isInteger(instanceIndex)) {
      return { valid: false, message: "instance-index must be an integer" };
    } else if (instanceIndex < 0) {
      return { valid: false, message: "instance-index must be greater than or equal to 0" };
    } else {
      parseResult.parameters.instanceIndex = instanceIndex;
    }
  }

  return parseResult;
}