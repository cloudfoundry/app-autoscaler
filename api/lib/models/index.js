module.exports = function(callback) {

  var fs = require('fs');
  var path = require('path');
  var Sequelize = require('sequelize');
  var logger = require('../log/logger');
  var sequelizeURL = process.env.DB_URI;
  var sequelize = new Sequelize(sequelizeURL);

  sequelize.authenticate()
    .then(function() {
      logger.info('DB Connection has been established successfully');
      if (callback) {
        callback();
      }
    })
  .catch(function(error) {
    logger.error('DB Connection failed ',error);
    if (callback) {
      callback (error);    	
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

  db.sequelize = sequelize;
  return db;

}
