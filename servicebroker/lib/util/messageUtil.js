'use strict';
var path = require('path');
var fs = require('fs');
var catalog = JSON.parse(
  fs.readFileSync(path.join(__dirname, '../../config/catalog.json'), 'utf8'));
module.exports = function(){
  var messageUtil = {};
  var message = {
    POLICY_REQUIRED: "Policy is required as a parameter for binding request.",
    INVALID_POLICY_FORMAT: "Invalid policy format.",
    DUPLICATED_BIND: "Another " + catalog.services[0].name + " service instance already bound to current application ${applicationId}. The multiple binding is not supported.",
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