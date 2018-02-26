'use strict';


module.exports = function(app, settings) {
  var path = require('path');
  var logger = require(path.join(__dirname, '../logger/logger.js'));
  var models = require(path.join(__dirname, '../models'))(settings.db);


  app.get('/v1/apps/:app_id/service_bindings', function(req, res) {
    var appId = req.params.app_id;
    if (appId == null || appId.trim() === "") {
      logger.error("appId is required");
      res.status(400).json({});
      return;
    }
    models.binding.findAll({ where: { appId: appId } }).then(function(result) {
      var length = result.length;
      if (length === 0) {
        logger.info("Service binding does not exist", { "appId": appId });
        res.status(200).json({ "binding": null });
        return;
      } else {
        var bindingRecord = result[0];
        logger.info("Service binding does not exist", { "appId": appId, "binding": bindingRecord });
        res.status(200).json({ "binding": bindingRecord });
        return;
      }
    }).catch(function(err) {
      logger.error("Fail to query binding info: ", { "appId": appId, "err": err });
      res.status(500).json({});
      return;
    });;
  });
}