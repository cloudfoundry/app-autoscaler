'use strict';

var path = require('path');
var expect = require('chai').expect;

var dbUri = "postgress://postgres@server:port/dbname";
describe('Util dbConnectionInfo Test Suite', function() {

  it('Should parse dbUri correctly', function() {
    var originUri = dbUri;
    var dbConnectionInfo = require(path.join(__dirname, '../../lib/util/dbConnectionInfo.js'))(dbUri);
    expect(dbConnectionInfo.dbUri).to.equal(dbUri);
    expect(dbConnectionInfo.dbServer).to.equal("postgress://postgres@server:port");
    expect(dbConnectionInfo.dbName).to.equal("dbname");
  });

  it('Should filter the last slash when parse dbUri', function() {
    var originUri = dbUri + "/";
    var dbConnectionInfo = require(path.join(__dirname, '../../lib/util/dbConnectionInfo.js'))(originUri);
    expect(dbConnectionInfo.dbUri).to.equal(dbUri);
    expect(dbConnectionInfo.dbServer).to.equal("postgress://postgres@server:port");
    expect(dbConnectionInfo.dbName).to.equal("dbname");
  });

  it('Should handle the upper case when parse dbUri', function() {
    var originUri = dbUri.toUpperCase();
    var dbConnectionInfo = require(path.join(__dirname, '../../lib/util/dbConnectionInfo.js'))(originUri);
    expect(dbConnectionInfo.dbUri).to.equal(dbUri);
    expect(dbConnectionInfo.dbServer).to.equal("postgress://postgres@server:port");
    expect(dbConnectionInfo.dbName).to.equal("dbname");
  });


});