'use strict';
var path = require('path');
var fs = require('fs');

module.exports = function(originDBUri) {
  var dbUri = originDBUri.replace(/\/$/g, "").toLowerCase();
  var dbName = dbUri.slice(dbUri.lastIndexOf("/") + 1, dbUri.length);
  var dbServer = dbUri.slice(0, dbUri.lastIndexOf("/"));

  return {
    "dbUri": dbUri,
    "dbName": dbName,
    "dbServer": dbServer
  }

}