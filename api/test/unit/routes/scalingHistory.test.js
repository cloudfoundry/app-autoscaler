'use strict';
process.env.NODE_TLS_REJECT_UNAUTHORIZED = '0';
var request = require('supertest');
var expect = require('chai').expect;
var fs = require('fs');
var path = require('path');
var settings = require(path.join(__dirname, '../../../lib/config/setting.js'))((JSON.parse(
  fs.readFileSync(path.join(__dirname, '../../../config/settings.json'), 'utf8'))));
var API = require('../../../app.js');
var app;
var scalingHistory = require('../../../lib/models')(settings.db).scalinghistory;

describe('ScalingHistory', function() {
  var testAppId;
  before(function() {
    app = API(path.join(__dirname, '../../../config/settings.json'));
  })
  after(function(done) {
    app.close(done);
  })
  beforeEach(function(done) {
    testAppId = "theTestAppId";
    scalingHistory.truncate().then(function() { done(); });
  });
  describe("getScalingHistoryList", function() {

    context("validate input parameters", function() {
      context("start-time", function() {
        context("when start-time is not provided", function() {
          it("should return 400", function(done) {
            request(app)
              .get('/v1/apps/' + testAppId + '/scaling_history')
              .query({ "end-time": 2000, "page": 0, "results-per-page": 10 })
              .end(function(error, result) {
                expect(result.statusCode).to.equal(400);
                expect(result.body.description).to.equal("start-time is required");
                done();
              });
          });
        });

        context("when start-time is not integer or integer-format string", function() {
          it("should return 400", function(done) {
            request(app)
              .get('/v1/apps/' + testAppId + '/scaling_history')
              .query({ "start-time": "notInteger", "end-time": 2000, "page": 0, "results-per-page": 10 })
              .end(function(error, result) {
                expect(result.statusCode).to.equal(400);
                expect(result.body.description).to.equal("start-time must be an integer");
                done();
              });
          });
        });
      });

      context("end-time", function() {

        context("when end-time is not provided", function() {
          it("should return 400", function(done) {
            request(app)
              .get('/v1/apps/' + testAppId + '/scaling_history')
              .query({ "start-time": 0, "page": 0, "results-per-page": 10 })
              .end(function(error, result) {
                expect(result.statusCode).to.equal(400);
                expect(result.body.description).to.equal("end-time is required");
                done();
              });
          });
        });

        context("when end-time is not an integer or integer-format string", function() {
          it("should return 400", function(done) {
            request(app)
              .get('/v1/apps/' + testAppId + '/scaling_history')
              .query({ "end-time": "notInteger", "start-time": 0, "page": 0, "results-per-page": 10 })
              .end(function(error, result) {
                expect(result.statusCode).to.equal(400);
                expect(result.body.description).to.equal("end-time must be an integer");
                done();
              });
          });
        });
      });

      context("when start-time is greater than end-time", function() {
        it("should return 400", function(done) {
          request(app)
            .get('/v1/apps/' + testAppId + '/scaling_history')
            .query({ "start-time": 1, "end-time": 0, "page": 0, "results-per-page": 10 })
            .end(function(error, result) {
              expect(result.statusCode).to.equal(400);
              expect(result.body.description).to.equal("start-time must be smaller than end-time");
              done();
            });
        });
        it("should return 400", function(done) {
          request(app)
            .get('/v1/apps/' + testAppId + '/scaling_history')
            .query({ "start-time": "1", "end-time": "0", "page": 0, "results-per-page": 10 })
            .end(function(error, result) {
              expect(result.statusCode).to.equal(400);
              expect(result.body.description).to.equal("start-time must be smaller than end-time");
              done();
            });
        });
      });
      context("page", function() {

        context("when page is not provided", function() {
          it("should return 400", function(done) {
            request(app)
              .get('/v1/apps/' + testAppId + '/scaling_history')
              .query({ "start-time": 0, "end-time": 1000, "results-per-page": 10 })
              .end(function(error, result) {
                expect(result.statusCode).to.equal(400);
                expect(result.body.description).to.equal("page is required");
                done();
              });
          });
        });

        context("when page is not an integer or integer-format string", function() {
          it("should return 400", function(done) {
            request(app)
              .get('/v1/apps/' + testAppId + '/scaling_history')
              .query({ "end-time": 1, "start-time": 0, "page": "notInteger", "results-per-page": 10 })
              .end(function(error, result) {
                expect(result.statusCode).to.equal(400);
                expect(result.body.description).to.equal("page must be an integer");
                done();
              });
          });
        });
      });

      context("results-per-page", function() {

        context("when results-per-page is not provided", function() {
          it("should return 400", function(done) {
            request(app)
              .get('/v1/apps/' + testAppId + '/scaling_history')
              .query({ "start-time": 0, "end-time": 1000, "page": 1 })
              .end(function(error, result) {
                expect(result.statusCode).to.equal(400);
                expect(result.body.description).to.equal("results-per-page is required");
                done();
              });
          });
        });

        context("when results-per-page is not an integer or integer-format string", function() {
          it("should return 400", function(done) {
            request(app)
              .get('/v1/apps/' + testAppId + '/scaling_history')
              .query({ "end-time": 1, "start-time": 0, "page": 0, "results-per-page": "notInteger" })
              .end(function(error, result) {
                expect(result.statusCode).to.equal(400);
                expect(result.body.description).to.equal("results-per-page must be an integer");
                done();
              });
          });
        });
      });


    });
    context("get list", function() {

      beforeEach(function(done) {
        scalingHistory.bulkCreate(
            [
              { app_guid: testAppId, timestamp: 1000, scaling_type: 1, status: 0, old_instances: 4, new_instances: 3, reason: "", message: "", error: "" },
              { app_guid: testAppId, timestamp: 2000, scaling_type: 1, status: 0, old_instances: 3, new_instances: 2, reason: "", message: "", error: "" },
              { app_guid: testAppId, timestamp: 3000, scaling_type: 1, status: 0, old_instances: 2, new_instances: 1, reason: "", message: "", error: "" },
              { app_guid: testAppId, timestamp: 4000, scaling_type: 1, status: 0, old_instances: 4, new_instances: 3, reason: "", message: "", error: "" },
              { app_guid: testAppId, timestamp: 5000, scaling_type: 1, status: 0, old_instances: 3, new_instances: 2, reason: "", message: "", error: "" },
              { app_guid: testAppId, timestamp: 6000, scaling_type: 1, status: 0, old_instances: 2, new_instances: 1, reason: "", message: "", error: "" },
              { app_guid: testAppId, timestamp: 7000, scaling_type: 1, status: 0, old_instances: 4, new_instances: 3, reason: "", message: "", error: "" },
              { app_guid: testAppId, timestamp: 8000, scaling_type: 1, status: 0, old_instances: 3, new_instances: 2, reason: "", message: "", error: "" },
              { app_guid: testAppId, timestamp: 9000, scaling_type: 1, status: 0, old_instances: 2, new_instances: 1, reason: "", message: "", error: "" },
              { app_guid: testAppId, timestamp: 10000, scaling_type: 1, status: 0, old_instances: 2, new_instances: 1, reason: "", message: "", error: "" }
            ])
          .then(function(response) {
            done();
          });
      });

      context("Filtered by start-time", function() {
        it('should return all scalingHistory list', function(done) {
          request(app)
            .get('/v1/apps/' + testAppId + '/scaling_history')
            .query({ "start-time": 5000, "end-time": 99999, "page": 0, "results-per-page": 10 })
            .end(function(error, result) {
              expect(result.statusCode).to.equal(200);
              expect(result.body.total_results).to.equal(6);
              expect(result.body.total_pages).to.equal(1);
              expect(result.body.page).to.equal(0);
              expect(result.body.resources.length).to.equal(6);
              done();
            });
        });
      });

      context("Filtered by end-time", function() {
        it('should return all scalingHistory list', function(done) {
          request(app)
            .get('/v1/apps/' + testAppId + '/scaling_history')
            .query({ "start-time": 0, "end-time": 5000, "page": 0, "results-per-page": 10 })
            .end(function(error, result) {
              expect(result.statusCode).to.equal(200);
              expect(result.body.total_results).to.equal(5);
              expect(result.body.total_pages).to.equal(1);
              expect(result.body.page).to.equal(0);
              expect(result.body.resources.length).to.equal(5);
              done();
            });
        });
      });

      context("Filtered by page", function() {

        it('should get 1st page', function(done) {
          request(app)
            .get('/v1/apps/' + testAppId + '/scaling_history')
            .query({ "start-time": 0, "end-time": 99999, "page": 0, "results-per-page": 2 })
            .end(function(error, result) {
              expect(result.statusCode).to.equal(200);
              expect(result.body.total_results).to.equal(10);
              expect(result.body.total_pages).to.equal(5);
              expect(result.body.page).to.equal(0);
              expect(result.body.resources.length).to.equal(2);
              expect(result.body.resources[0].timestamp).to.equal(10000);
              expect(result.body.resources[1].timestamp).to.equal(9000);
              done();
            });
        });

        it('should return the 2nd page', function(done) {
          request(app)
            .get('/v1/apps/' + testAppId + '/scaling_history')
            .query({ "start-time": 0, "end-time": 99999, "page": 1, "results-per-page": 2 })
            .end(function(error, result) {
              expect(result.statusCode).to.equal(200);
              expect(result.body.total_results).to.equal(10);
              expect(result.body.total_pages).to.equal(5);
              expect(result.body.page).to.equal(1);
              expect(result.body.resources.length).to.equal(2);
              expect(result.body.resources[0].timestamp).to.equal(8000);
              expect(result.body.resources[1].timestamp).to.equal(7000);
              done();
            });
        });
      });

      context("Filtered by results-per-page", function() {

        it('should get 1st page', function(done) {
          request(app)
            .get('/v1/apps/' + testAppId + '/scaling_history')
            .query({ "start-time": 0, "end-time": 99999, "page": 0, "results-per-page": 9 })
            .end(function(error, result) {
              expect(result.statusCode).to.equal(200);
              expect(result.body.total_results).to.equal(10);
              expect(result.body.total_pages).to.equal(2);
              expect(result.body.page).to.equal(0);
              expect(result.body.resources.length).to.equal(9);
              done();
            });
        });

        it('should return the 2nd page', function(done) {
          request(app)
            .get('/v1/apps/' + testAppId + '/scaling_history')
            .query({ "start-time": 0, "end-time": 99999, "page": 1, "results-per-page": 9 })
            .end(function(error, result) {
              expect(result.statusCode).to.equal(200);
              expect(result.body.total_results).to.equal(10);
              expect(result.body.total_pages).to.equal(2);
              expect(result.body.page).to.equal(1);
              expect(result.body.resources.length).to.equal(1);
              done();
            });
        });
      });

    });
  });
});
