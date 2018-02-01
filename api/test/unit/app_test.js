'use strict';

var sinon = require('sinon');
var expect = require('chai').expect;

var fs = require('fs');
var path = require('path');
var ApiServer = require(path.join(__dirname, '../../app.js'));
var configFilePath = path.join(__dirname, '../../config/settings.json');
var settings = require(path.join(__dirname, '../../lib/config/setting.js'))((JSON.parse(
  fs.readFileSync(configFilePath, 'utf8'))));

describe("Incorrect database configuration", function() {
  var server;
  var stub;
  beforeEach(function() {
    settings.db.uri = "postgres://postgres@127.0.0.1:5432/wrong-db-name";
    stub = sinon.stub(process, 'exit');
  });

  afterEach(function(done) {
    server.internalServer.close(function(){
      server.publicServer.close(done);
    });
    stub.restore();
  });
  it("server should exit", function(done) {
    server = ApiServer(settings, function(err) {
      if (err) {
        process.exit(1);
      }
      expect(process.exit.isSinonProxy).to.equal(true);
      sinon.assert.called(process.exit);
      sinon.assert.calledWith(process.exit, 1);

      done();
    });

  });
});