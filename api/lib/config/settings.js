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

  var scheduler = function(schedulerUri) {
    if (schedulerUri != null) {
      return schedulerUri.replace(/\/$/g, "").toLowerCase();
    }
  };

  settings = {
    port: settings.port,
    dbMaxConnectionCount: settings.dbMaxConnectionCount,
    dbMinConnectionCount: settings.dbMinConnectionCount,
    dbMaxIdleTime: settings.dbMaxIdleTime,
    db: db(settings.dbUri),
    schedulerUri: scheduler(settings.schedulerUri)
  };
  settings.validate = function(){
    if(typeof(settings.port) != "number") {
      return {valid:false,message:"The port must be a number"};
    }
    if (settings.port < 0 || settings.port > 65536) {
      return {valid:false,message:"The value of port must between 0 and 65536"};
    }

    if(typeof(settings.dbMaxConnectionCount) != "number") {
      return {valid:false,message:"dbMaxConnectionCount is required"};
    }
    if(settings.dbMaxConnectionCount < 0) {
      return {valid:false,message:"dbMaxConnectionCount must be greater than 0"};
    }
    if(typeof(settings.dbMinConnectionCount) != "number") {
      return {valid:false,message:"dbMinConnectionCount is required"};
    }
    if(settings.dbMinConnectionCount < 0) {
      return {valid:false,message:"dbMinConnectionCount must be greater than 0"};
    }
    if(typeof(settings.dbMaxIdleTime) != "number") {
      return {valid:false,message:"dbMaxIdleTime is required"};
    }
    if(settings.dbMaxIdleTime < 0) {
      return {valid:false,message:"dbMaxIdleTime must be greater than 0"};
    }
    if(typeof(settings.db.uri) != "string") {
      return {valid:false,message:"dbUri is required"};
    }
    if(typeof(settings.schedulerUri) != "string") {
      return {valid:false,message:"schedulerUri is required"};
    }
    return {valid:true}
  }

  return settings;
};
