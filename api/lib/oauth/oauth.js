'use strict';
module.exports = function(settings) {
  var request = require("request");
  var logger = require('../log/logger');
  var path = require('path');
  var fs = require('fs');
  var HttpStatus = require('http-status-codes');
  var obj = {};
  obj.checkUserAuthorization = function(req, callback) {
    var appId = req.params.app_id;
    if (!appId) {
      logger.error("Failed to get appId");
      callback({ "statusCode": HttpStatus.NotFound, "message": "Failed to get appId" }, null);
      return;
    }
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
          rejectUnauthorized: !settings.skipSSLValidation,
          headers: {
            "Authorization": userToken,
            "Content-Type": "application/json"
          }
        };
        request(options, function(error1, response, body) {
          if (error1) {
            error1.statusCode = HttpStatus.INTERNAL_SERVER_ERROR;
            logger.error("Failed to get space with userId and AppId during permission check", { "appId": appId, "userId": userId, "error": error1 });
            callback(error1, null);
          } else {
            if (response.statusCode == HttpStatus.OK) {
              var totalResults = body.total_results;
              if (totalResults > 0) {
                logger.info("Permission check passed", { "appId": appId, "userId": userId })
                callback(null, true);
              } else {
                logger.info("Permission check not passed", { "appId": appId, "userId": userId });
                callback(null, false);
              }

            } else {
              var errorObj = {
                "statusCode": response.statusCode
              };
              logger.error("Failed to get space with userId and AppId during permission check", { "appId": appId, "userId": userId, "error": errorObj });
              callback(errorObj, null);
            }
          }
        });
      }


    });

  }

  function getCloudControllerEndpoint() {
    return settings.cfApi;;
  };

  function getCloudFoundryInfo(callback) {
    var ccEndpoint = getCloudControllerEndpoint();
    var options = {
      url: ccEndpoint + "/v2/info",
      method: "GET",
      json: true,
      timeout: 10000,
      rejectUnauthorized: !settings.skipSSLValidation,
    };
    request(options, function(error, response, body) {
      if (error) {
        error.statusCode = HttpStatus.INTERNAL_SERVER_ERROR;
        logger.error("Failed to get Cloud Foundry API information", { "info url": options.url, "error": error });
        callback(error, null);
      } else if (response.statusCode === HttpStatus.OK) {
        logger.info("Get Cloud Foundry API information successfully", { "info url": options.url });
        callback(null, { "statusCode": HttpStatus.OK, "body": body });
      } else {
        var errorObj = {
          "statusCode": response.statusCode
        };
        logger.error("Failed to get Cloud Foundry API information", { "info url": options.url, "error": errorObj });
        callback(errorObj, null);
      }
    });
  }


  function requestUserInfoFromUAA(req, callback) {
    var userToken = req.header("Authorization");
    var options = {
      url: obj.tokenEndpoint + "/userinfo",
      method: "GET",
      json: true,
      timeout: 10000,
      rejectUnauthorized: !settings.skipSSLValidation,
      headers: {
        "Authorization": userToken,
        "Content-Type": "application/json"
      }
    };
    request(options, function(error, response, body) {
      if (error) {
        logger.error("Failed to get user info from UAA", { "userToken": userToken, "http-options": options, "error": error });
        error.statusCode = HttpStatus.INTERNAL_SERVER_ERROR;
        callback(error, null);
      } else {
        if (response.statusCode == HttpStatus.OK) {
          callback(null, body.user_id);
        } else {
          var errorObj = {
            "statusCode": response.statusCode
          };
          logger.error("Failed to get user info from UAA", { "userToken": userToken, "http-options": options, "error": errorObj });
          callback(errorObj, null);
        }
      }
    });
  }

  function getUserInfo(req, callback) {
    if (obj.tokenEndpoint) {
      requestUserInfoFromUAA(req, callback);
    } else {
      getCloudFoundryInfo(function(error, responseBody) {
        if (error) {
          callback(error, null);
        } else {
          obj.tokenEndpoint = responseBody.body.token_endpoint;
          requestUserInfoFromUAA(req, callback);
        }
      });
    }

  }

  return obj;
}