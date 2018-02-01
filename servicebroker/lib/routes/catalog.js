'use strict';

module.exports = function(app, settings, catalog, models) {
  var path = require('path');
  app.get('/v2/catalog', function(req, res) {
    res.status(200).json(catalog)
  });
}