'use strict';
var request = require('request');
var fs = require('fs');

module.exports = function(settings) {
  var apiUtil = {};
  var apiserverTLSOptions = {
      key: fs.readFileSync(settings.apiserver.tls.keyFile),
      cert: fs.readFileSync(settings.apiserver.tls.certFile),
      ca: fs.readFileSync(settings.apiserver.tls.caCertFile)
};
  apiUtil.attachPolicy = function(appId, policy, callback) {
    var options = { 
      uri: settings.apiserver.uri + '/v1/policies/' + appId,
      timeout: settings.httpRequestTimeout,
      json: policy,
      cert: apiserverTLSOptions.cert,
      key: apiserverTLSOptions.key,
      ca: apiserverTLSOptions.ca 
    };
    request.put(options, function(error, response, body) {
      callback(error, response);
    });

    },
  apiUtil.detachPolicy = function(appId, callback) {
    var options = { 
      uri: settings.apiserver.uri + '/v1/policies/' + appId, 
      timeout: settings.httpRequestTimeout, 
      cert: apiserverTLSOptions.cert, 
      key: apiserverTLSOptions.key, 
      ca: apiserverTLSOptions.ca
    };
    request.delete(options, function(error, response, body) {
      callback(error, response);
    });

  }
  return apiUtil;
}
