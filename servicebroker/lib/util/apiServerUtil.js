'use strict';
var request = require('request');

module.exports = function(apiServerUri){
  var apiUtil = {};
  apiServerUri = apiServerUri.replace(/\/$/g, "").toLowerCase();
  apiUtil.attachPolicy= function(appId, policy, callback) {
    var options = {
      uri: apiServerUri + '/v1/apps/' + appId + '/policy',
      method: 'PUT',
      json: policy
    };
    request(options, function(error, response, body) {
    callback(error, response);

  });

  }
return apiUtil;
}