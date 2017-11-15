"use strict";

process.env.NODE_TLS_REJECT_UNAUTHORIZED = "0";
var request = require("supertest");
var expect = require("chai").expect;
var fs = require("fs");
var path = require("path");
var settings = require(path.join(__dirname, "../../../lib/config/setting.js"))((JSON.parse(
  fs.readFileSync(path.join(__dirname, "../../../config/settings.json"), "utf8"))));
var API = require("../../../app.js");
var app;
var publicApp;
var servers;

describe("Routing info", function() {

  before(function() {
    servers = API(path.join(__dirname, "../../../config/settings.json"));
    app = servers.internalServer;
    publicApp = servers.publicServer;
  })
  after(function(done) {
    app.close(function() {
      publicApp.close(done);
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
            expect(result.body).to.deep.equal({
              "name": "autoscaler",
              "build": "001",
              "support": "https://github.com/cloudfoundry-incubator/app-autoscaler",
              "description": "Automatically increase or decrease the number of application instances based on a policy you define."
            });
            done();
          });
      });
    });
  });
});