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
    models.binding.count({ where: { appId: appId } }).then(function(count) {
      if (count > 0) {
        res.status(200).json({});
        return;
      } else {
        res.status(404).json({});
        return;
      }
    }).catch(function(err) {
      logger.error("Fail to query binding info: ", { "appId": appId, err: err });
      res.status(500).json({});
    });
  });
}