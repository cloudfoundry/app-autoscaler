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

  var cleanUpUri = function(uri) {
    if (uri != null) {
       return uri.replace(/\/$/g, "").toLowerCase();
      }
    };
  settingsObj.scheduler.uri = cleanUpUri(settingsObj.scheduler.uri);
  settingsObj.scalingEngine.uri = cleanUpUri(settingsObj.scalingEngine.uri);
  var settings = {
    port: settingsObj.port,
    scheduler: settingsObj.scheduler,
    scalingEngine: settingsObj.scalingEngine,
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
  var isMissing = function(value){
    return typeof(value) === "undefined" || value === null;
  }
  var isNumber = function(value){
    return typeof(value) === "number";
  }
  var isString = function(value){
    return typeof(value) === "string";
  }
  var isObject = function(value){
    return typeof(value) === "object";
  }
  settings.validate = function(){
    if (isMissing(settings.port)){
      return{valid:false, message:"port is required"}
    }
    if (!isNumber(settings.port)) {
      return {valid:false,message:"port must be a number"};
    }
    if (settings.port < 1 || settings.port > 65535) {
      return {valid:false,message:"value of port must between 1 and 65535"};
    }

    if (isMissing(settings.db.maxConnections)){
      return {valid:false,message:"db.maxConnections is required"};
    }
    if (!isNumber(settings.db.maxConnections)) {
      return {valid:false,message:"db.maxConnections must be a number"};
    }
    if (settings.db.maxConnections <= 0) {
      return {valid:false,message:"db.maxConnections must be greater than 0"};
    }
    if (isMissing(settings.db.minConnections)){
      return {valid:false,message:"db.minConnections is required"};
    }
    if (!isNumber(settings.db.minConnections)) {
      return {valid:false,message:"db.minConnections must be a number"};
    }
    if (settings.db.minConnections < 0) {
      return {valid:false,message:"db.minConnections must be greater than or equal to 0"};
    }
    if (isMissing(settings.db.idleTimeout)){
      return {valid:false,message:"db.idleTimeout is required"};
    }
    if (!isNumber(settings.db.idleTimeout)) {
      return {valid:false,message:"db.idleTimeout must be a number"};
    }
    if (settings.db.idleTimeout <= 0) {
      return {valid:false,message:"db.idleTimeout must be greater than 0"};
    }
    if (isMissing(settings.db.uri)) {
      return {valid:false,message:"db.uri is required"};
    }
    if (!isString(settings.db.uri)) {
      return {valid:false,message:"db.uri must be a string"};
    }
    if (!isMissing(settings.tls)){
      if (!isObject(settings.tls)){
        return { valid: false, message: "tls must be an object" };
      } 
      if (isMissing(settings.tls.keyFile)) {
        return { valid: false, message: "tls.keyFile is required" };
      }
      if (!isString(settings.tls.keyFile)) {
        return { valid: false, message: "tls.keyFile must be a string" };
      }
      if (isMissing(settings.tls.certFile)) {
        return { valid: false, message: "tls.certFile is required" };
      }
      if (!isString(settings.tls.certFile)) {
        return { valid: false, message: "tls.certFile must be a string" };
      }
      if (isMissing(settings.tls.caCertFile)) {
        return { valid: false, message: "tls.caCertFile is required" };
      }
      if (!isString(settings.tls.caCertFile)) {
        return { valid: false, message: "tls.caCertFile must be a string" };
      }
    }
    if (isMissing(settings.scheduler)) {
      return { valid: false, message: "scheduler is required" };
    }
    if (isMissing(settings.scheduler.uri)) {
      return { valid: false, message: "scheduler.uri is required" };
    }
    if (!isString(settings.scheduler.uri)) {
      return { valid: false, message: "scheduler.uri must be a string" };
    }
    if (!isMissing(settings.scheduler.tls)){
      if (!isObject(settings.scheduler.tls)) {
        return { valid: false, message: "scheduler.tls must be an object" };
      }
      if (isMissing(settings.scheduler.tls.keyFile)) {
        return { valid: false, message: "scheduler.tls.keyFile is required" };
      }
      if (!isString(settings.scheduler.tls.keyFile)) {
        return { valid: false, message: "scheduler.tls.keyFile must be a string" };
      }
      if (isMissing(settings.scheduler.tls.caCertFile)) {
        return { valid: false, message: "scheduler.tls.caCertFile is required" };
      }
      if (!isString(settings.scheduler.tls.caCertFile)) {
        return { valid: false, message: "scheduler.tls.caCertFile must be a string" };
      }
      if (isMissing(settings.scheduler.tls.certFile)) {
        return { valid: false, message: "scheduler.tls.certFile is required" };
      }
      if (!isString(settings.scheduler.tls.certFile)) {
        return { valid: false, message: "scheduler.tls.certFile must be a string" };
      }
    }

    if (isMissing(settings.scalingEngine)) {
      return { valid: false, message: "scalingEngine is required" };
    }
    if (isMissing(settings.scalingEngine.uri)) {
      return { valid: false, message: "scalingEngine.uri is required" };
    }
    if (!isString(settings.scalingEngine.uri)) {
      return { valid: false, message: "scalingEngine.uri must be a string" };
    }
    if (!isMissing(settings.scalingEngine.tls)){
      if (!isObject(settings.scalingEngine.tls)) {
        return { valid: false, message: "scalingEngine.tls must be an object" };
      }
      if (isMissing(settings.scalingEngine.tls.keyFile)) {
        return { valid: false, message: "scalingEngine.tls.keyFile is required" };
      }
      if (!isString(settings.scalingEngine.tls.keyFile)) {
        return { valid: false, message: "scalingEngine.tls.keyFile must be a string" };
      }
      if (isMissing(settings.scalingEngine.tls.caCertFile)) {
        return { valid: false, message: "scalingEngine.tls.caCertFile is required" };
      }
      if (!isString(settings.scalingEngine.tls.caCertFile)) {
        return { valid: false, message: "scalingEngine.tls.caCertFile must be a string" };
      }
      if (isMissing(settings.scalingEngine.tls.certFile)) {
        return { valid: false, message: "scalingEngine.tls.certFile is required" };
      }
      if (!isString(settings.scalingEngine.tls.certFile)) {
        return { valid: false, message: "scalingEngine.tls.certFile must be a string" };
      }
    }
    
    return {valid:true}
  }

  return settings;
};
