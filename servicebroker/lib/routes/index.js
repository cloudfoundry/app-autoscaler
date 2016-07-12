var fs = require('fs');
var path = require('path');

module.exports = function(app, settings) {
  fs
    .readdirSync(__dirname)
    .filter(function(file) {
      return file.indexOf('.') !== 0 && file !== 'index.js';
    })
    .forEach(function(file) {
      var name = file.substr(0, file.indexOf('.'));
      require(path.join(__dirname, file))(app, settings);
    });
}