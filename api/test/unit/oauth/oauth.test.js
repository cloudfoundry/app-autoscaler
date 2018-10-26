"use strict";

process.env.NODE_TLS_REJECT_UNAUTHORIZED = "0";
var request = require("supertest");
var expect = require("chai").expect;
var fs = require("fs");
var path = require("path");
var HttpStatus = require('http-status-codes');

var configFilePath = path.join(__dirname, '../../../config/settings.json');
var settings = require(path.join(__dirname, '../../../lib/config/setting.js'))((JSON.parse(
  fs.readFileSync(configFilePath, 'utf8'))));
var logger = require("../../../lib/log/logger");
var oauth;
var nock = require("nock");
var theAppId = "the-app-guid";
var theUserId = "the-user-id";
var req = {};
describe("Oauth", function() {
  this.timeout(10000 + 1000); //timeout is 10s and here wait 11s
  
  beforeEach(function() {
    nock.cleanAll();
    oauth = require("../../../lib/oauth/oauth")(settings);
    req = {
      protocol: "https",
      host: "app.bosh-lite.com",
      path: "/v1/apps/" + theAppId + "/routes",
      params: {
        "app_id": theAppId
      },
      header: function(headerName) {
        return "fake-type fake-token";
      }
    };
  });
  describe("oauth", function() {
    context("appId is not provided", function() {
      beforeEach(function() {
        req.path = "/v1/apps//routes";
        req.params = {"app_id": "" };
      });
      it("should return error", function(done) {
        oauth.checkUserAuthorization(req, function(error, result) {
          expect(result).to.equal(null);
          expect(error).to.deep.equal({ statusCode: HttpStatus.NotFound, message: "Failed to get appId" });
          done();
        });
      });
    });
    context("user token is not provided", function() {
      beforeEach(function() {
        req.header = function(headerName) {
          return null;
        }
      });
      it("should return false", function(done) {
        oauth.checkUserAuthorization(req, function(error, result) {
          expect(result).to.equal(false);
          expect(error).to.equal(null);
          done();
        });
      });
    });
    context("Cloud Controller API endpoint is not available", function() {
      it("should return error", function(done) {
        oauth.checkUserAuthorization(req, function(error, result) {
          expect(result).to.equal(null);
          expect(error.statusCode).to.equal(HttpStatus.INTERNAL_SERVER_ERROR);
          done();
        });
      });
    });

    context("Cloud Controller /v2/info returns 503", function() {
      beforeEach(function() {
        nock("https://api.bosh-lite.com")
          .get("/v2/info")
          .reply(HttpStatus.SERVICE_UNAVAILABLE, { "message": "SERVICE_UNAVAILABLE" });
      });
      it("should return error with status code of Cloud Controller /v2/info", function(done) {
        oauth.checkUserAuthorization(req, function(error, result) {
          expect(result).to.equal(null);
          expect(error).to.deep.equal({
            "statusCode": HttpStatus.SERVICE_UNAVAILABLE
          });
          done();
        });
      });
    });

    context("Authorization endpoint is not available", function() {
      beforeEach(function() {
        nock("https://api.bosh-lite.com")
          .get("/v2/info")
          .reply(HttpStatus.OK, { "token_endpoint": "https://uaa.bosh-lite.com" });
      });
      it("should return error", function(done) {
        oauth.checkUserAuthorization(req, function(error, result) {
          expect(result).to.equal(null);
          expect(error.statusCode).to.equal(HttpStatus.INTERNAL_SERVER_ERROR);
          done();
        });
      });
    });

    context("Token check endpoint returns 401", function() {
      beforeEach(function() {
        nock("https://api.bosh-lite.com")
          .get("/v2/info")
          .reply(HttpStatus.OK, { "token_endpoint": "https://uaa.bosh-lite.com" });

        nock("https://uaa.bosh-lite.com")
          .post("/check_token?token=fake-token")
          .reply(HttpStatus.UNAUTHORIZED, { "message": "UNAUTHORIZED" });

      });
      it("should return error", function(done) {
        oauth.checkUserAuthorization(req, function(error, result) {
          expect(result).to.equal(null);
          expect(error).to.deep.equal({
            "statusCode": HttpStatus.UNAUTHORIZED
          });
          done();
        });
      });
    });
    context("Token check endpoint returns 503", function() {
      beforeEach(function() {
        nock("https://api.bosh-lite.com")
          .get("/v2/info")
          .reply(HttpStatus.OK, { "token_endpoint": "https://uaa.bosh-lite.com" });

        nock("https://uaa.bosh-lite.com")
          .post("/check_token?token=fake-token")
          .reply(HttpStatus.SERVICE_UNAVAILABLE, { "message": "SERVICE_UNAVAILABLE" });

      });
      it("should return error with status code of Authorization endpoint", function(done) {
        oauth.checkUserAuthorization(req, function(error, result) {
          expect(result).to.equal(null);
          expect(error).to.deep.equal({
            "statusCode": HttpStatus.SERVICE_UNAVAILABLE
          });
          done();
        });
      });
    });

    context("User is a cloud controller admin", function() {
      beforeEach(function() {
        nock("https://api.bosh-lite.com")
          .get("/v2/info")
          .reply(HttpStatus.OK, { "token_endpoint": "https://uaa.bosh-lite.com" });

        nock("https://uaa.bosh-lite.com")
          .post("/check_token?token=fake-token")
          .reply(HttpStatus.OK, { "scope": ["openid","routing.router_groups.write","network.write","scim.read","cloud_controller.admin","uaa.user","routing.router_groups.read","cloud_controller.read","password.write","cloud_controller.write","network.admin","doppler.firehose","scim.write"] });

      });
      it("should return true", function(done) {
        oauth.checkUserAuthorization(req, function(error, result) {
          expect(result).to.equal(true);
          expect(error).to.deep.equal(null);
          done();
        });
      });
    });

    context("User is not a cloud controller admin", function() {
      beforeEach(function() {
        nock("https://api.bosh-lite.com")
          .get("/v2/info")
          .reply(HttpStatus.OK, { "token_endpoint": "https://uaa.bosh-lite.com" });

        nock("https://uaa.bosh-lite.com")
          .post("/check_token?token=fake-token")
          .reply(HttpStatus.OK, { "scope": ["cloud_controller.read","password.write","cloud_controller.write","openid","network.admin","network.write","uaa.user"] });

      });

      context("Authorization endpoint returns 401", function() {
        beforeEach(function() {
          nock("https://api.bosh-lite.com")
            .get("/v2/info")
            .reply(HttpStatus.OK, { "token_endpoint": "https://uaa.bosh-lite.com" });
  
          nock("https://uaa.bosh-lite.com")
            .get("/userinfo")
            .reply(HttpStatus.UNAUTHORIZED, { "message": "UNAUTHORIZED" });
  
        });
        it("should return error", function(done) {
          oauth.checkUserAuthorization(req, function(error, result) {
            expect(result).to.equal(null);
            expect(error).to.deep.equal({
              "statusCode": HttpStatus.UNAUTHORIZED
            });
            done();
          });
        });
      });
      context("Authorization endpoint returns 503", function() {
        beforeEach(function() {
          nock("https://api.bosh-lite.com")
            .get("/v2/info")
            .reply(HttpStatus.OK, { "token_endpoint": "https://uaa.bosh-lite.com" });
  
          nock("https://uaa.bosh-lite.com")
            .get("/userinfo")
            .reply(HttpStatus.SERVICE_UNAVAILABLE, { "message": "SERVICE_UNAVAILABLE" });
  
        });
        it("should return error with status code of Authorization endpoint", function(done) {
          oauth.checkUserAuthorization(req, function(error, result) {
            expect(result).to.equal(null);
            expect(error).to.deep.equal({
              "statusCode": HttpStatus.SERVICE_UNAVAILABLE
            });
            done();
          });
        });
      });
  
      context("Cloud Controller user spaces API is not available", function() {
        beforeEach(function() {
          nock("https://api.bosh-lite.com/v2/info")
            .get("*")
            .reply(HttpStatus.OK, { "token_endpoint": "https://uaa.bosh-lite.com" });
  
          nock("https://uaa.bosh-lite.com")
            .get("/userinfo")
            .reply(HttpStatus.OK, { "user_id": theUserId });
        });
        it("should return error", function(done) {
          oauth.checkUserAuthorization(req, function(error, result) {
            expect(result).to.equal(null);
            expect(error.statusCode).to.equal(HttpStatus.INTERNAL_SERVER_ERROR);
            done();
          });
        });
      });
  
      context("Cloud Controller user spaces api returns 401", function() {
        beforeEach(function() {
          nock("https://api.bosh-lite.com")
            .get("/v2/info")
            .reply(HttpStatus.OK, { "token_endpoint": "https://uaa.bosh-lite.com" });
  
          nock("https://uaa.bosh-lite.com")
            .get("/userinfo")
            .reply(HttpStatus.OK, { "user_id": theUserId });
          nock("https://api.bosh-lite.com")
            .get(/\/v2\/users\/.+\/spaces\?.+/)
            .reply(HttpStatus.UNAUTHORIZED, { "message": "UNAUTHORIZED" });
  
        });
        it("should return error", function(done) {
          oauth.checkUserAuthorization(req, function(error, result) {
            expect(result).to.equal(null);
            expect(error).to.deep.equal({
              "statusCode": HttpStatus.UNAUTHORIZED
            });
            done();
          });
        });
      });
      context("Cloud Controller user spaces api returns 503", function() {
        beforeEach(function() {
          nock("https://api.bosh-lite.com")
            .get("/v2/info")
            .reply(HttpStatus.OK, { "token_endpoint": "https://uaa.bosh-lite.com" });
  
          nock("https://uaa.bosh-lite.com")
            .get("/userinfo")
            .reply(HttpStatus.OK, { "user_id": theUserId });
          nock("https://api.bosh-lite.com")
            .get(/\/v2\/users\/.+\/spaces\?.+/)
            .reply(HttpStatus.SERVICE_UNAVAILABLE, { "message": "SERVICE_UNAVAILABLE" });
  
        });
        it("should return error with status code of Cloud Controller user spaces api", function(done) {
          oauth.checkUserAuthorization(req, function(error, result) {
            expect(result).to.equal(null);
            expect(error).to.deep.equal({
              "statusCode": HttpStatus.SERVICE_UNAVAILABLE
            });
            done();
          });
        });
      });
  
  
  
      context("user is not space developer", function() {
        beforeEach(function() {
          nock("https://api.bosh-lite.com")
            .get("/v2/info")
            .reply(HttpStatus.OK, { "token_endpoint": "https://uaa.bosh-lite.com" });
  
          nock("https://uaa.bosh-lite.com")
            .get("/userinfo")
            .reply(HttpStatus.OK, { "user_id": theUserId });
          nock("https://api.bosh-lite.com")
            .get(/\/v2\/users\/.+\/spaces\?.+/)
            .reply(HttpStatus.OK, {
              "total_results": 0,
              "total_pages": 1,
              "prev_url": null,
              "next_url": null,
              "resources": [
  
              ]
            });
  
        });
        it("should return false", function(done) {
          oauth.checkUserAuthorization(req, function(error, result) {
            expect(result).to.equal(false);
            expect(error).to.deep.equal(null);
            done();
          });
        });
      });
  
      context("user is space developer", function() {
        beforeEach(function() {
          nock("https://api.bosh-lite.com")
            .get("/v2/info")
            .reply(HttpStatus.OK, { "token_endpoint": "https://uaa.bosh-lite.com" });
  
          nock("https://uaa.bosh-lite.com")
            .get("/userinfo")
            .reply(HttpStatus.OK, { "user_id": theUserId });
          nock("https://api.bosh-lite.com")
            .get(/\/v2\/users\/.+\/spaces\?.+/)
            .reply(HttpStatus.OK, {
              "total_results": 1,
              "total_pages": 1,
              "prev_url": null,
              "next_url": null
            });
  
        });
        it("should return true", function(done) {
          oauth.checkUserAuthorization(req, function(error, result) {
            expect(result).to.equal(true);
            expect(error).to.deep.equal(null);
            done();
          });
        });
      });

    });

  });
});