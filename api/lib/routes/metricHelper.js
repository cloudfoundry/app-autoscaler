'use strict';

exports.parseParameter = function(req) {
  var routeHelper = require("./routeHelper");
  var metricType = req.params.metric_type;
  var instanceIndex = req.query["instance-index"];
  if (!metricType) {
    return { valid: false, message: "metric_type is required" };
  }
  if (routeHelper.isMissing(instanceIndex)) {
    instanceIndex = 0;
  } else {
    instanceIndex = parseInt(instanceIndex);
    if (!routeHelper.isInteger(instanceIndex)) {
      return { valid: false, message: "instance-index must be an integer" };
    }
  }
  var parseResult = routeHelper.parseParameter(req);
  if (parseResult.valid) {
    parseResult.parameters.metricType = metricType;
    parseResult.parameters.instanceIndex = instanceIndex;
  }
  return parseResult;
}