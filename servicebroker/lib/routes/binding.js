'use strict';


module.exports = function(app, settings) {
  var path = require('path');
  var fs = require('fs');
  var request = require('request');
  var logger = require(path.join(__dirname, '../logger/logger.js'));
  var models = require(path.join(__dirname, '../models'))(settings.db);

  var apiServerUtil = require(path.join(__dirname, '../util/apiServerUtil.js'))(settings.apiServerUri);
  var messageUtil = require(path.join(__dirname, '../util/messageUtil.js'))();
  app.put('/v2/service_instances/:instance_id/service_bindings/:binding_id', function(req, res) {
    var serviceInstanceId = req.params.instance_id;
    var bindingId = req.params.binding_id;
    var appId = req.body.app_guid;
    var policyJSON = req.body.parameters;
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
    if (typeof(policyJSON) === "undefined") {
      res.status(400).json({ "description": messageUtil.getMessage("POLICY_REQUIRED") });
    } else {
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
              apiServerUtil.attachPolicy(appId, policyJSON, function(error, response) {
                if (error == null) {
                  var statusCode = response.statusCode;
                  logger.info("Api Server response", { status_code: statusCode, response: response.body });
                  if (statusCode === 200 || statusCode === 201) {
                    t.commit();
                    res.status(statusCode).json({});
                  } else {
                    t.rollback();
                    if (statusCode === 400 || statusCode === 500) {
                      res.status(statusCode).json({});
                    } else {
                      res.status(500).json({});
                    }
                  }
                } else {
                  logger.error("Bind failed: attach policy error", { error: error });
                  res.status(500).json({});
                  t.rollback();
                }
              });
            }).catch(function(error1) { //catch findorcreate
              logger.error("Bind failed: create error", { error: error1 });
              t.rollback();
              if (error1 instanceof models.sequelize.UniqueConstraintError) {
                res.status(409).json({});
              } else if (error1 instanceof models.sequelize.ForeignKeyConstraintError) {
                res.status(404).json({ "description": messageUtil.getMessage("SERVICEINSTANCE_NOT_EXIST", { "serviceInstanceId": serviceInstanceId }) });
              } else {
                res.status(500).json({});
              }
            });
          } else if (length > 1) { // an app has been bound to more than one service instance, this error should not exist
            logger.error("Bind failed: duplicate bind", { app_guid: appId });
            res.status(409).json({ "description": messageUtil.getMessage("DUPLICATE_BIND", { "applicationId": appId }) });
          } else if (length == 1) { // an app has been bound to a service instance
            var bindingRecord = result[0];
            if (bindingRecord.serviceInstanceId === serviceInstanceId) {
              logger.error("Bind failed: app already bound", { app_guid: appId, serviceInstanceId: serviceInstanceId });
              res.status(409).json({});
            } else {
              logger.error("Bind failed: duplicate bind", { app_guid: appId });
              res.status(409).json({ "description": messageUtil.getMessage("DUPLICATE_BIND", { "applicationId": appId }) });
            }
          }
        });
      }).catch(function(error2) { //catch transaction
        logger.error("Bind failed: transaction error", { error: error2 });
        t.rollback();
        res.status(500).json({});
      });
    }

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
      models.binding.findById(bindingId).then(function(result) {
        if (result != null) {
          var appId = result.appId;
          models.binding.destroy({ where: { bindingId: bindingId }, transaction: t }).then(function(count) {
            if (count > 0) {
              apiServerUtil.detachPolicy(appId, function(error, response) {
                if (error == null) {
                  var statusCode = response.statusCode;
                  logger.info("Api Server response", { status_code: statusCode, response: response.body });
                  if (statusCode === 200) {
                    t.commit();
                    res.status(statusCode).json({});
                  } else if (statusCode === 404) {
                    t.commit();
                    res.status(200).json({});
                  } else { //for 400,500 and other status, return 500
                    logger.error("Unbind failed: detach policy failed", { status_code: statusCode });
                    t.rollback();
                    res.status(500).json({});
                  }
                } else {
                  t.rollback();
                  logger.error("Bind failed: detach policy error", { error: error });
                  res.status(500).json({});
                }

              });
            } else {
              t.commit();
              res.status(410).json({});
            }
          }).catch(function(error1) {
            logger.error("Unbind failed: destroy error", { error: error1 });
            t.rollback();
            res.status(500).json({});
          });
        } else {
          t.commit();
          res.status(410).json({});
        }
      }).catch(function(error3) {
        logger.error("Unbind failed: find binding failed", { error: error3 });
        t.rollback();
        res.status(500).json({});
      });
    }).catch(function(error2) {
      logger.error("Unbind failed: transaction error", { error: error2 });
      t.rollback();
      res.status(500).json({});
    });
  });
}
