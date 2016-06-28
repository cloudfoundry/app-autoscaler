'use strict';

module.exports = function(app) {
  var path = require('path');
  app.get('/v2/catalog', function(req, res) {
    res.sendFile(path.join(__dirname, '../../config/catalog.json'));
  });
}