'use strict';

module.exports = function(app, settings,catalog) {
  var path = require('path');
  app.get('/v2/catalog', function(req, res) {
    res.status(200).json(catalog)
  });
}