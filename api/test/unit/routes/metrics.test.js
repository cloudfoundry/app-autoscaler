"use strict";

process.env.NODE_TLS_REJECT_UNAUTHORIZED = "0";
var request = require("supertest");
var expect = require("chai").expect;
var fs = require("fs");
var path = require("path");
var settings = require(path.join(__dirname, '../../../lib/config/setting.js'))((JSON.parse(
  fs.readFileSync(path.join(__dirname, '../../../config/settings.json'), 'utf8'))));
var relativePath = path.relative(process.cwd(), path.join(__dirname, "../../../../test-certs"));
var testSetting = require(path.join(__dirname, '../test.helper.js'))(relativePath, settings);
var API = require("../../../app.js");
var nock = require("nock");
var HttpStatus = require('http-status-codes');

var app;
var publicApp;
var healthApp;
var servers;
var metricsCollectorUri = testSetting.metricsCollector.uri;
var theAppId = "the-app-guid";
var metricType = "memoryused";
var theUserId = "the-user-id";
var theUserToken = "token-type the-user-token"

describe("Routing Metrics", function() {

  before(function() {
    testSetting.metricsCollector.tls = null;
    servers = API(testSetting, function() {});
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
  beforeEach(function() {
    nock.cleanAll();
    nock("https://api.bosh-lite.com")
      .get("/v2/info")
      .reply(HttpStatus.OK, { "token_endpoint": "https://uaa.bosh-lite.com" });

    nock("https://uaa.bosh-lite.com")
      .post("/check_token?token=the-user-token")
      .reply(HttpStatus.OK, { "scope": ["cloud_controller.read","password.write","cloud_controller.write","openid","network.admin","network.write","uaa.user"] });

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
  var histories = [
    { "app_id": theAppId, "timestamp": 100, "instance_index": 0, "collected_at": 0, "name": "memoryused", "unit": "megabytes", "value": "200" },
    { "app_id": theAppId, "timestamp": 110, "instance_index": 1, "collected_at": 1, "name": "memoryused", "unit": "megabytes", "value": "200" },
    { "app_id": theAppId, "timestamp": 150, "instance_index": 0, "collected_at": 0, "name": "memoryused", "unit": "megabytes", "value": "200" },
    { "app_id": theAppId, "timestamp": 170, "instance_index": 1, "collected_at": 1, "name": "memoryused", "unit": "megabytes", "value": "200" },
    { "app_id": theAppId, "timestamp": 200, "instance_index": 0, "collected_at": 0, "name": "memoryused", "unit": "megabytes", "value": "200" }
  ]
  describe("get metrics", function() {
    context("parameters", function() {

      context("instance_index", function() {
        it("should return 200 when instance_index is not provided", function(done) {
          nock(metricsCollectorUri)
            .get(/\/v1\/apps\/.+\/metric_histories\/memoryused/)
            .reply(200, histories);
          request(publicApp)
            .get("/v1/apps/12345/metric_histories/" + metricType)
            .set("Authorization", theUserToken)
            .query({ "end-time": 200, "order-direction": "desc", "page": 1, "results-per-page": 2 })
            .end(function(error, result) {
              expect(error).to.equal(null);
              expect(result.statusCode).to.equal(200);
              done();
            });
        });

        it("should return 400 when instance_index is not integer", function(done) {
          nock(metricsCollectorUri)
            .get(/\/v1\/apps\/.+\/metric_histories\/memoryused/)
            .reply(200, histories);
          request(publicApp)
            .get("/v1/apps/12345/metric_histories/" + metricType)
            .set("Authorization", theUserToken)
            .query({ "instance-index": "not-integer", "start-time": 100, "end-time": 200, "order-direction": "desc", "page": 1, "results-per-page": 2 })
            .end(function(error, result) {
              expect(error).to.equal(null);
              expect(result.statusCode).to.equal(400);
              expect(result.body).to.deep.equal({
                "error": "instance-index must be an integer"
              });
              done();
            });
        });

        it("should return 400 when instance_index is smaller than 0", function(done) {
          nock(metricsCollectorUri)
            .get(/\/v1\/apps\/.+\/metric_histories\/memoryused/)
            .reply(200, histories);
          request(publicApp)
            .get("/v1/apps/12345/metric_histories/" + metricType)
            .set("Authorization", theUserToken)
            .query({ "instance-index": -1,"start-time": 100, "end-time": 200, "order-direction": "desc", "page": 1, "results-per-page": 2 })
            .end(function(error, result) {
              expect(error).to.equal(null);
              expect(result.statusCode).to.equal(400);
              expect(result.body).to.deep.equal({
                "error": "instance-index must be greater than or equal to 0"
              });
              done();
            });
        });
      });

      context("start-time", function() {
        it("should return 200 when start-time is not provided", function(done) {
          nock(metricsCollectorUri)
            .get(/\/v1\/apps\/.+\/metric_histories\/memoryused/)
            .reply(200, histories);
          request(publicApp)
            .get("/v1/apps/12345/metric_histories/" + metricType)
            .set("Authorization", theUserToken)
            .query({ "end-time": 200, "order-direction": "desc", "page": 1, "results-per-page": 2 })
            .end(function(error, result) {
              expect(error).to.equal(null);
              expect(result.statusCode).to.equal(200);
              done();
            });
        });

        it("should return 400 when start-time is not integer", function(done) {
          nock(metricsCollectorUri)
            .get(/\/v1\/apps\/.+\/metric_histories\/memoryused/)
            .reply(200, histories);
          request(publicApp)
            .get("/v1/apps/12345/metric_histories/" + metricType)
            .set("Authorization", theUserToken)
            .query({ "start-time": "not-integer", "end-time": 200, "order-direction": "desc", "page": 1, "results-per-page": 2 })
            .end(function(error, result) {
              expect(error).to.equal(null);
              expect(result.statusCode).to.equal(400);
              expect(result.body).to.deep.equal({
                "error": "start-time must be an integer"
              });
              done();
            });
        });
      });

      context("end-time", function() {
        it("should return 200 when end-time is not provided", function(done) {
          nock(metricsCollectorUri)
            .get(/\/v1\/apps\/.+\/metric_histories\/memoryused/)
            .reply(200, histories);
          request(publicApp)
            .get("/v1/apps/12345/metric_histories/" + metricType)
            .set("Authorization", theUserToken)
            .query({ "start-time": 100, "order-direction": "desc", "page": 1, "results-per-page": 2 })
            .end(function(error, result) {
              expect(error).to.equal(null);
              expect(result.statusCode).to.equal(200);
              done();
            });
        });

        it("should return 400 when end-time is not integer", function(done) {
          nock(metricsCollectorUri)
            .get(/\/v1\/apps\/.+\/metric_histories\/memoryused/)
            .reply(200, histories);
          request(publicApp)
            .get("/v1/apps/12345/metric_histories/" + metricType)
            .set("Authorization", theUserToken)
            .query({ "end-time": "not-integer", "start-time": 100, "order-direction": "desc", "page": 1, "results-per-page": 2 })
            .end(function(error, result) {
              expect(error).to.equal(null);
              expect(result.statusCode).to.equal(400);
              expect(result.body).to.deep.equal({
                "error": "end-time must be an integer"
              });
              done();
            });
        });
      });

      context("order-direction", function() {
        it("should return 200 when order is not provided", function(done) {
          nock(metricsCollectorUri)
            .get(/\/v1\/apps\/.+\/metric_histories\/memoryused/)
            .reply(200, histories);
          request(publicApp)
            .get("/v1/apps/12345/metric_histories/" + metricType)
            .set("Authorization", theUserToken)
            .query({ "start-time": 100, "end-time": 200, "page": 1, "results-per-page": 2 })
            .end(function(error, result) {
              expect(error).to.equal(null);
              expect(result.statusCode).to.equal(200);
              done();
            });
        });

        it("should return 400 when order is not desc or asc", function(done) {
          nock(metricsCollectorUri)
            .get(/\/v1\/apps\/.+\/metric_histories\/memoryused/)
            .reply(200, histories);
          request(publicApp)
            .get("/v1/apps/12345/metric_histories/" + metricType)
            .set("Authorization", theUserToken)
            .query({ "start-time": 100, "end-time": 200, "order-direction": "not-desc-asc", "page": 1, "results-per-page": 2 })
            .end(function(error, result) {
              expect(error).to.equal(null);
              expect(result.statusCode).to.equal(400);
              expect(result.body).to.deep.equal({
                "error": "order-direction must be DESC or ASC"
              });
              done();
            });
        });
      });

      context("page", function() {
        it("should return 200 when page is not provided", function(done) {
          nock(metricsCollectorUri)
            .get(/\/v1\/apps\/.+\/metric_histories\/memoryused/)
            .reply(200, histories);
          request(publicApp)
            .get("/v1/apps/12345/metric_histories/" + metricType)
            .set("Authorization", theUserToken)
            .query({ "start-time": 100, "end-time": 200, "order-direction": "desc", "results-per-page": 2 })
            .end(function(error, result) {
              expect(error).to.equal(null);
              expect(result.statusCode).to.equal(200);
              done();
            });
        });

        it("should return 400 when page is not integer", function(done) {
          nock(metricsCollectorUri)
            .get(/\/v1\/apps\/.+\/metric_histories\/memoryused/)
            .reply(200, histories);
          request(publicApp)
            .get("/v1/apps/12345/metric_histories/" + metricType)
            .set("Authorization", theUserToken)
            .query({ "start-time": 100, "end-time": 200, "order-direction": "desc", "page": "not-integer", "results-per-page": 2 })
            .end(function(error, result) {
              expect(error).to.equal(null);
              expect(result.statusCode).to.equal(400);
              expect(result.body).to.deep.equal({
                "error": "page must be an integer"
              });
              done();
            });
        });
      });

      context("results-per-page", function() {
        it("should return 200 when results-per-page is not provided", function(done) {
          nock(metricsCollectorUri)
            .get(/\/v1\/apps\/.+\/metric_histories\/memoryused/)
            .reply(200, histories);
          request(publicApp)
            .get("/v1/apps/12345/metric_histories/" + metricType)
            .set("Authorization", theUserToken)
            .query({ "start-time": 100, "end-time": 200, "order-direction": "desc", "page": 1 })
            .end(function(error, result) {
              expect(error).to.equal(null);
              expect(result.statusCode).to.equal(200);

              done();
            });
        });

        it("should return 400 when results-per-page is not integer", function(done) {
          nock(metricsCollectorUri)
            .get(/\/v1\/apps\/.+\/metric_histories\/memoryused/)
            .reply(200, histories);
          request(publicApp)
            .get("/v1/apps/12345/metric_histories/" + metricType)
            .set("Authorization", theUserToken)
            .query({ "start-time": 100, "end-time": 200, "order-direction": "desc", "page": 1, "results-per-page": "not-integer" })
            .end(function(error, result) {
              expect(error).to.equal(null);
              expect(result.statusCode).to.equal(400);
              expect(result.body).to.deep.equal({
                "error": "results-per-page must be an integer"
              });
              done();
            });
        });
      });
    });
    context("metricsCollector error", function() {
      it("should return 500 when there is error when requesting to metricsCollector", function(done) {
        nock(metricsCollectorUri)
          .get(/\/v1\/apps\/.+\/metric_histories\/memoryused/)
          .replyWithError({
            'message': 'Error in requests metricsCollector',
            'details': 'fake body'
          });
        request(publicApp)
          .get("/v1/apps/12345/metric_histories/" + metricType)
          .set("Authorization", theUserToken)
          .query({ "start-time": 100, "end-time": 200, "order-direction": "desc", "page": 1, "results-per-page": 2 })
          .end(function(error, result) {
            expect(error).to.equal(null);
            expect(result.statusCode).to.equal(500);
            expect(result.body).to.deep.equal({
              error: 'Error in requests metricsCollector'
            });
            done();
          });
      });

      it('should return 500 when there is internal error in metricsCollector', function(done) {
        nock(metricsCollectorUri)
          .get(/\/v1\/apps\/.+\/metric_histories\/memoryused/)
          .reply(500, { code: 'Interal-Server-Error', message: 'Error getting scaling histories from database' });
        request(publicApp)
          .get("/v1/apps/12345/metric_histories/" + metricType)
          .set("Authorization", theUserToken)
          .query({ "start-time": 100, "end-time": 200, "order-direction": "desc", "page": 1, "results-per-page": 2 })
          .end(function(error, result) {
            expect(error).to.equal(null);
            expect(result.statusCode).to.equal(500);
            expect(result.body).to.deep.equal({
              error: 'Error getting scaling histories from database'
            });
            done();
          });
      });
    });

    context("get metrics", function() {
      it("get the 1st page", function(done) {
        nock(metricsCollectorUri)
          .get(/\/v1\/apps\/.+\/metric_histories\/memoryused/)
          .reply(200, histories);
        request(publicApp)
          .get("/v1/apps/12345/metric_histories/" + metricType)
          .set("Authorization", theUserToken)
          .query({ "start-time": 100, "end-time": 500, "order-direction": "desc", "page": 1, "results-per-page": 2 })
          .end(function(error, result) {
            expect(error).to.equal(null);
            expect(result.statusCode).to.equal(200);
            expect(result.body).to.deep.equal({
              total_results: 5,
              total_pages: 3,
              page: 1,
              prev_url: null,
              next_url: "/v1/apps/12345/metric_histories/memoryused?start-time=100&end-time=500&order-direction=desc&page=2&results-per-page=2",
              resources: histories.slice(0, 2)
            });
            done();
          });
      });

      it("get the 2nd page", function(done) {
        nock(metricsCollectorUri)
          .get(/\/v1\/apps\/.+\/metric_histories\/memoryused/)
          .reply(200, histories);
        request(publicApp)
          .get("/v1/apps/12345/metric_histories/" + metricType)
          .set("Authorization", theUserToken)
          .query({ "start-time": 100, "end-time": 500, "order-direction": "desc", "page": 2, "results-per-page": 2 })
          .end(function(error, result) {
            expect(error).to.equal(null);
            expect(result.statusCode).to.equal(200);
            expect(result.body).to.deep.equal({
              total_results: 5,
              total_pages: 3,
              page: 2,
              prev_url: "/v1/apps/12345/metric_histories/memoryused?start-time=100&end-time=500&order-direction=desc&page=1&results-per-page=2",
              next_url: "/v1/apps/12345/metric_histories/memoryused?start-time=100&end-time=500&order-direction=desc&page=3&results-per-page=2",
              resources: histories.slice(2, 4)
            });
            done();
          });
      });

      it("get the 3rd page and only has one record", function(done) {
        nock(metricsCollectorUri)
          .get(/\/v1\/apps\/.+\/metric_histories\/memoryused/)
          .reply(200, histories);
        request(publicApp)
          .get("/v1/apps/12345/metric_histories/" + metricType)
          .set("Authorization", theUserToken)
          .query({ "start-time": 100, "end-time": 500, "order-direction": "desc", "page": 3, "results-per-page": 2 })
          .end(function(error, result) {
            expect(error).to.equal(null);
            expect(result.statusCode).to.equal(200);
            expect(result.body).to.deep.equal({
              total_results: 5,
              total_pages: 3,
              page: 3,
              prev_url: "/v1/apps/12345/metric_histories/memoryused?start-time=100&end-time=500&order-direction=desc&page=2&results-per-page=2",
              next_url: null,
              resources: histories.slice(4)
            });
            done();
          });
      });

      it("get the 4th page and there is no record", function(done) {
        nock(metricsCollectorUri)
          .get(/\/v1\/apps\/.+\/metric_histories\/memoryused/)
          .reply(200, histories);
        request(publicApp)
          .get("/v1/apps/12345/metric_histories/" + metricType)
          .set("Authorization", theUserToken)
          .query({ "start-time": 100, "end-time": 500, "order-direction": "desc", "page": 4, "results-per-page": 2 })
          .end(function(error, result) {
            expect(error).to.equal(null);
            expect(result.statusCode).to.equal(200);
            expect(result.body).to.deep.equal({
              total_results: 5,
              total_pages: 3,
              page: 4,
              prev_url: "/v1/apps/12345/metric_histories/memoryused?start-time=100&end-time=500&order-direction=desc&page=3&results-per-page=2",
              next_url: null,
              resources: []
            });
            done();
          });
      });

      it("get the 5th page and there is no record and the prev_url and next_url are both null", function(done) {
        nock(metricsCollectorUri)
          .get(/\/v1\/apps\/.+\/metric_histories\/memoryused/)
          .reply(200, histories);
        request(publicApp)
          .get("/v1/apps/12345/metric_histories/" + metricType)
          .set("Authorization", theUserToken)
          .query({ "start-time": 100, "end-time": 500, "order-direction": "desc", "page": 5, "results-per-page": 2 })
          .end(function(error, result) {
            expect(error).to.equal(null);
            expect(result.statusCode).to.equal(200);
            expect(result.body).to.deep.equal({
              total_results: 5,
              total_pages: 3,
              page: 5,
              prev_url: null,
              next_url: null,
              resources: []
            });
            done();
          });
      });
    });
  });
});