'use strict';


module.exports = function(app, settings, catalog, models) {
  var path = require('path');
  var fs = require('fs');
  var request = require('request');
  var Sequelize = require('sequelize');
  var logger = require(path.join(__dirname, '../logger/logger.js'));

  var apiServerUtil = require(path.join(__dirname, '../util/apiServerUtil.js'))(settings);
  var messageUtil = require(path.join(__dirname, '../util/messageUtil.js'))(catalog);

  function commitTransaction(transaction, response, statusCode, responseBody) {
    transaction.commit().then(function(res) {
      response.status(statusCode).json(responseBody || {});
    }).catch(function(err) {
      response.status(500).json(responseBody || {});
    });
  }

  function rollbackTransaction(transaction, response, statusCode, responseBody) {
    transaction.rollback().then(function(res) {
      response.status(statusCode).json(responseBody || {});
    }).catch(function(err) {
      response.status(500).json(responseBody || {});
    });
  }

  function generateCredentials(username,password) {
    return {
        credentials: {
            custom_metrics:{
              username: username,
              password: password,
              url: settings.customMetricsUrl
            }
        },
    };
  }

  app.put('/v2/service_instances/:instance_id/service_bindings/:binding_id', function(req, res) {
    var serviceInstanceId = req.params.instance_id;
    var bindingId = req.params.binding_id;
    var appId = req.body.app_guid;
    var policyJSON = req.body.parameters;
    var credentialsObject;

    if (serviceInstanceId == null || serviceInstanceId.trim() === "") {
      logger.error("serviceInstanceId is required");
      res.status(400).json({});
      return;
    }
    if (bindingId == null || bindingId.trim() === "") {
      logger.error("bindingId is required");
      res.status(400).json({});
      return;
    }
    if (appId == null || appId.trim() === "") {
      logger.error("app_guid is required");
      res.status(400).json({});
      return;
    }
    models.sequelize.transaction().then(function(t) {
      models.binding.findAll({ where: { appId: appId } }).then(function(result) {
        var length = result.length;
        if (length === 0) { //no binding , so create one
          models.binding.create({
            bindingId: bindingId,
            appId: appId,
            serviceInstanceId: serviceInstanceId,
            timestamp: new Date().getTime()
          }, { transaction: t }).then(function(result) {
            if (typeof(policyJSON) === "undefined") {
              if (settings.enableCustomMetrics) { // Create Custom Metrics Credentials only if feature is enabled
                logger.info("custom metrics feature enabled, generating credentials");
                apiServerUtil.getCreds(appId,function(credError, credResponse) {
                  if (credError != null) {
                    logger.error("Bind failed: credential generation error", { "error": credError });
                    rollbackTransaction(t, res, 500, {});
                    return;
                  }
                  var parsedResponse = JSON.parse(credResponse.body);
                  var credentialsObject = generateCredentials(parsedResponse.username,parsedResponse.password);
                  logger.info("credentials created/updated successfully",credentialsObject);
                  commitTransaction(t, res, 201, credentialsObject);
                });
              } else {
                logger.info("custom metrics feature not enabled");
                commitTransaction(t, res, 201, {credentials: {}});
              }
              return;
            } else {
              apiServerUtil.attachPolicy(appId, policyJSON, function(attachError, attachResponse) {
                if (attachError != null) {
                  logger.error("Bind failed: attach policy error", { error: attachError });
                  rollbackTransaction(t, res, 500, {});
                  return;
                }
                var attachmentStatusCode = attachResponse.statusCode;
                logger.info("Api Server response", { status_code: attachmentStatusCode, response: attachResponse.body });
                  switch (attachmentStatusCode) {
                    case 200:
                    case 201:
                      if (settings.enableCustomMetrics){ // Create Custom Metrics Credentials only if feature is enabled
                        logger.info("custom metrics feature enabled, generating credentials");
                        apiServerUtil.getCreds(appId,function(credError, credResponse) {
                          if (credError != null) {
                            logger.error("Bind failed: credential generation error", { "error": credError });
                            rollbackTransaction(t, res, 500, {});
                            return;
                          }
                          var parsedResponse = JSON.parse(credResponse.body);
                          var credentialsObject = generateCredentials(parsedResponse.username,parsedResponse.password);
                          logger.info("credentials created/updated successfully");
                          commitTransaction(t, res, attachmentStatusCode, credentialsObject);
                        });
                      } else {
                        logger.info("custom metrics feature not enabled");
                        commitTransaction(t, res, attachmentStatusCode, {credentials: {}});
                      }
                      return;;
                    case 400:
                      rollbackTransaction(t, res, attachmentStatusCode, { "description": attachResponse.body.error });
                      return;
                    default:
                      rollbackTransaction(t, res, 500, {});
                      return;
                    }
              });
            }

          }).catch(function(error) { //catch findorcreate
            logger.error("Bind failed: create error", { error: error });
            if (error instanceof Sequelize.UniqueConstraintError) {
              rollbackTransaction(t, res, 409, {});
              return;
            } else if (error instanceof Sequelize.ForeignKeyConstraintError) {
              rollbackTransaction(t, res, 404, { "description": messageUtil.getMessage("SERVICEINSTANCE_NOT_EXIST", { "serviceInstanceId": serviceInstanceId }) });
              return;
            }
            rollbackTransaction(t, res, 500, {});
          });
        } else if (length > 1) { // an app has been bound to more than one service instance, this error should not exist
          logger.error("Bind failed: duplicate bind", { app_guid: appId });
          res.status(409).json({ "description": messageUtil.getMessage("DUPLICATE_BIND", { "applicationId": appId }) });
          return;
        } else if (length == 1) { // an app has been bound to a service instance
          var bindingRecord = result[0];
          if (bindingRecord.serviceInstanceId === serviceInstanceId) {
            logger.error("Bind failed: app already bound", { app_guid: appId, serviceInstanceId: serviceInstanceId });
            rollbackTransaction(t, res, 409, {});
            return;
          }

          logger.error("Bind failed: duplicate bind", { app_guid: appId });
          rollbackTransaction(t, res, 409, { "description": messageUtil.getMessage("DUPLICATE_BIND", { "applicationId": appId }) });
        }
      });
    });

  });

  app.delete('/v2/service_instances/:instance_id/service_bindings/:binding_id', function(req, res) {
    var serviceInstanceId = req.params.instance_id;
    var bindingId = req.params.binding_id;
    var appId = req.body.app_guid;
    if (serviceInstanceId == null || serviceInstanceId.trim() === "") {
      logger.error("serviceInstanceId is required");
      res.status(400).json({});
      return;
    }
    if (bindingId == null || bindingId.trim() === "") {
      logger.error("bindingId is required");
      res.status(400).json({});
      return;
    }
    models.sequelize.transaction().then(function(t) {
      models.binding.findByPk(bindingId).then(function(result) {
        if (result != null) {
          var appId = result.appId;
          models.binding.destroy({ where: { bindingId: bindingId }, transaction: t }).then(function(count) {
            if (count > 0) {
              apiServerUtil.detachPolicy(appId, function(error, response) {
                if (error == null) {
                  var statusCode = response.statusCode;
                  logger.info("Api Server response", { status_code: statusCode, response: response.body });
                  if (statusCode === 200) {
                    if (settings.enableCustomMetrics) { // Delete Custom Metrics Credentials only if feature is enabled
                      logger.info("Custom Metrics enabled deleting credentials");
                      apiServerUtil.deleteCreds(appId,function(credError, credResponse) {
                        if (credError != null) {
                          logger.error("Unbind failed: credential deletion error", { "error": credError });
                          rollbackTransaction(t, res, 500, {});
                        } else {
                          logger.info("credential deleted successfully");
                          commitTransaction(t, res, statusCode, {});
                        }
                      });
                    } else {
                      commitTransaction(t, res, statusCode, {});
                    }
                    return;
                  } else if (statusCode === 404) {
                    commitTransaction(t, res, 200, {});
                    return;
                  }
                  //for 400,500 and other status, return 500
                  logger.error("Unbind failed: detach policy failed", { status_code: statusCode });
                  rollbackTransaction(t, res, 500, {});
                  return;
                }

                logger.error("Unbind failed: detach policy error", { error: error });
                rollbackTransaction(t, res, 500, {});
              });
              return;
            }
            rollbackTransaction(t, res, 410, {});
          }).catch(function(error1) {
            logger.error("Unbind failed: destroy error", { error: error1 });
            rollbackTransaction(t, res, 500, {});
          });
          return;
        }
        rollbackTransaction(t, res, 410, {});
      }).catch(function(error3) {
        logger.error("Unbind failed: find binding failed", { error: error3 });
        rollbackTransaction(t, res, 500, {});
      });
    }).catch(function(error2) {
      logger.error("Unbind failed: transaction error", { error: error2 });
      res.status(500).json({});
    });
  });
}
