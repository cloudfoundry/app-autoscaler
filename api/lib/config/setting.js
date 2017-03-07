'use strict';
var path = require('path');
module.exports = function(settingsObj) {
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
  settingsObj.scheduler.uri = scheduler(settingsObj.scheduler.uri);
  var settings = {
    port: settingsObj.port,
    username: settingsObj.username,
    password: settingsObj.password,
    scheduler: settingsObj.scheduler,
    tls: settingsObj.tls
  };
  if (settingsObj.db) {
    var dbObj = db(settingsObj.db.uri);
    settings.db = {
        maxConnections: settingsObj.db.maxConnections,
        minConnections: settingsObj.db.minConnections,
        idleTimeout: settingsObj.db.idleTimeout,
        uri: dbObj.uri,
        name: dbObj.name,
        server: dbObj.server
      }

  }
  settings.validate = function(){
    if(typeof(settings.port) != "number") {
      return {valid:false,message:"The port must be a number"};
    }
    if (settings.port < 0 || settings.port > 65536) {
      return {valid:false,message:"The value of port must between 0 and 65536"};
    }

    if(typeof(settings.db.maxConnections) != "number") {
      return {valid:false,message:"maxConnections is required"};
    }
    if(settings.db.maxConnections < 0) {
      return {valid:false,message:"maxConnections must be greater than 0"};
    }
    if(typeof(settings.db.minConnections) != "number") {
      return {valid:false,message:"minConnections is required"};
    }
    if(settings.db.minConnections < 0) {
      return {valid:false,message:"minConnections must be greater than 0"};
    }
    if(typeof(settings.db.idleTimeout) != "number") {
      return {valid:false,message:"idleTimeout is required"};
    }
    if(settings.db.idleTimeout < 0) {
      return {valid:false,message:"idleTimeout must be greater than 0"};
    }
    if(typeof(settings.db.uri) != "string") {
      return {valid:false,message:"dbUri is required"};
    }
    if (!settings.tls) {
      return { valid: false, message: "tls is required" };
    }
    if(typeof(settings.tls) != "object"){
      return { valid: false, message: "tls must be an object" };
    } 
    if (typeof(settings.tls.keyFile) != "string") {
      return { valid: false, message: "tls.keyFile is required" };
    }
    if (typeof(settings.tls.certFile) != "string") {
      return { valid: false, message: "tls.certFile is required" };
    }
    if (typeof(settings.tls.caCertFile) != "string") {
      return { valid: false, message: "tls.caCertFile is required" };
    }
    if (!settings.scheduler) {
      return { valid: false, message: "scheduler details required" };
    }
    if (typeof (settings.scheduler.tls) != "object") {
      return { valid: false, message: "scheduler.tls should be an object" };
    }
    if (!settings.scheduler.uri) {
      return { valid: false, message: "scheduler uri is required" };
    }
    if (!settings.scheduler.tls) {
      return { valid: false, message: "scheduler tls is required" };
    }
    if (!settings.scheduler.tls.keyFile) {
      return { valid: false, message: "scheduler.tls.keyFile is required" };
    }
    if (!settings.scheduler.tls.caCertFile) {
      return { valid: false, message: "scheduler.tls.caCertFile is required" };
    }
    if (!settings.scheduler.tls.certFile) {
      return { valid: false, message: "scheduler.tls.certFile is required" };
    }
    return {valid:true}
  }

  return settings;
};
