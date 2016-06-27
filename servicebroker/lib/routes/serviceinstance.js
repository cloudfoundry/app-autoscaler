'use strict';

module.exports = function(app) {
  var path = require('path');
  var models = require(path.join(__dirname, '../models'))();

  app.put('/v2/service_instances/:serviceId', function(req, res) {
    var serviceId = req.params.serviceId;
    var orgId = req.body.organization_guid;
    var spaceId = req.body.space_guid

    models.service_instance.findOrCreate({
        serviceId: serviceId,
        orgId: orgId,
        spaceId: spaceId,
        where: {
          serviceId: serviceId,
          orgId: orgId,
          spaceId: spaceId
        }
      })
      .then(function(result) {
        var isNew = result[1];
        if (isNew === true) {
          res.status(201);
        } else {
          res.status(200);
        }
        res.send({ "dashboard_url": "" });
      }).catch(function(err) {
        if (err instanceof models.sequelize.UniqueConstraintError) {
          res.status(409);
        } else {
          res.status(500);
        }
        res.end();
      });

  });
}