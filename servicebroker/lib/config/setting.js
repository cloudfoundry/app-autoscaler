'use strict';
var path = require('path');
module.exports = function(settings) {
  var settingObj = {};
  var db = function(dbUri) {
    var uri = dbUri.replace(/\/$/g, "").toLowerCase();
    var name = uri.slice(uri.lastIndexOf("/") + 1, uri.length);
    var server = uri.slice(0, uri.lastIndexOf("/"));
    return {
      uri: uri,
      name: name,
      server: server
    };
  };

  var apiServer = function(apiServerUri) {
    return apiServerUri.replace(/\/$/g, "").toLowerCase();
  };
  settingObj.getSetting = function() {
    return {
      port: settings.port,
      username: settings.username,
      password: settings.password,
      db: db(settings.dbUri),
      apiServerUri: apiServer(settings.apiServerUri)
    };
  };
  settingObj.validate = function(){
    if(settings.port == null || typeof(settings.port) != "number" || String(settings.port).match(/^([0-9]|[1-9]\d|[1-9]\d{2}|[1-9]\d{3}|[1-5]\d{4}|6[0-4]\d{3}|65[0-4]\d{2}|655[0-2]\d|6553[0-5])$/) == null){
      return {valid:false,message:"The value of port should not be null or its format is invalid"};
    }
    if(settings.username == null || typeof(settings.username) != "string"){
      return {valid:false,message:"The value of username should not be null or its format is invalid"};
    }
    if(settings.password == null || typeof(settings.password) != "string"){
      return {valid:false,message:"The value of password should not be null or its format is invalid"};
    }
    if(settings.dbUri == null || typeof(settings.dbUri) != "string" || settings.dbUri.match(/^postgres:\/\/\w+@((\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])|((\w+\.)*\w+)):(([0-9]|[1-9]\d|[1-9]\d{2}|[1-9]\d{3}|[1-5]\d{4}|6[0-4]\d{3}|65[0-4]\d{2}|655[0-2]\d|6553[0-5]))\/(\w+)$/) == null){
      return {valid:false,message:"The value of dbUri should not be null or undefined and its format is invalid"};
    }
    if(settings.apiServerUri == null || typeof(settings.apiServerUri) != "string"){
      return {valid:false,message:"The value of apiServerUri should not be null or undefined and its format is invalid"}; 
    }
    return {valid:true}
  }
  return settingObj;

};