'use strict';
var path = require('path');
module.exports = function(settings) {
  var settingObj = {};
  var db = function(dbUri) {
    if (dbUri != null) {
      let uri = dbUri.replace(/\/$/g, "").toLowerCase();
      let name = uri.slice(uri.lastIndexOf("/") + 1, uri.length);
      let server = uri.slice(0, uri.lastIndexOf("/"));
      return {
        uri: uri,
        name: name,
        server: server
      };
    }
  };

  var apiServer = function(apiServerUri) {
    if (apiServerUri != null) {
      return apiServerUri.replace(/\/$/g, "").toLowerCase();
    }
  };

  settings = {
    port: settings.port,
    username: settings.username,
    password: settings.password,
    db: db(settings.dbUri),
    apiServerUri: apiServer(settings.apiServerUri)
  };

  settings.validate = function(){
    if(typeof(settings.port) != "number") {
      return {valid:false,message:"The port must be a number"};
    }
    if (settings.port < 0 || settings.port > 65536) {
      return {valid:false,message:"The value of port must between 0 and 65536"};
    }

    if(typeof(settings.username) != "string") {
      return {valid:false,message:"username is required"};
    }
    if(typeof(settings.password) != "string") {
      return {valid:false,message:"password is required"};
    }
    if(typeof(settings.db.uri) != "string") {
      return {valid:false,message:"dbUri is required"};
    }
    if(typeof(settings.apiServerUri) != "string") {
      return {valid:false,message:"apiServerUri is required"};
    }
    return {valid:true}
  }

  return settings;
};
