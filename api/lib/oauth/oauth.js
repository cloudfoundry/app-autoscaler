'use strict';
module.exports = function(configFilePath) {
  var request = require("request");
  var logger = require('../log/logger');
  var path = require('path');
  var fs = require('fs');
  var settings = require(path.join(__dirname, './../config/setting.js'))((JSON.parse(
    fs.readFileSync(configFilePath, 'utf8'))));
  var HttpStatus = require('http-status-codes');
  var obj = {};
  var prefix = "/v1/apps/";
  obj.checkUserAuthorization = function(req, callback) {
    var reqPath = req.path;
    var startIndex = reqPath.indexOf(prefix) + prefix.length;
    var endIndex = reqPath.indexOf("/", startIndex);
    var appId = reqPath.substring(startIndex, endIndex);
    var ccEndpoint = getCloudControllerEndpoint(req);
    var userToken = req.header("Authorization");
    if (!userToken) {
      logger.info("User token is not provided");
      callback(null, false);
      return;
    }
    getUserInfo(req, function(error, userId) {
      if (error) {
        callback(error, null);
      } else {
        var options = {
          url: ccEndpoint + "/v2/users/" + userId + "/spaces?q=app_guid:" + appId + "&q=developer_guid:" + userId,
          method: "GET",
          json: true,
          timeout: 10000,
          headers: {
            "Authorization": userToken,
            "Content-Type": "application/json"
          }
        };
        request(options, function(error1, response, body) {
          if (error1) {
            logger.error("Error occurred during get space by userId and appId", { "appId": appId, "userId": userId, "error": error1 });
            callback(error1, null);
          } else {
            if (response.statusCode == HttpStatus.OK) {
              var totalResults = body.total_results;
              if (totalResults > 0) {
                logger.info("Check user authorization successfully", { "appId": appId, "userId": userId })
                callback(null, true);
              } else {
                logger.info("Check user authorization failed", { "appId": appId, "userId": userId });
                callback(null, false);
              }

            } else {
              var errorObj = {
                "statusCode": response.statusCode
              };
              logger.error("Error occurred during get space by userId and appId", { "appId": appId, "userId": userId, "error": errorObj });
              callback(errorObj, null);
            }
          }
        });
      }


    });

  }

  function getCloudControllerEndpoint(req) {
    return settings.cfApi;;
  };

  function getCloudFoundryInfo(req, callback) {
    var ccEndpoint = getCloudControllerEndpoint(req);
    var options = {
      url: ccEndpoint + "/v2/info",
      method: "GET",
      json: true,
      timeout: 10000,

    };
    request(options, function(error, response, body) {
      if (error) {
        logger.error("Error occurred during get cloud foundry information ", { "info url": options.url, "error": error });
        error.statusCode = HttpStatus.INTERNAL_SERVER_ERROR;
        callback(error, null);
      } else if (response.statusCode === HttpStatus.OK) {
        logger.info("Get cound foundry information successfully", { "info url": options.url });
        callback(null, { "statusCode": HttpStatus.OK, "body": body });
      } else {
        var errorObj = {
          "statusCode": response.statusCode
        };
        logger.error("Error occurred during get cloud foundry information ", { "info url": options.url, "error": errorObj });
        callback(errorObj, null);
      }
    });
  }



  function getUserInfo(req, callback) {
    getCloudFoundryInfo(req, function(error, responseBody) {
      if (error) {
        callback(error, null);
      } else {
        var userToken = req.header("Authorization");
        var authorizationEndpoint = responseBody.body.authorization_endpoint;
        var options = {
          url: authorizationEndpoint + "/userinfo",
          method: "GET",
          json: true,
          timeout: 10000,
          headers: {
            "Authorization": userToken,
            "Content-Type": "application/json"
          }
        };
        request(options, function(error1, response, body) {
          if (error1) {
            logger.error("Error occurred during get userinfo", { "userToken": userToken, "http-options": options, "error": error1 });
            error1.statusCode = HttpStatus.INTERNAL_SERVER_ERROR;
            callback(error1, null);
          } else {
            if (response.statusCode == HttpStatus.OK) {
              callback(null, body.user_id);
            } else {
              var errorObj = {
                "statusCode": response.statusCode
              };
              logger.error("Error occurred during get userinfo", { "userToken": userToken, "http-options": options, "error": errorObj });
              callback(errorObj, null);
            }
          }
        });
      }
    });
  }

  return obj;
}