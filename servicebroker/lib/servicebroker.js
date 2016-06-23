'use strict';

var fs = require('fs');
var path = require('path');
var async = require('asyncawait/async');
var await = require('asyncawait/await');
var model = require(path.join(__dirname, './model'))();

var ServiceBroker = function ServiceBroker() {};
ServiceBroker.prototype.getCatalog = function() {
  var catalog = JSON.parse(
    fs.readFileSync(path.join(__dirname, '../config/catalog.json'), 'utf8'));
  return catalog;
};

ServiceBroker.prototype.provisionService = async(function(serviceId, orgId, spaceId) {
  try {
    var result = await (model.service_instance.findOrCreate({
      serviceId: serviceId,
      orgId: orgId,
      spaceId: spaceId,
      where: {
        serviceId: serviceId,
        orgId: orgId,
        spaceId: spaceId
      }
    }));

    var isNew = result[1];
    if (isNew === true) {
          return { code: 201, dashboard_url: "" };
    } else {
          return { code: 200, dashboard_url: "" };
    }
  } catch (err) {   
      if (err.name.indexOf('SequelizeUniqueConstraintError') >= 0) {
        return { code: 409 };
      } else {
        return { code: 500 };
      } 
  }

});

module.exports = ServiceBroker;