'use strict';

module.exports = function(settings) {
  var path = require('path');
  var fs = require('fs');
  var express = require("express");
  var HttpStatus = require("http-status-codes");

  var router = express.Router();
  var infoFilePath = path.resolve(settings.infoFilePath);
  var info = JSON.parse(fs.readFileSync(infoFilePath, 'utf8'));
  router.get("/info", function(req, resp) {
  	resp.status(HttpStatus.OK).json(info);
  });
  return router;
}