'use strict';
var expect = require("chai").expect;
var path = require('path');
var dbConnectionInfo = require(path.join(__dirname, '../../lib/util/dbConnectionInfo.js'));

describe('Database Initialization Test Suite', function() {
  afterEach(function() {
    delete require.cache[require.resolve('../../lib/util/dbConnectionInfo.js')];
  });

  it('Should be OK to return service_instance model', function() {
    var models = require('../../lib/models')();
    expect(models.service_instance).to.be.ok;
  });

  it('Should fail for with invalid DB_URI', function() {
    var correctDBUri = dbConnectionInfo.dbUri;
    dbConnectionInfo.dbUri = 'postgres://postgres@127.0.0.1:5432/invalidDB';
    var models = require('../../lib/models')(function(error) {
      expect(error).to.not.be.null;
    });
    dbConnectionInfo.dbUri = correctDBUri;
  });

});