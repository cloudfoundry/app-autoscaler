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
    httpRequestTimeout: settingsObj.httpRequestTimeout
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
    return { valid: true }
  }

  return settings;
};
