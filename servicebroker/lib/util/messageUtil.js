'use strict';
var path = require('path');
var fs = require('fs');
module.exports = function(catalog){
  var messageUtil = {};
  var message = {
    SERVICEINSTANCE_NOT_EXIST: "Target service instance ${serviceInstanceId} does not exist",
    POLICY_REQUIRED: "Policy is required as a parameter.",
    DUPLICATE_BIND: "A " + catalog.services[0].name + " service instance is already bound to application ${applicationId}. Multiple bindings are not supported.",
  };
  messageUtil.getMessage = function(key, params){
  	var msg = message[key];
  	if(typeof(params) !== "undefined"){
  		for(var paramKey in params){
  			msg = msg.replace("${" + paramKey + "}", params[paramKey]);
  		}
  	}
  	return msg;
  };
  return messageUtil;
};
