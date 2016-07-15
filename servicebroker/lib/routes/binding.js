'use strict';


module.exports = function(app, settings) {
  var path = require('path');
  var fs = require('fs');
  var request = require('request');
  var logger = require(path.join(__dirname, '../logger/logger.js'));
  var models = require(path.join(__dirname, '../models'))(settings.db.uri);

  var apiServerUtil = require(path.join(__dirname, '../util/apiServerUtil.js'))(settings.apiServerUri);
  var messageUtil = require(path.join(__dirname, '../util/messageUtil.js'))();
  app.put('/v2/service_instances/:instance_id/service_bindings/:binding_id', function(req, res) {
    var serviceInstanceId = req.params.instance_id;
    var bindingId = req.params.binding_id;
    var appId = req.body.app_guid;
    var policyJSON = req.body.parameters;
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
              timestamp: new Date().getTime(),
              transaction: t
            }).then(function(result) {
              apiServerUtil.attachPolicy(appId, policyJSON, function(error, response) {
                var statusCode = response.statusCode;
                logger.info("Api server response, status code=>" + statusCode + ", response=>" + JSON.stringify(response.body));
                if (!error && (statusCode == 200 || statusCode == 201)) {
                  t.commit();
                  res.status(statusCode).json({});

                } else {
                  logger.error("Fail to bind service when call api server, error=>" + error);
                  t.rollback();
                  if (statusCode === 400 || statusCode === 500) {
                    res.status(statusCode).json({});
                  } else {
                    res.status(500).json({});
                  }

                }
              });

            }).catch(function(error1) { //catch findorcreate
              logger.error("Fail to bind service when add data to database, error=>" + error1);
              t.rollback();
              if (error1 instanceof models.sequelize.UniqueConstraintError) {
                res.status(409).json({});
              } else if (error1 instanceof models.sequelize.ForeignKeyConstraintError) {
                res.status(404).json({ "description": messageUtil.getMessage("SERVICEINSTANCE_NOT_EXIST", { "serviceInstanceId": serviceInstanceId }) });
              } else {
                logger.error("Fail to handle request: " + JSON.stringify(req) + " with ERROR: " + error1);
                res.status(500).json({});
              }

            });
          } else if (length > 1) { // an app has been bound to more than one service instance, this error should not exist
            logger.error("");
            res.status(499).json({ "description": messageUtil.getMessage("DUPLICATE_BIND", { "applicationId": appId }) });
          } else if (length == 1) { // an app has been bound to a service instance
            var bindingRecord = result[0];
            if (bindingRecord.serviceInstanceId === serviceInstanceId) {
              res.status(409).json({});
            } else {
              res.status(499).json({ "description": messageUtil.getMessage("DUPLICATE_BIND", { "applicationId": appId }) });
            }
          }
        });
      }).catch(function(error2) { //catch transaction
        logger.error("Fail to bind service when handle transaction, error=>" + error2);
        t.rollback();
        res.status(500).json({});
      });
    }

  });




}