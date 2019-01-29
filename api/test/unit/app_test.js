'use strict';

var expect = require('chai').expect;

var fs = require('fs');
var path = require('path');
var NodeCache = require('node-cache');
var ApiServer = require(path.join(__dirname, '../../app.js'));
var configFilePath = path.join(__dirname, '../../config/settings.json');
var credentialCache = new NodeCache();

describe("Fatal configuration error", function() {
  var settings;

  beforeEach(function() {
    settings = require(path.join(__dirname, '../../lib/config/setting.js'))((JSON.parse(
      fs.readFileSync(configFilePath, 'utf8'))));
  });

  context("Wrong db configuration", function() {
    var server;
    afterEach(function(done) {
      server.internalServer.close(function() {
        server.publicServer.close(function(){
          server.healthServer.close(done);
        });
      });
    })
    it("should throw error", function(done) {
      settings.db.uri = "postgres://postgres@127.0.0.1:5432/wrong-db-name";
      server = ApiServer(settings, credentialCache, function(err) {
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
        ApiServer(settings);
      } catch (e) {
        expect(e.message).to.equal("settings.json is invalid");
      }
    });
  });

  context("TLS key file does not exist", function() {
    it("should throw error", function() {
      settings.tls.keyFile = "invalid-file-path";
      try {
        ApiServer(settings);
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
        ApiServer(settings);
      } catch (e) {
        expect(e.message).to.equal("Invalid TLS certificate path: " + settings.tls.certFile);
      }
    });
  });

  context("TLS ca cert file does not exist", function() {
    it("should throw error", function() {
      settings.tls.caCertFile = "invalid-file-path";
      try {
        ApiServer(settings);
      } catch (e) {
        expect(e.message).to.equal("Invalid TLS ca certificate path: " + settings.tls.caCertFile);
      }
    });
  });

  context("Public TLS key file does not exist", function() {
    it("should throw error", function() {
      settings.publicTls.keyFile = "invalid-file-path";
      try {
        ApiServer(settings);
      } catch (e) {
        expect(e.message).to.equal("Invalid public TLS key path: " + settings.publicTls.keyFile);
      }
    });
  });

  context("Public TLS cert file does not exist", function() {
    it("should throw error", function() {
      settings.publicTls.certFile = "invalid-file-path";
      try {
        ApiServer(settings);
      } catch (e) {
        expect(e.message).to.equal("Invalid public TLS certificate path: " + settings.publicTls.certFile);
      }
    });
  });

  context("Public TLS ca cert file does not exist", function() {
    it("should throw error", function() {
      settings.publicTls.caCertFile = "invalid-file-path";
      try {
        ApiServer(settings);
      } catch (e) {
        expect(e.message).to.equal("Invalid public TLS ca certificate path: " + settings.publicTls.caCertFile);
      }
    });
  });

});