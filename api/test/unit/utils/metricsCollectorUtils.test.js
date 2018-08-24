'use strict';

var request = require('supertest');
var expect = require('chai').expect;
var fs = require('fs');
var path = require('path');
var settings = require(path.join(__dirname, '../../../lib/config/setting.js'))((JSON.parse(
  fs.readFileSync(path.join(__dirname, '../../../config/settings.json'), 'utf8'))));
var logger = require('../../../lib/log/logger');
var nock = require('nock');
var tlsOptions = {
  key: fs.readFileSync(settings.tls.keyFile),
  cert: fs.readFileSync(settings.tls.certFile),
  ca: fs.readFileSync(settings.tls.caCertFile)
}
var metricsCollectorUtils = require('../../../lib/utils/metricsCollectorUtils')(settings.metricsCollector);
var metricsCollectorUri = settings.metricsCollector.uri;

var theAppId = 'an-app-id';
var metricType = 'memoryused';
describe('metricsCollector Utility functions', function() {
  context('Get metrics', function() {
    beforeEach(function() {
      nock.cleanAll();
    });
    context('all parameters are valid', function() {
      var metrics = [
        { "app_id": theAppId, "timestamp": 100, "instance_index": 0, "collected_at": 0, "name": metricType, "unit": "megabytes", "value": "200" },
        { "app_id": theAppId, "timestamp": 110, "instance_index": 1, "collected_at": 1, "name": metricType, "unit": "megabytes", "value": "200" },
        { "app_id": theAppId, "timestamp": 150, "instance_index": 0, "collected_at": 0, "name": metricType, "unit": "megabytes", "value": "200" },
        { "app_id": theAppId, "timestamp": 170, "instance_index": 1, "collected_at": 1, "name": metricType, "unit": "megabytes", "value": "200" },
        { "app_id": theAppId, "timestamp": 200, "instance_index": 0, "collected_at": 0, "name": metricType, "unit": "megabytes", "value": "200" }
      ];
      context("instanceIndex is not provided", function() {
        it('should get the metrics of all instances', function(done) {
          nock(metricsCollectorUri)
            .get(/\/v1\/apps\/.+\/metric_histories/)
            .reply(200, metrics);
          var mockParameters = {
            appId: theAppId,
            metricType: metricType,
            startTime: 100,
            endTime: 200,
            order: 'desc'
          };
          metricsCollectorUtils.getMetricHistory(mockParameters, function(error, result) {
            expect(error).to.be.null;
            expect(result.statusCode).to.equal(200);
            expect(result.body).to.deep.equal(metrics);
            done();
          });
        });
      });
      context("instanceIndex is provided", function() {
        it('should get the metrics of the provided instance', function(done) {
          nock(metricsCollectorUri)
            .get(/\/v1\/apps\/.+\/metric_histories/)
            .reply(200, [metrics[0], metrics[2], metrics[4]]);
          var mockParameters = {
            appId: theAppId,
            metricType: metricType,
            instanceIndex: 0,
            startTime: 100,
            endTime: 200,
            order: 'desc'
          };
          metricsCollectorUtils.getMetricHistory(mockParameters, function(error, result) {
            expect(error).to.be.null;
            expect(result.statusCode).to.equal(200);
            expect(result.body).to.deep.equal([metrics[0], metrics[2], metrics[4]]);
            done();
          });
        });
      });

    });

    context('there is error when requesting metricsCollector ', function() {
      var mockError = {
        'message': 'Error in requests metricsCollector',
        'details': 'fake body'
      };
      it('should fail to get the metrics', function(done) {
        nock(metricsCollectorUri)
          .get(/\/v1\/apps\/.+\/metric_histories/)
          .replyWithError(mockError);
        var mockParameters = {
          appId: theAppId,
          metricType: metricType,
          instanceIndex: 0,
          startTime: 100,
          endTime: 200,
          order: 'desc'
        };
        metricsCollectorUtils.getMetricHistory(mockParameters, function(error, result) {
          expect(error).to.not.be.null;
          expect(error).to.deep.equal(mockError);
          done();
        });
      });
    });

    context('instanceindex is not a number', function() {
      var mockBody = { code: 'Bad-Request', message: 'Error parsing instanceIndex' };
      it('should fail to get the metrics', function(done) {
        nock(metricsCollectorUri)
          .get(/\/v1\/apps\/.+\/metric_histories/)
          .reply(400, mockBody);
        var mockParameters = {
          appId: theAppId,
          metricType: metricType,
          instanceIndex: 'not-integer',
          startTime: 100,
          endTime: 200,
          order: 'desc'
        };
        metricsCollectorUtils.getMetricHistory(mockParameters, function(error, result) {
          expect(error).to.not.be.null;
          expect(error).to.deep.equal({ statusCode: 400, message: 'Error parsing instanceIndex' })
          done();
        });
      });
    });

    context('instanceindex is smaller than 0', function() {
      var mockBody = { code: 'Bad-Request', message: 'InstanceIndex must be greater than or equal to 0' };
      it('should fail to get the metrics', function(done) {
        nock(metricsCollectorUri)
          .get(/\/v1\/apps\/.+\/metric_histories/)
          .reply(400, mockBody);
        var mockParameters = {
          appId: theAppId,
          metricType: metricType,
          instanceIndex: -1,
          startTime: 100,
          endTime: 200,
          order: 'desc'
        };
        metricsCollectorUtils.getMetricHistory(mockParameters, function(error, result) {
          expect(error).to.not.be.null;
          expect(error).to.deep.equal({ statusCode: 400, message: 'InstanceIndex must be greater than or equal to 0' })
          done();
        });
      });
    });

    context('start-time is not a number', function() {
      var mockBody = { code: 'Bad-Request', message: 'Error parsing start time' };
      it('should fail to get the metrics', function(done) {
        nock(metricsCollectorUri)
          .get(/\/v1\/apps\/.+\/metric_histories/)
          .reply(400, mockBody);
        var mockParameters = {
          appId: theAppId,
          metricType: metricType,
          instanceIndex: 0,
          startTime: 'not-integer',
          endTime: 200,
          order: 'desc'
        };
        metricsCollectorUtils.getMetricHistory(mockParameters, function(error, result) {
          expect(error).to.not.be.null;
          expect(error).to.deep.equal({ statusCode: 400, message: 'Error parsing start time' })
          done();
        });
      });
    });

    context('end-time is not a number', function() {
      var mockBody = { code: 'Bad-Request', message: 'Error parsing end time' };
      it('should fail to get the metrics', function(done) {
        nock(metricsCollectorUri)
          .get(/\/v1\/apps\/.+\/metric_histories/)
          .reply(400, mockBody);
        var mockParameters = {
          appId: theAppId,
          metricType: metricType,
          instanceIndex: 0,
          startTime: 100,
          endTime: 'not-integer',
          order: 'desc'
        };
        metricsCollectorUtils.getMetricHistory(mockParameters, function(error, result) {
          expect(error).to.not.be.null;
          expect(error).to.deep.equal({ statusCode: 400, message: 'Error parsing end time' })
          done();
        });
      });
    });

    context('order is not desc or asc', function() {
      var mockBody = { code: 'Bad-Request', message: 'Incorrect order parameter in query string, the value can only be desc or asc' };
      it('should fail to get the metrics', function(done) {
        nock(metricsCollectorUri)
          .get(/\/v1\/apps\/.+\/metric_histories/)
          .reply(400, mockBody);
        var mockParameters = {
          appId: theAppId,
          metricType: metricType,
          instanceIndex: 0,
          startTime: 100,
          endTime: 200,
          order: 'not-desc-asc'
        };
        metricsCollectorUtils.getMetricHistory(mockParameters, function(error, result) {
          expect(error).to.not.be.null;
          expect(error).to.deep.equal({ statusCode: 400, message: 'Incorrect order parameter in query string, the value can only be desc or asc' })
          done();
        });
      });
    });

    context('internal error in metricsCollector', function() {
      var mockBody = { code: 'Interal-Server-Error', message: 'Error getting metrics from database' };
      it('should fail to get the metrics', function(done) {
        nock(metricsCollectorUri)
          .get(/\/v1\/apps\/.+\/metric_histories/)
          .reply(500, mockBody);
        var mockParameters = {
          appId: theAppId,
          metricType: metricType,
          instanceIndex: 0,
          startTime: 100,
          endTime: 200,
          order: 'desc'
        };
        metricsCollectorUtils.getMetricHistory(mockParameters, function(error, result) {
          expect(error).to.not.be.null;
          expect(error).to.deep.equal({ statusCode: 500, message: 'Error getting metrics from database' })
          done();
        });
      });
    });

  });
});