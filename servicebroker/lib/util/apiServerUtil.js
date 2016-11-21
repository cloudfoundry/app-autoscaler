'use strict';
var request = require('request');

module.exports = function(apiServerUri,httpRequestTimeout) {
  var apiUtil = {};
  apiUtil.attachPolicy = function(appId, policy, callback) {
      request.put({ uri: apiServerUri + '/v1/policies/' + appId, timeout: httpRequestTimeout, json: policy }, function(error, response, body) {
        callback(error, response);
      });

    },
    apiUtil.detachPolicy = function(appId, callback) {
      request.delete({ uri: apiServerUri + '/v1/policies/' + appId, timeout: httpRequestTimeout}, function(error, response, body) {
        callback(error, response);
      });

    }
  return apiUtil;
}
