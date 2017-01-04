'use strict';
var request = require('request');

module.exports = function(apiServerUri,httpRequestTimeout,tlsOptions) {
  var apiUtil = {};
  apiUtil.attachPolicy = function(appId, policy, callback) {
    var options = { 
      uri: apiServerUri + '/v1/policies/' + appId,
      timeout: httpRequestTimeout,
      json: policy,
      cert: tlsOptions.cert,
      key: tlsOptions.key,
      ca: tlsOptions.ca 
    };
    request.put(options, function(error, response, body) {
      callback(error, response);
    });

    },
  apiUtil.detachPolicy = function(appId, callback) {
    var options = { 
      uri: apiServerUri + '/v1/policies/' + appId, 
      timeout: httpRequestTimeout, 
      cert: tlsOptions.cert, 
      key: tlsOptions.key, 
      ca: tlsOptions.ca
    };
    request.delete(options, function(error, response, body) {
      callback(error, response);
    });

  }
  return apiUtil;
}
