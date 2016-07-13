'use strict';
var path = require('path');
var validateUtil = require(path.join(__dirname, '../../lib/util/validateUtil.js'));
module.exports = function(settings) {

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
  return {
    port: settings.port,
    username: settings.username,
    password: settings.password,
    db: db(settings.dbUri),
    apiServerUri: apiServer(settings.apiServerUri)
  };

};