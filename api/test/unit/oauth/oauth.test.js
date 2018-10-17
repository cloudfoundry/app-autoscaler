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
        return "fake-token";
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
    context("user token belongs to admin", function() {
      beforeEach(function() {
        req.header = function(headerName) {
          return "bearer eyJhbGciOiJSUzI1NiIsImtpZCI6ImtleS0xIiwidHlwIjoiSldUIn0.eyJqdGkiOiI5OTIxMjY1MjY2Zjg0MDVhOWI2ZWM5NDQ2MDY1YjIxMyIsInN1YiI6IjA0ZmJmZWEzLWE0ZjQtNDMwYi04NzdhLTQ3NWMxNGNmMTRmZCIsInNjb3BlIjpbIm9wZW5pZCIsInJvdXRpbmcucm91dGVyX2dyb3Vwcy53cml0ZSIsInNjaW0ucmVhZCIsImNsb3VkX2NvbnRyb2xsZXIuYWRtaW4iLCJ1YWEudXNlciIsInJvdXRpbmcucm91dGVyX2dyb3Vwcy5yZWFkIiwiY2xvdWRfY29udHJvbGxlci5yZWFkIiwicGFzc3dvcmQud3JpdGUiLCJjbG91ZF9jb250cm9sbGVyLndyaXRlIiwibmV0d29yay5hZG1pbiIsImRvcHBsZXIuZmlyZWhvc2UiLCJzY2ltLndyaXRlIl0sImNsaWVudF9pZCI6ImNmIiwiY2lkIjoiY2YiLCJhenAiOiJjZiIsImdyYW50X3R5cGUiOiJwYXNzd29yZCIsInVzZXJfaWQiOiIwNGZiZmVhMy1hNGY0LTQzMGItODc3YS00NzVjMTRjZjE0ZmQiLCJvcmlnaW4iOiJ1YWEiLCJ1c2VyX25hbWUiOiJhZG1pbiIsImVtYWlsIjoiYWRtaW4iLCJyZXZfc2lnIjoiMjhhMzIwYjQiLCJpYXQiOjE1Mzc5NTI1OTUsImV4cCI6MTUzNzk1MzE5NSwiaXNzIjoiaHR0cHM6Ly91YWEuYm9zaC1saXRlLmNvbS9vYXV0aC90b2tlbiIsInppZCI6InVhYSIsImF1ZCI6WyJzY2ltIiwiY2xvdWRfY29udHJvbGxlciIsInBhc3N3b3JkIiwiY2YiLCJ1YWEiLCJvcGVuaWQiLCJkb3BwbGVyIiwicm91dGluZy5yb3V0ZXJfZ3JvdXBzIiwibmV0d29yayJdfQ.Gex65d_k_2715a54vzXJqANodUE6hvrZKrITLTucGseFobdV2PnzwiIJT7-GQj-34lWA_aRdeX3rn6aXhmhJtCEETTF72ZVUbb8onuRWR1L3Q6P1j9L1BB3_W6V1GvN7sEPiVugdj9qBQgPeIYgfNMTOUUU9Z2hWkczJUXTX0ynjSeMASoGm8O5M3vCqVw86WKYBg0AfF4qwS2hCzACYngFQNuzrNGwhVGaH5yaAjjHvr1fyyYrJdehuytgUB08c-iGQLemdlLLutAiP3Rss8PF5C3orBQah0AT-OMlKahjlWHuRC9jB88TJ71TS08t98nBwfU8QBgG1z9XCP0NkuQ";
        }
      });
      it("should return true", function() {
        oauth.checkUserAuthorization(req, function(error, result) {
          expect(result).to.equal(true);
          expect(error).to.equal(null);
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