'use strict';

var supertest = require("supertest");
var uuid = require('uuid');
var sinon = require('sinon');
var expect = require('chai').expect;

var fs = require('fs');
var path = require('path');
var BrokerServer = require(path.join(__dirname, '../lib/server.js'));
var configFilePath = path.join(__dirname, '../config/settings.json');
var settings = require(path.join(__dirname, '../lib/config/setting.js'))((JSON.parse(
  fs.readFileSync(configFilePath, 'utf8'))));
var catalog = JSON.parse(fs.readFileSync(path.join(__dirname, '../config/catalog.json'), 'utf8'));
var auth = new Buffer(settings.username + ":" + settings.password).toString('base64')
process.env.NODE_TLS_REJECT_UNAUTHORIZED = '0';
describe("Invalid path for RESTful API", function() {
  var server;
  before(function() {
    server = BrokerServer(settings, catalog, function() {});
  });

  after(function(done) {
    server.close(done);
  });


  it("should resturn 404 when path is invalid", function(done) {
    supertest(server)
      .get("/v2/invalidpath")
      .set("Authorization", "Basic " + auth)
      .expect(404, done);
  });

});

describe("Auth for RESTful API", function() {
  var server;
  before(function() {
    server = BrokerServer(settings, catalog, function() {});
  });

  after(function(done) {
    server.close(done);
  });

  it("should return 401 when no auth info provided", function(done) {
    supertest(server)
      .get("/v2/catalog")
      .expect(401, done);
  });

  it("should return 401 when incorrect user/password provided", function(done) {
    supertest(server)
      .get("/v2/catalog")
      .set("Authorization", "Basic " + new Buffer("incorrectuser:incorrectpassword").toString('base64'))
      .expect(401, done);
  });

  it("should return 401 when incorrect user provided", function(done) {
    supertest(server)
      .get("/v2/catalog")
      .set("Authorization", "Basic " + new Buffer("incorrectuser" + ":" + settings.password).toString('base64'))
      .expect(401, done);

  });

  it("should return 401 when incorrect password provided", function(done) {
    supertest(server)
      .get("/v2/catalog")
      .set("Authorization", "Basic " + new Buffer(settings.username + ":" + "incorrectpassword").toString('base64'))
      .expect(401, done);

  });

});

describe("Fatal configuration error", function() {
  var settings;

  beforeEach(function() {
    settings = require(path.join(__dirname, '../lib/config/setting.js'))((JSON.parse(
  fs.readFileSync(configFilePath, 'utf8'))));
  });

  context("Wrong db configuration", function() {
    var server;
    afterEach(function(done) {
      server.close(done);
    })
    it("should throw error", function(done) {
      settings.db.uri = "postgres://postgres@127.0.0.1:5432/wrong-db-name";
      server = BrokerServer(settings, catalog, function(err) {
        expect(err).not.to.be.null;
        expect(err.message).to.equal("database \"wrong-db-name\" does not exist");
        done();
      });
    });
  });

  context("Setting is invalid", function() {
    it("should throw error", function() {
      settings.port = "not-valid-port";
      try {
        BrokerServer(settings, catalog);
      } catch (e) {
        expect(e.message).to.equal("settings.json is invalid");
      }
    });
  });

  context("TLS key file does not exist", function() {
    it("should throw error", function() {
      settings.tls.keyFile = "invalid-file-path";
      try {
        BrokerServer(settings, catalog);
      } catch (e) {
        console.log(e)
        expect(e.message).to.equal("Invalid TLS key path: " + settings.tls.keyFile);
      }
    });
  });

  context("TLS cert file does not exist", function() {
    it("should throw error", function() {
      settings.tls.certFile = "invalid-file-path";
      try {
        BrokerServer(settings, catalog);
      } catch (e) {
        expect(e.message).to.equal("Invalid TLS certificate path: " + settings.tls.certFile);
      }
    });
  });

  context("TLS ca cert file does not exist", function() {
    it("should throw error", function() {
      settings.tls.caCertFile = "invalid-file-path";
      try {
        BrokerServer(settings, catalog);
      } catch (e) {
        expect(e.message).to.equal("Invalid TLS ca certificate path: " + settings.tls.caCertFile);
      }
    });
  });

});