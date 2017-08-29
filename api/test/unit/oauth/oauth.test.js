"use strict";

process.env.NODE_TLS_REJECT_UNAUTHORIZED = "0";
var request = require("supertest");
var expect = require("chai").expect;
var fs = require("fs");
var path = require("path");
var HttpStatus = require('http-status-codes');

var logger = require("../../../lib/log/logger");
var oauth = require("../../../lib/oauth/oauth")(path.join(__dirname, "../../../config/settings.json"));
var nock = require("nock");
var theAppId = "the-app-guid";
var theUserId = "the-user-id";
var req = {};
var cloudControllerEndpoint;
var oauthEndpoint;
describe("Routing Metrics", function() {
  this.timeout(10000 + 1000); //timeout is 10s and here wait 11s
  beforeEach(function() {
    req = {
      protocol: "https",
      host: "app.bosh-lite.com",
      path: "/v1/apps/1503978853156012454/routes",
      params: {
        "app_id": theAppId
      },
      header: function(hearName) {
        return "fake-token";
      }
    };
  })
  beforeEach(function() {
    nock.cleanAll();
  });
  describe("oauth", function() {

    context("user token is not provided", function() {
      beforeEach(function() {
        req.header = function(hearName) {
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
    context("cloudController info endpoint is down", function() {
      it("should return error", function(done) {
        oauth.checkUserAuthorization(req, function(error, result) {
          expect(result).to.equal(null);
          expect(error.statusCode).to.equal(HttpStatus.INTERNAL_SERVER_ERROR);
          done();
        });
      });
    });

    context("cloudController info endpoint returns error", function() {
      beforeEach(function() {
        nock("https://api.bosh-lite.com")
          .get("/v2/info")
          .reply(HttpStatus.BAD_REQUEST, { "message": "bad request" });
      });
      it("should return error", function(done) {
        oauth.checkUserAuthorization(req, function(error, result) {
          expect(result).to.equal(null);
          expect(error).to.deep.equal({
            "statusCode": HttpStatus.BAD_REQUEST
          });
          done();
        });
      });
    });

    context("authorizationEndpoint is down", function() {
      beforeEach(function() {
        nock("https://api.bosh-lite.com")
          .get("/v2/info")
          .reply(HttpStatus.OK, { "authorization_endpoint": "https://uaa.bosh-lite.com" });
      });
      it("should return error", function(done) {
        oauth.checkUserAuthorization(req, function(error, result) {
          expect(result).to.equal(null);
          expect(error.statusCode).to.equal(HttpStatus.INTERNAL_SERVER_ERROR);
          done();
        });
      });
    });

    context("authorizationEndpoint returns error", function() {
      beforeEach(function() {
        nock("https://api.bosh-lite.com")
          .get("/v2/info")
          .reply(HttpStatus.OK, { "authorization_endpoint": "https://uaa.bosh-lite.com" });

        nock("https://uaa.bosh-lite.com")
          .get("/userinfo")
          .reply(HttpStatus.BAD_REQUEST, { "message": "bad request" });

      });
      it("should return error", function(done) {
        oauth.checkUserAuthorization(req, function(error, result) {
          expect(result).to.equal(null);
          expect(error).to.deep.equal({
            "statusCode": HttpStatus.BAD_REQUEST
          });
          done();
        });
      });
    });

    context("check user space endpoint is down", function() {
      beforeEach(function() {
        nock("https://api.bosh-lite.com/v2/info")
          .get("*")
          .reply(HttpStatus.OK, { "authorization_endpoint": "https://uaa.bosh-lite.com" });

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

    context("check user space endpoint returns error", function() {
      beforeEach(function() {
        nock("https://api.bosh-lite.com")
          .get("/v2/info")
          .reply(HttpStatus.OK, { "authorization_endpoint": "https://uaa.bosh-lite.com" });

        nock("https://uaa.bosh-lite.com")
          .get("/userinfo")
          .reply(HttpStatus.OK, { "user_id": theUserId });
        nock("https://api.bosh-lite.com")
          .get(/\/v2\/users\/.+\/spaces\?.+/)
          .reply(HttpStatus.BAD_REQUEST, { "message": "bad request" });

      });
      it("should return error", function(done) {
        oauth.checkUserAuthorization(req, function(error, result) {
          expect(result).to.equal(null);
          expect(error).to.deep.equal({
            "statusCode": HttpStatus.BAD_REQUEST
          });
          done();
        });
      });
    });

    context("check user space endpoint returns error", function() {
      beforeEach(function() {
        nock("https://api.bosh-lite.com")
          .get("/v2/info")
          .reply(HttpStatus.OK, { "authorization_endpoint": "https://uaa.bosh-lite.com" });

        nock("https://uaa.bosh-lite.com")
          .get("/userinfo")
          .reply(HttpStatus.OK, { "user_id": theUserId });
        // "/v2/users/" + userId + "/spaces?q=app_guid:" + appId + "&q=developer_guid:" + userId
        nock("https://api.bosh-lite.com")
          .get(/\/v2\/users\/.+\/spaces\?.+/)
          .reply(HttpStatus.BAD_REQUEST, { "message": "bad request" });

      });
      it("should return error", function(done) {
        oauth.checkUserAuthorization(req, function(error, result) {
          expect(result).to.equal(null);
          expect(error).to.deep.equal({
            "statusCode": HttpStatus.BAD_REQUEST
          });
          done();
        });
      });
    });

    context("user is not space developer", function() {
      beforeEach(function() {
        nock("https://api.bosh-lite.com")
          .get("/v2/info")
          .reply(HttpStatus.OK, { "authorization_endpoint": "https://uaa.bosh-lite.com" });

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
          .reply(HttpStatus.OK, { "authorization_endpoint": "https://uaa.bosh-lite.com" });

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