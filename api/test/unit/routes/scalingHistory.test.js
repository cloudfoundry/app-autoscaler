"use strict";

process.env.NODE_TLS_REJECT_UNAUTHORIZED = "0";
var request = require("supertest");
var expect = require("chai").expect;
var fs = require("fs");
var path = require("path");
var uuidV4 = require("uuid/v4");
var settings = require(path.join(__dirname, "../../../lib/config/setting.js"))((JSON.parse(
  fs.readFileSync(path.join(__dirname, "../../../config/settings.json"), "utf8"))));
var API = require("../../../app.js");
var app;
var logger = require("../../../lib/log/logger");
var nock = require("nock");
var scalingEngineUri = settings.scalingEngine.uri;
var theAppId = "the-app-guid";

describe("Routing ScalingHistory", function() {

  before(function() {
    app = API(path.join(__dirname, "../../../config/settings.json"));
  })
  after(function(done) {
    app.close(done);
  })
  beforeEach(function() {
    nock.cleanAll();
  });
  var histories = [
    { "app_id": theAppId, "timestamp": 300, "scaling_type": 0, "status": 0, "old_instances": 2, "new_instances": 4, "reason": "a reason", "message": "", "error": "" },
    { "app_id": theAppId, "timestamp": 250, "scaling_type": 1, "status": 1, "old_instances": 2, "new_instances": 4, "reason": "a reason", "message": "", "error": "" },
    { "app_id": theAppId, "timestamp": 200, "scaling_type": 0, "status": 0, "old_instances": 2, "new_instances": 4, "reason": "a reason", "message": "", "error": "" },
    { "app_id": theAppId, "timestamp": 150, "scaling_type": 1, "status": 1, "old_instances": 2, "new_instances": 4, "reason": "a reason", "message": "", "error": "" },
    { "app_id": theAppId, "timestamp": 100, "scaling_type": 0, "status": 0, "old_instances": 2, "new_instances": 4, "reason": "a reason", "message": "", "error": "" }
  ]
  describe("get scaling history", function() {
    context("parameters", function() {

      context("start-time", function() {
        it("should return 200 when start-time is not provided", function(done) {
          nock(scalingEngineUri)
            .get(/\/v1\/apps\/.+\/scaling_histories/)
            .reply(200, histories);
          request(app)
            .get("/v1/apps/12345/scaling_histories")
            .query({ "end-time": 200, "order": "desc", "page": 1, "results-per-page": 2 })
            .end(function(error, result) {
              expect(error).to.equal(null);
              expect(result.statusCode).to.equal(200);
              done();
            });
        });

        it("should return 400 when start-time is not integer", function(done) {
          nock(scalingEngineUri)
            .get(/\/v1\/apps\/.+\/scaling_histories/)
            .reply(200, histories);
          request(app)
            .get("/v1/apps/12345/scaling_histories")
            .query({ "start-time": "not-integer", "end-time": 200, "order": "desc", "page": 1, "results-per-page": 2 })
            .end(function(error, result) {
              expect(error).to.equal(null);
              expect(result.statusCode).to.equal(400);
              expect(result.body).to.deep.equal({
                "description": "start-time must be an integer"
              });
              done();
            });
        });
      });

      context("end-time", function() {
        it("should return 200 when end-time is not provided", function(done) {
          nock(scalingEngineUri)
            .get(/\/v1\/apps\/.+\/scaling_histories/)
            .reply(200, histories);
          request(app)
            .get("/v1/apps/12345/scaling_histories")
            .query({ "start-time": 100, "order": "desc", "page": 1, "results-per-page": 2 })
            .end(function(error, result) {
              expect(error).to.equal(null);
              expect(result.statusCode).to.equal(200);
              done();
            });
        });

        it("should return 400 when end-time is not integer", function(done) {
          nock(scalingEngineUri)
            .get(/\/v1\/apps\/.+\/scaling_histories/)
            .reply(200, histories);
          request(app)
            .get("/v1/apps/12345/scaling_histories")
            .query({ "end-time": "not-integer", "start-time": 100, "order": "desc", "page": 1, "results-per-page": 2 })
            .end(function(error, result) {
              expect(error).to.equal(null);
              expect(result.statusCode).to.equal(400);
              expect(result.body).to.deep.equal({
                "description": "end-time must be an integer"
              });
              done();
            });
        });
      });

      context("order", function() {
        it("should return 200 when order is not provided", function(done) {
          nock(scalingEngineUri)
            .get(/\/v1\/apps\/.+\/scaling_histories/)
            .reply(200, histories);
          request(app)
            .get("/v1/apps/12345/scaling_histories")
            .query({ "start-time": 100, "end-time": 200, "page": 1, "results-per-page": 2 })
            .end(function(error, result) {
              expect(error).to.equal(null);
              expect(result.statusCode).to.equal(200);
              done();
            });
        });

        it("should return 400 when order is not desc or asc", function(done) {
          nock(scalingEngineUri)
            .get(/\/v1\/apps\/.+\/scaling_histories/)
            .reply(200, histories);
          request(app)
            .get("/v1/apps/12345/scaling_histories")
            .query({ "start-time": 100, "end-time": 200, "order": "not-desc-asc", "page": 1, "results-per-page": 2 })
            .end(function(error, result) {
              expect(error).to.equal(null);
              expect(result.statusCode).to.equal(400);
              expect(result.body).to.deep.equal({
                "description": "order must be desc or asc"
              });
              done();
            });
        });
      });

      context("page", function() {
        it("should return 200 when page is not provided", function(done) {
          nock(scalingEngineUri)
            .get(/\/v1\/apps\/.+\/scaling_histories/)
            .reply(200, histories);
          request(app)
            .get("/v1/apps/12345/scaling_histories")
            .query({ "start-time": 100, "end-time": 200, "order": "desc", "results-per-page": 2 })
            .end(function(error, result) {
              expect(error).to.equal(null);
              expect(result.statusCode).to.equal(200);
              done();
            });
        });

        it("should return 400 when page is not integer", function(done) {
          nock(scalingEngineUri)
            .get(/\/v1\/apps\/.+\/scaling_histories/)
            .reply(200, histories);
          request(app)
            .get("/v1/apps/12345/scaling_histories")
            .query({ "start-time": 100, "end-time": 200, "order": "desc", "page": "not-integer", "results-per-page": 2 })
            .end(function(error, result) {
              expect(error).to.equal(null);
              expect(result.statusCode).to.equal(400);
              expect(result.body).to.deep.equal({
                "description": "page must be an integer"
              });
              done();
            });
        });
      });

      context("results-per-page", function() {
        it("should return 200 when results-per-page is not provided", function(done) {
          nock(scalingEngineUri)
            .get(/\/v1\/apps\/.+\/scaling_histories/)
            .reply(200, histories);
          request(app)
            .get("/v1/apps/12345/scaling_histories")
            .query({ "start-time": 100, "end-time": 200, "order": "desc", "page": 1 })
            .end(function(error, result) {
              expect(error).to.equal(null);
              expect(result.statusCode).to.equal(200);

              done();
            });
        });

        it("should return 400 when results-per-page is not integer", function(done) {
          nock(scalingEngineUri)
            .get(/\/v1\/apps\/.+\/scaling_histories/)
            .reply(200, histories);
          request(app)
            .get("/v1/apps/12345/scaling_histories")
            .query({ "start-time": 100, "end-time": 200, "order": "desc", "page": 1, "results-per-page": "not-integer" })
            .end(function(error, result) {
              expect(error).to.equal(null);
              expect(result.statusCode).to.equal(400);
              expect(result.body).to.deep.equal({
                "description": "results-per-page must be an integer"
              });
              done();
            });
        });
      });
    });
    context("scalingEngine error", function() {
      it("should return 500 when there is error when requesting to scalingEngine", function(done) {
        nock(scalingEngineUri)
          .get(/\/v1\/apps\/.+\/scaling_histories/)
          .replyWithError({
            'message': 'Error in requests scalingEngine',
            'details': 'fake body'
          });
        request(app)
          .get("/v1/apps/12345/scaling_histories")
          .query({ "start-time": 100, "end-time": 200, "order": "desc", "page": 1, "results-per-page": 2 })
          .end(function(error, result) {
            expect(error).to.equal(null);
            expect(result.statusCode).to.equal(500);
            expect(result.body).to.deep.equal({
              description: 'Error in requests scalingEngine'
            });
            done();
          });
      });

      it('should return 500 when there is internal error in scalingEngine', function(done) {
        nock(scalingEngineUri)
          .get(/\/v1\/apps\/.+\/scaling_histories/)
          .reply(500, { code: 'Interal-Server-Error', message: 'Error getting scaling histories from database' });
        request(app)
          .get("/v1/apps/12345/scaling_histories")
          .query({ "start-time": 100, "end-time": 200, "order": "desc", "page": 1, "results-per-page": 2 })
          .end(function(error, result) {
            expect(error).to.equal(null);
            expect(result.statusCode).to.equal(500);
            expect(result.body).to.deep.equal({
              description: 'Error getting scaling histories from database'
            });
            done();
          });
      });
    });

    context("get scaling histories", function() {
      it("get the 1st page", function(done) {
        nock(scalingEngineUri)
          .get(/\/v1\/apps\/.+\/scaling_histories/)
          .reply(200, histories);
        request(app)
          .get("/v1/apps/12345/scaling_histories")
          .query({ "start-time": 100, "end-time": 500, "order": "desc", "page": 1, "results-per-page": 2 })
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
        nock(scalingEngineUri)
          .get(/\/v1\/apps\/.+\/scaling_histories/)
          .reply(200, histories);
        request(app)
          .get("/v1/apps/12345/scaling_histories")
          .query({ "start-time": 100, "end-time": 500, "order": "desc", "page": 2, "results-per-page": 2 })
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
        nock(scalingEngineUri)
          .get(/\/v1\/apps\/.+\/scaling_histories/)
          .reply(200, histories);
        request(app)
          .get("/v1/apps/12345/scaling_histories")
          .query({ "start-time": 100, "end-time": 500, "order": "desc", "page": 3, "results-per-page": 2 })
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
        nock(scalingEngineUri)
          .get(/\/v1\/apps\/.+\/scaling_histories/)
          .reply(200, histories);
        request(app)
          .get("/v1/apps/12345/scaling_histories")
          .query({ "start-time": 100, "end-time": 500, "order": "desc", "page": 4, "results-per-page": 2 })
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
