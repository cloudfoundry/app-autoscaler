'use strict';
var request = require('request');
var fs = require('fs');

module.exports = function(settings) {
  var apiUtil = {};

  var getOptions = function(appId, policy){
    var options = { 
      uri: settings.apiserver.uri + '/v1/apps/' + appId + '/policy',
      timeout: settings.httpRequestTimeout,
    };
    if(policy){
      options.json = policy;
    }
    if(settings.apiserver.tls){
      var apiserverTLSOptions = {
        key: fs.readFileSync(settings.apiserver.tls.keyFile),
        cert: fs.readFileSync(settings.apiserver.tls.certFile),
        ca: fs.readFileSync(settings.apiserver.tls.caCertFile)
      };
      options.cert = apiserverTLSOptions.cert,
      options.key = apiserverTLSOptions.key,
      options.ca = apiserverTLSOptions.ca;
    }
    return options
  }

  var credentialOptions = function(appId){
    var options = { 
      timeout: settings.httpRequestTimeout,
      uri: settings.apiserver.uri + '/v1/apps/' + appId + '/credentials',
    };
    if(settings.apiserver.tls){
      var apiserverTLSOptions = {
        key: fs.readFileSync(settings.apiserver.tls.keyFile),
        cert: fs.readFileSync(settings.apiserver.tls.certFile),
        ca: fs.readFileSync(settings.apiserver.tls.caCertFile)
      };
      options.cert = apiserverTLSOptions.cert,
      options.key = apiserverTLSOptions.key,
      options.ca = apiserverTLSOptions.ca;
    }
    return options
  }

  var credentialValidateOptions = function(parameters){
    var appId = parameters.appId;
    var options = { 
      timeout: settings.httpRequestTimeout,
      uri: settings.apiserver.uri + '/v1/apps/' + appId + '/credentials/validate',
      qs: { "username": parameters.username, "password":parameters.password }
    };
    if(settings.apiserver.tls){
      var apiserverTLSOptions = {
        key: fs.readFileSync(settings.apiserver.tls.keyFile),
        cert: fs.readFileSync(settings.apiserver.tls.certFile),
        ca: fs.readFileSync(settings.apiserver.tls.caCertFile)
      };
      options.cert = apiserverTLSOptions.cert,
      options.key = apiserverTLSOptions.key,
      options.ca = apiserverTLSOptions.ca;
    }
    return options
  }
  
  apiUtil.attachPolicy = function(appId, policy, callback) {
    var options = getOptions(appId,policy);
    request.put(options, function(error, response, body) {
      callback(error, response);
    });
  };
  apiUtil.detachPolicy = function(appId, callback) {
    var options = getOptions(appId);
    request.delete(options, function(error, response, body) {
      callback(error, response);
    });
  };
  apiUtil.getCreds = function(appId, callback){
    var options = credentialOptions(appId);
    request.post(options,function(error, response, body){
      callback(error,response);
    });
  };

  apiUtil.deleteCreds = function(appId, callback){
    var options = credentialOptions(appId);
    request.delete(options,function(error, response, body){
      callback(error,response);
    });
  };
  return apiUtil;
}
