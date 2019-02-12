"use strict";

var request = require("supertest");
var expect = require("chai").expect;
var fs = require("fs");
var path = require("path");
var settings = require(path.join(__dirname, '../../../lib/config/setting.js'))((JSON.parse(
  fs.readFileSync(path.join(__dirname, '../../../config/settings.json'), 'utf8'))));
var relativePath = path.relative(process.cwd(), path.join(__dirname, "../../../../test-certs"));
var testSetting = require(path.join(__dirname, '../test.helper.js'))(relativePath,settings);
var API = require("../../../app.js");
var app;
var publicApp;
var healthApp;
var servers;
var info = JSON.parse(fs.readFileSync(path.join(__dirname, '../../../config/info.json'), 'utf8'));
describe("Routing info", function() {

  before(function() {
    servers = API(testSetting, function(){});
    app = servers.internalServer;
    publicApp = servers.publicServer;
    healthApp = servers.healthServer;
  })
  after(function(done) {
    app.close(function() {
      publicApp.close(function(){
        healthApp.close(done);
      });
    });
  })

  describe("get public apiserver info", function() {
    context("info", function() {

      it("should return info", function(done) {
        request(publicApp)
          .get("/v1/info")
          .end(function(error, result) {
            expect(error).to.equal(null);
            console.log(result.body)
            expect(result.statusCode).to.equal(200);
            expect(JSON.stringify(result.body)).to.be.equal(JSON.stringify(info));
            done();
          });
      });
    });
  });
});