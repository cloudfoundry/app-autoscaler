'use strict';
var request = require('request');

module.exports = function(apiServerUri) {
  var apiUtil = {};
  apiUtil.attachPolicy = function(appId, policy, callback) {
    request.put({ uri: apiServerUri + '/v1/policies/' + appId, json: policy }, function(error, response, body) {
      callback(error, response);
    });

  },
  apiUtil.detachPolicy = function(appId, callback) {
    request.delete({ uri: apiServerUri + '/v1/policies/' + appId}, function(error, response, body) {
      callback(error, response);
    });

  }
  return apiUtil;
}