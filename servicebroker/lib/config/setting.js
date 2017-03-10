'use strict';
var path = require('path');
module.exports = function(settingsObj) {
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
  var settings = {
    port: settingsObj.port,
    username: settingsObj.username,
    password: settingsObj.password,
    apiServerUri: apiServer(settingsObj.apiServerUri),
    httpRequestTimeout: settingsObj.httpRequestTimeout,
    tls: settingsObj.tls,
    services: settingsObj.services
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

  settings.validate = function() {
    if (typeof(settings.port) != "number") {
      return { valid: false, message: "The port must be a number" };
    }
    if (settings.port < 0 || settings.port > 65536) {
      return { valid: false, message: "The value of port must between 0 and 65536" };
    }

    if (typeof(settings.username) != "string") {
      return { valid: false, message: "username is required" };
    }
    if (typeof(settings.password) != "string") {
      return { valid: false, message: "password is required" };
    }
    if (typeof(settings.db.maxConnections) != "number") {
      return { valid: false, message: "maxConnections is required" };
    }
    if (settings.db.maxConnections < 0) {
      return { valid: false, message: "maxConnections must be greater than 0" };
    }
    if (typeof(settings.db.minConnections) != "number") {
      return { valid: false, message: "minConnections is required" };
    }
    if (settings.db.minConnections < 0) {
      return { valid: false, message: "minConnections must be greater than 0" };
    }
    if (typeof(settings.db.idleTimeout) != "number") {
      return { valid: false, message: "idleTimeout is required" };
    }
    if (settings.db.idleTimeout < 0) {
      return { valid: false, message: "idleTimeout must be greater than 0" };
    }
    if (typeof(settings.db.uri) != "string") {
      return { valid: false, message: "dbUri is required" };
    }
    if (typeof(settings.apiServerUri) != "string") {
      return { valid: false, message: "apiServerUri is required" };
    }
    if (typeof(settings.httpRequestTimeout) != "number") {
      return { valid: false, message: "The httpRequestTimeout must be a number" };
    }
    if (settings.httpRequestTimeout < 0) {
      return { valid: false, message: "The value of httpRequestTimeout must be greater than 0" };
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
    if(!settings.services){
        return { valid: false, message: "catalog services required" };
    }
    if(!Array.isArray(settings.services)){
        return { valid: false, message: "catalog services must be an array " };
    }
    if(settings.services.length<=0){
        return { valid: false, message: "catalog services array should not be empty" };
    }
    for(var i=0;i<settings.services.length;i++){
      if(!settings.services[i].id){
        return {valid: false, message:"service must have an id"}
      }
      if(!settings.services[i].name){
        return {valid: false, message:"service must have a name"}
      }
      if(settings.services[i].bindable){
        if(typeof(settings.services[i].bindable) !== "boolean"){
          return {valid: false, message:"bindable parameter of catalog service should be of boolean type"}
        }
      }
      if(!settings.services[i].plans){
        return {valid: false, message:"service must have plans"}
      }
      if(!Array.isArray(settings.services[i].plans)){
        return {valid: false, message:"service plans must be an array"}
      }
      if(settings.services[i].plans.length<=0){
        return {valid: false, message:"service plans must be a non empty array"}
      }
      for(var j=0;j<settings.services[i].plans.length;j++){
        if(!settings.services[i].plans[j].name){
          return {valid: false, message:"service plan must have a name"}
        }
        if(!settings.services[i].plans[j].id){
          return {valid: false, message:"service plan must have an id"}
        }
      }
    }
    return { valid: true }
  }
  return settings;
};
