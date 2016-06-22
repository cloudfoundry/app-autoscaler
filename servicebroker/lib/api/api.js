'use strict';

var fs = require('fs');
var path = require('path');
var async = require('asyncawait/async');
var await = require('asyncawait/await');
var dbConnectionInfo = require(path.join(__dirname, '../../lib/util/dbConnectionInfo.js'));
var ServiceInstanceDao = require(path.join(__dirname, '../../lib/dao/sequelize/serviceInstanceDao.js'));

var catalog = JSON.parse(
  fs.readFileSync(path.join(__dirname, '../../config/catalog.json'), 'utf8'));

var dbUri = dbConnectionInfo.dbConnectionUri;
var serviceInstanceDao = new ServiceInstanceDao(dbUri);

var API = function API() {};
API.prototype.getCatalog = function() {
  return catalog;
};

API.prototype.provisionService = async(function(serviceId, orgId, spaceId) {
  var result = await (serviceInstanceDao.add(serviceId, orgId, spaceId));
  if ('err' in result) {
    var error = result.err.name;
    if (error.indexOf('SequelizeUniqueConstraintError') >= 0) {
      return { code: 409 };
    } else {
      return { code: 500 };
    }
  } else {
    var isNew = result[1];
    if (isNew === true) {
      return { code: 201, dashboard_url: "" };
    } else {
      return { code: 200, dashboard_url: "" };
    }
  }

});

module.exports = API;