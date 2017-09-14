'use strict';
var path = require('path');
module.exports = function(settingsObj) {
  var settingObj = {};
  var db = function(dbUri) {
    if (dbUri != null) {
      let uri = dbUri.replace(/\/$/g, "");
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
    if (uri) {
       uri = uri.replace(/\/$/g, "").toLowerCase();
      }
      return uri;
    };
  var addProtocol = function(uri) {
    if(uri && (uri.indexOf("https://") < 0 && uri.indexOf("http://") < 0)){
      uri = "https://" + uri;
    }
    return uri;
  }
  settingsObj.scheduler.uri = addProtocol(cleanUpUri(settingsObj.scheduler.uri));
  settingsObj.scalingEngine.uri = addProtocol(cleanUpUri(settingsObj.scalingEngine.uri));
  settingsObj.metricsCollector.uri = addProtocol(cleanUpUri(settingsObj.metricsCollector.uri));
  var settings = {
    port: settingsObj.port,
    cfApi: addProtocol(cleanUpUri(settingsObj.cfApi)),
    publicPort: settingsObj.publicPort,
    scheduler: settingsObj.scheduler,
    scalingEngine: settingsObj.scalingEngine,
    metricsCollector: settingsObj.metricsCollector,
    tls: settingsObj.tls,
    publicTls: settingsObj.publicTls
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
      return {valid:false,message:"value of port must be between 1 and 65535"};
    }

    if (isMissing(settings.publicPort)){
      return{valid:false, message:"publicPort is required"}
    }
    if (!isNumber(settings.publicPort)) {
      return {valid:false,message:"publicPort must be a number"};
    }
    if (settings.publicPort < 1 || settings.publicPort > 65535) {
      return {valid:false,message:"value of publicPort must be between 1 and 65535"};
    }
    if (settings.port == settings.publicPort){
      return {valid:false,message:"internal api port and public api port should be different"}
    }
    if (isMissing(settings.cfApi)){
      return{valid:false, message:"cfApi is required"}
    }
    if (!isString(settings.cfApi)) {
      return {valid:false,message:"cfApi must be a string"};
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
    if (!isMissing(settings.publicTls)){
      if (!isObject(settings.publicTls)){
        return { valid: false, message: "publicTls must be an object" };
      } 
      if (isMissing(settings.publicTls.keyFile)) {
        return { valid: false, message: "publicTls.keyFile is required" };
      }
      if (!isString(settings.publicTls.keyFile)) {
        return { valid: false, message: "publicTls.keyFile must be a string" };
      }
      if (isMissing(settings.publicTls.certFile)) {
        return { valid: false, message: "publicTls.certFile is required" };
      }
      if (!isString(settings.publicTls.certFile)) {
        return { valid: false, message: "publicTls.certFile must be a string" };
      }
      if (isMissing(settings.publicTls.caCertFile)) {
        return { valid: false, message: "publicTls.caCertFile is required" };
      }
      if (!isString(settings.publicTls.caCertFile)) {
        return { valid: false, message: "publicTls.caCertFile must be a string" };
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

    if (isMissing(settings.metricsCollector)) {
      return { valid: false, message: "metricsCollector is required" };
    }
    if (isMissing(settings.metricsCollector.uri)) {
      return { valid: false, message: "metricsCollector.uri is required" };
    }
    if (!isString(settings.metricsCollector.uri)) {
      return { valid: false, message: "metricsCollector.uri must be a string" };
    }
    if (!isMissing(settings.metricsCollector.tls)){
      if (!isObject(settings.metricsCollector.tls)) {
        return { valid: false, message: "metricsCollector.tls must be an object" };
      }
      if (isMissing(settings.metricsCollector.tls.keyFile)) {
        return { valid: false, message: "metricsCollector.tls.keyFile is required" };
      }
      if (!isString(settings.metricsCollector.tls.keyFile)) {
        return { valid: false, message: "metricsCollector.tls.keyFile must be a string" };
      }
      if (isMissing(settings.metricsCollector.tls.caCertFile)) {
        return { valid: false, message: "metricsCollector.tls.caCertFile is required" };
      }
      if (!isString(settings.metricsCollector.tls.caCertFile)) {
        return { valid: false, message: "metricsCollector.tls.caCertFile must be a string" };
      }
      if (isMissing(settings.metricsCollector.tls.certFile)) {
        return { valid: false, message: "metricsCollector.tls.certFile is required" };
      }
      if (!isString(settings.metricsCollector.tls.certFile)) {
        return { valid: false, message: "metricsCollector.tls.certFile must be a string" };
      }
    }
    
    return {valid:true}
  }

  return settings;
};
