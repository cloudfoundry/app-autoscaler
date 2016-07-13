module.exports = function(dbUri, callback) {

  var fs = require('fs');
  var path = require('path');
  var Sequelize = require('sequelize');
  var logger = require(path.join(__dirname, '../logger/logger.js'));
  var sequelize = new Sequelize(dbUri, { logging: false });

  sequelize.authenticate()
    .then(function() {
      if (callback) {
        callback();
      }
    })
    .catch(function(error) {
      logger.error('DB Connection failed ', error);
      if (callback) {
        callback(error);
      }
    });
  var db = {};

  fs
    .readdirSync(__dirname)
    .filter(function(file) {
      return file.indexOf('.') !== 0 && file !== 'index.js';
    })
    .forEach(function(file) {
      var model = sequelize.import(path.join(__dirname, file));
      db[model.name] = model;
    });
  Object.keys(db).forEach(function(modelName) {
    if ("associate" in db[modelName]) {
      db[modelName].associate(db);
    }
  });
  db.sequelize = sequelize;
  return db;

}