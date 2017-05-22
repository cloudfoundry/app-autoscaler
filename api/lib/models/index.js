module.exports = function(dbSettings, callback) {

  var fs = require('fs');
  var path = require('path');
  var Sequelize = require('sequelize');
  var logger = require('../log/logger');

  const DEFAULT_DB_MAX_CONNECTIONS = 10;
  const DEFAULT_DB_MIN_CONNECTIONS = 0;
  const DEFAULT_DB_MAX_IDLETIME = 1000;
  var db = {};
  //init policyDb
  var policyDbSequelize = new Sequelize(dbSettings.policyDb.uri, {
    pool: {
      max: dbSettings.maxConnections || DEFAULT_DB_MAX_CONNECTIONS,
      min: dbSettings.minConnections || DEFAULT_DB_MIN_CONNECTIONS,
      idle: dbSettings.idleTimeout || DEFAULT_DB_MAX_IDLETIME
    }
  });

  policyDbSequelize.authenticate()
    .then(function() {
      logger.info('PolicyDB Connection has been established successfully');
      if (callback) {
        callback();
      }
    })
    .catch(function(error) {
      logger.error('PolicyDB Connection failed ',{ 'error':error });
      if (callback) {
        callback(error);
      }
    });
  

  fs
    .readdirSync(__dirname + '/policy')
    .filter(function(file) {
      return file.indexOf('.') !== 0 && file !== 'index.js';
    })
    .forEach(function(file) {
      var model = policyDbSequelize.import(path.join(__dirname + '/policy', file));
      db[model.name] = model;
    });

    //init scalingEngineDb
  var scalingEngineDbSequelize = new Sequelize(dbSettings.scalingEngineDb.uri, {
    pool: {
      max: dbSettings.maxConnections || DEFAULT_DB_MAX_CONNECTIONS,
      min: dbSettings.minConnections || DEFAULT_DB_MIN_CONNECTIONS,
      idle: dbSettings.idleTimeout || DEFAULT_DB_MAX_IDLETIME
    }
  });

  scalingEngineDbSequelize.authenticate()
    .then(function() {
      logger.info('ScalingEngineDB Connection has been established successfully');
      if (callback) {
        callback();
      }
    })
    .catch(function(error) {
      logger.error('ScalingEngineDB Connection failed ',{ 'error':error });
      if (callback) {
        callback(error);
      }
    });

  fs
    .readdirSync(__dirname + '/scalingHistory')
    .filter(function(file) {
      return file.indexOf('.') !== 0 && file !== 'index.js';
    })
    .forEach(function(file) {
      var model = scalingEngineDbSequelize.import(path.join(__dirname + '/scalingHistory', file));
      db[model.name] = model;
    });

  db.policyDbSequelize = policyDbSequelize;
  db.scalingEngineDbSequelize = scalingEngineDbSequelize;
  return db;

}
