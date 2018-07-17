"use strict";

process.env.NODE_TLS_REJECT_UNAUTHORIZED = "0";
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
var servers;
var logger = require("../../../lib/log/logger");
var nock = require("nock");
var eventGeneratorUri = testSetting.eventGenerator.uri;
var theAppId = "the-app-guid";
var metricType = "memoryused";

describe("Routing Aggregated App Metrics", function() {

  before(function() {
    servers = API(testSetting, function(){});
    app = servers.internalServer;
    publicApp = servers.publicServer;
  })
  after(function(done) {
    app.close(function(){
      publicApp.close(done);
    });
  })
  beforeEach(function() {
    nock.cleanAll();
  });

  var histories = [
    { "app_id": theAppId, "timestamp": 100,"name": "memoryused", "unit": "megabytes", "value": "200"},
    { "app_id": theAppId, "timestamp": 110,"name": "memoryused", "unit": "megabytes", "value": "200"},
    { "app_id": theAppId, "timestamp": 150,"name": "memoryused", "unit": "megabytes", "value": "200"},
    { "app_id": theAppId, "timestamp": 170,"name": "memoryused", "unit": "megabytes", "value": "200"},
    { "app_id": theAppId, "timestamp": 200,"name": "memoryused", "unit": "megabytes", "value": "200"}
  ]
  describe("get metrics", function() {
    context("parameters", function() {

      context("start-time", function() {
        it("should return 200 when start-time is not provided", function(done) {
          nock(eventGeneratorUri)
            .get(/\/v1\/apps\/.+\/aggregated_metric_histories\/memoryused/)
            .reply(200, histories);
          request(app)
            .get("/v1/apps/12345/aggregated_metric_histories/" + metricType)
            .query({ "metric-type": metricType, "end-time": 200, "order": "desc", "page": 1, "results-per-page": 2 })
            .end(function(error, result) {
              expect(error).to.equal(null);
              expect(result.statusCode).to.equal(200);
              done();
            });
        });

        it("should return 400 when start-time is not integer", function(done) {
          nock(eventGeneratorUri)
            .get(/\/v1\/apps\/.+\/aggregated_metric_histories\/memoryused/)
            .reply(200, histories);
          request(app)
            .get("/v1/apps/12345/aggregated_metric_histories/" + metricType)
            .query({ "metric-type": metricType, "start-time": "not-integer", "end-time": 200, "order": "desc", "page": 1, "results-per-page": 2 })
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
          nock(eventGeneratorUri)
            .get(/\/v1\/apps\/.+\/aggregated_metric_histories\/memoryused/)
            .reply(200, histories);
          request(app)
            .get("/v1/apps/12345/aggregated_metric_histories/" + metricType)
            .query({ "metric-type": metricType, "start-time": 100, "order": "desc", "page": 1, "results-per-page": 2 })
            .end(function(error, result) {
              expect(error).to.equal(null);
              expect(result.statusCode).to.equal(200);
              done();
            });
        });

        it("should return 400 when end-time is not integer", function(done) {
          nock(eventGeneratorUri)
            .get(/\/v1\/apps\/.+\/aggregated_metric_histories\/memoryused/)
            .reply(200, histories);
          request(app)
            .get("/v1/apps/12345/aggregated_metric_histories/" + metricType)
            .query({ "metric-type": metricType, "end-time": "not-integer", "start-time": 100, "order": "desc", "page": 1, "results-per-page": 2 })
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

      context("order", function() {
        it("should return 200 when order is not provided", function(done) {
          nock(eventGeneratorUri)
            .get(/\/v1\/apps\/.+\/aggregated_metric_histories\/memoryused/)
            .reply(200, histories);
          request(app)
            .get("/v1/apps/12345/aggregated_metric_histories/" + metricType)
            .query({ "metric-type": metricType, "start-time": 100, "end-time": 200, "page": 1, "results-per-page": 2 })
            .end(function(error, result) {
              expect(error).to.equal(null);
              expect(result.statusCode).to.equal(200);
              done();
            });
        });

        it("should return 400 when order is not desc or asc", function(done) {
          nock(eventGeneratorUri)
            .get(/\/v1\/apps\/.+\/aggregated_metric_histories\/memoryused/)
            .reply(200, histories);
          request(app)
            .get("/v1/apps/12345/aggregated_metric_histories/" + metricType)
            .query({ "metric-type": metricType, "start-time": 100, "end-time": 200, "order": "not-desc-asc", "page": 1, "results-per-page": 2 })
            .end(function(error, result) {
              expect(error).to.equal(null);
              expect(result.statusCode).to.equal(400);
              expect(result.body).to.deep.equal({
                "error": "order must be DESC or ASC"
              });
              done();
            });
        });
      });

      context("page", function() {
        it("should return 200 when page is not provided", function(done) {
          nock(eventGeneratorUri)
            .get(/\/v1\/apps\/.+\/aggregated_metric_histories\/memoryused/)
            .reply(200, histories);
          request(app)
            .get("/v1/apps/12345/aggregated_metric_histories/" + metricType)
            .query({ "metric-type": metricType, "start-time": 100, "end-time": 200, "order": "desc", "results-per-page": 2 })
            .end(function(error, result) {
              expect(error).to.equal(null);
              expect(result.statusCode).to.equal(200);
              done();
            });
        });

        it("should return 400 when page is not integer", function(done) {
          nock(eventGeneratorUri)
            .get(/\/v1\/apps\/.+\/aggregated_metric_histories\/memoryused/)
            .reply(200, histories);
          request(app)
            .get("/v1/apps/12345/aggregated_metric_histories/" + metricType)
            .query({ "metric-type": metricType, "start-time": 100, "end-time": 200, "order": "desc", "page": "not-integer", "results-per-page": 2 })
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
          nock(eventGeneratorUri)
            .get(/\/v1\/apps\/.+\/aggregated_metric_histories\/memoryused/)
            .reply(200, histories);
          request(app)
            .get("/v1/apps/12345/aggregated_metric_histories/" + metricType)
            .query({ "metric-type": metricType, "start-time": 100, "end-time": 200, "order": "desc", "page": 1 })
            .end(function(error, result) {
              expect(error).to.equal(null);
              expect(result.statusCode).to.equal(200);

              done();
            });
        });

        it("should return 400 when results-per-page is not integer", function(done) {
          nock(eventGeneratorUri)
            .get(/\/v1\/apps\/.+\/aggregated_metric_histories\/memoryused/)
            .reply(200, histories);
          request(app)
            .get("/v1/apps/12345/aggregated_metric_histories/" + metricType)
            .query({ "metric-type": metricType, "start-time": 100, "end-time": 200, "order": "desc", "page": 1, "results-per-page": "not-integer" })
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
        nock(eventGeneratorUri)
          .get(/\/v1\/apps\/.+\/aggregated_metric_histories\/memoryused/)
          .replyWithError({
            'message': 'Error in requests metricsCollector',
            'details': 'fake body'
          });
        request(app)
          .get("/v1/apps/12345/aggregated_metric_histories/" + metricType)
          .query({ "metric-type": metricType, "start-time": 100, "end-time": 200, "order": "desc", "page": 1, "results-per-page": 2 })
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
        nock(eventGeneratorUri)
          .get(/\/v1\/apps\/.+\/aggregated_metric_histories\/memoryused/)
          .reply(500, { code: 'Interal-Server-Error', message: 'Error getting scaling histories from database' });
        request(app)
          .get("/v1/apps/12345/aggregated_metric_histories/" + metricType)
          .query({ "metric-type": metricType, "start-time": 100, "end-time": 200, "order": "desc", "page": 1, "results-per-page": 2 })
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
        nock(eventGeneratorUri)
          .get(/\/v1\/apps\/.+\/aggregated_metric_histories\/memoryused/)
          .reply(200, histories);
        request(app)
          .get("/v1/apps/12345/aggregated_metric_histories/" + metricType)
          .query({ "metric-type": metricType, "start-time": 100, "end-time": 500, "order": "desc", "page": 1, "results-per-page": 2 })
          .end(function(error, result) {
            expect(error).to.equal(null);
            expect(result.statusCode).to.equal(200);
            expect(result.body).to.deep.equal({
              total_results: 5,
              total_pages: 3,
              page: 1,
              resources: histories.slice(0, 2)
            });
            done();
          });
      });

      it("get the 2nd page", function(done) {
        nock(eventGeneratorUri)
          .get(/\/v1\/apps\/.+\/aggregated_metric_histories\/memoryused/)
          .reply(200, histories);
        request(app)
          .get("/v1/apps/12345/aggregated_metric_histories/" + metricType)
          .query({ "metric-type": metricType, "start-time": 100, "end-time": 500, "order": "desc", "page": 2, "results-per-page": 2 })
          .end(function(error, result) {
            expect(error).to.equal(null);
            expect(result.statusCode).to.equal(200);
            expect(result.body).to.deep.equal({
              total_results: 5,
              total_pages: 3,
              page: 2,
              resources: histories.slice(2, 4)
            });
            done();
          });
      });

      it("get the 3rd page and only has one record", function(done) {
        nock(eventGeneratorUri)
          .get(/\/v1\/apps\/.+\/aggregated_metric_histories\/memoryused/)
          .reply(200, histories);
        request(app)
          .get("/v1/apps/12345/aggregated_metric_histories/" + metricType)
          .query({ "metric-type": metricType, "start-time": 100, "end-time": 500, "order": "desc", "page": 3, "results-per-page": 2 })
          .end(function(error, result) {
            expect(error).to.equal(null);
            expect(result.statusCode).to.equal(200);
            expect(result.body).to.deep.equal({
              total_results: 5,
              total_pages: 3,
              page: 3,
              resources: histories.slice(4)
            });
            done();
          });
      });

      it("get the 4th page and there is no record", function(done) {
        nock(eventGeneratorUri)
          .get(/\/v1\/apps\/.+\/aggregated_metric_histories\/memoryused/)
          .reply(200, histories);
        request(app)
          .get("/v1/apps/12345/aggregated_metric_histories/" + metricType)
          .query({ "metric-type": metricType, "start-time": 100, "end-time": 500, "order": "desc", "page": 4, "results-per-page": 2 })
          .end(function(error, result) {
            expect(error).to.equal(null);
            expect(result.statusCode).to.equal(200);
            expect(result.body).to.deep.equal({
              total_results: 5,
              total_pages: 3,
              page: 4,
              resources: []
            });
            done();
          });
      });
    });
  });
});
