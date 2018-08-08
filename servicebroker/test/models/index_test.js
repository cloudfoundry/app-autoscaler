'use strict';
var expect = require("chai").expect;

var fs = require('fs');
var path = require('path');
var settings = require(path.join(__dirname, '../../lib/config/setting.js'))((JSON.parse(
  fs.readFileSync(path.join(__dirname, '../../config/settings.json'), 'utf8'))));

describe('Database Initialization Test Suite', function() {
  afterEach(function() {
    delete require.cache[require.resolve('../../lib/config/setting.js')];
  });

  it('Should be OK to return service_instance model', function() {
    var models = require('../../lib/models')(settings.db);
    expect(models.service_instance).to.be.ok;
  });

  it('Should fail for with invalid DB_URI', function(done) {
    var correctDBUri = settings.db.uri;
    settings.db.uri = 'postgres://postgres@127.0.0.1:5432/invalidDB';
    var models = require('../../lib/models')(settings.db, function(error) {
      expect(error).not.to.be.null;
      done();
    });
    settings.db.uri = correctDBUri;
  });

});
