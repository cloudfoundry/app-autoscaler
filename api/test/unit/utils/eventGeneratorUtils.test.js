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
var eventGeneratorUtils = require('../../../lib/utils/eventGeneratorUtils')(settings.eventGenerator);
var eventGeneratorUri = settings.eventGenerator.uri;

var theAppId = 'an-app-id';
var metricType = 'memoryused';
describe('eventGenerator Utility functions', function() {
  context('Get Aggregated metrics', function() {
    beforeEach(function() {
      nock.cleanAll();
    });
    context('all parameters are valid', function() {
      var aggregatedMetrics = [
        { "app_id": theAppId, "timestamp": 100,"name": "memoryused", "unit": "megabytes", "value": "200"},
        { "app_id": theAppId, "timestamp": 110,"name": "memoryused", "unit": "megabytes", "value": "200"},
        { "app_id": theAppId, "timestamp": 150,"name": "memoryused", "unit": "megabytes", "value": "200"},
        { "app_id": theAppId, "timestamp": 170,"name": "memoryused", "unit": "megabytes", "value": "200"},
        { "app_id": theAppId, "timestamp": 200,"name": "memoryused", "unit": "megabytes", "value": "200"}
      ];
      it('should get the aggregatedMetrics', function(done) {
        nock(eventGeneratorUri)
          .get(/\/v1\/apps\/.+\/aggregated_metric_histories/)
          .reply(200, aggregatedMetrics);
        var mockParameters = {
          appId: theAppId,
          metricType: metricType,
          startTime: 100,
          endTime: 200,
          order: 'desc'
        };
        eventGeneratorUtils.getAggregatedMetricHistory(mockParameters, function(error, result) {
          expect(error).to.be.null;
          expect(result.statusCode).to.equal(200);
          expect(result.body).to.deep.equal(aggregatedMetrics);
          done();
        });
      });
    });

    context('there is error when requesting eventGenerator ', function() {
      var mockError = {
        'message': 'Error in requests eventGenerator',
        'details': 'fake body'
      };
      it('should fail to get the aggregatedMetrics', function(done) {
        nock(eventGeneratorUri)
          .get(/\/v1\/apps\/.+\/aggregated_metric_histories/)
          .replyWithError(mockError);
        var mockParameters = {
          appId: theAppId,
          metricType: metricType,
          startTime: 100,
          endTime: 200,
          order: 'desc'
        };
        eventGeneratorUtils.getAggregatedMetricHistory(mockParameters, function(error, result) {
          expect(error).to.not.be.null;
          expect(error).to.deep.equal(mockError);
          done();
        });
      });
    });

    context('start-time is not number', function() {
      var mockBody = { code: 'Bad-Request', message: 'Error parsing start time' };
      it('should fail to get the aggregated_metrics', function(done) {
        nock(eventGeneratorUri)
          .get(/\/v1\/apps\/.+\/aggregated_metric_histories/)
          .reply(400, mockBody);
        var mockParameters = {
          appId: theAppId,
          metricType: metricType,
          startTime: 'not-integer',
          endTime: 200,
          order: 'desc'
        };
        eventGeneratorUtils.getAggregatedMetricHistory(mockParameters, function(error, result) {
          expect(error).to.not.be.null;
          expect(error).to.deep.equal({ statusCode: 400, message: 'Error parsing start time' })
          done();
        });
      });
    });

    context('end-time is not number', function() {
      var mockBody = { code: 'Bad-Request', message: 'Error parsing end time' };
      it('should fail to get the aggregatedMetrics', function(done) {
        nock(eventGeneratorUri)
          .get(/\/v1\/apps\/.+\/aggregated_metric_histories/)
          .reply(400, mockBody);
        var mockParameters = {
          appId: theAppId,
          metricType: metricType,
          startTime: 100,
          endTime: 'not-integer',
          order: 'desc'
        };
        eventGeneratorUtils.getAggregatedMetricHistory(mockParameters, function(error, result) {
          expect(error).to.not.be.null;
          expect(error).to.deep.equal({ statusCode: 400, message: 'Error parsing end time' })
          done();
        });
      });
    });

    context('order is not desc or asc', function() {
      var mockBody = { code: 'Bad-Request', message: 'Incorrect order parameter in query string, the value can only be desc or asc' };
      it('should fail to get the aggregated_metrics', function(done) {
        nock(eventGeneratorUri)
          .get(/\/v1\/apps\/.+\/aggregated_metric_histories/)
          .reply(400, mockBody);
        var mockParameters = {
          appId: theAppId,
          metricType: metricType,
          startTime: 100,
          endTime: 200,
          order: 'not-desc-asc'
        };
        eventGeneratorUtils.getAggregatedMetricHistory(mockParameters, function(error, result) {
          expect(error).to.not.be.null;
          expect(error).to.deep.equal({ statusCode: 400, message: 'Incorrect order parameter in query string, the value can only be desc or asc' })
          done();
        });
      });
    });

    context('internal error in eventGenerator', function() {
      var mockBody = { code: 'Interal-Server-Error', message: 'Error getting aggregated_metrics from database' };
      it('should fail to get the aggregatedMetrics', function(done) {
        nock(eventGeneratorUri)
          .get(/\/v1\/apps\/.+\/aggregated_metric_histories/)
          .reply(500, mockBody);
        var mockParameters = {
          appId: theAppId,
          metricType: metricType,
          startTime: 100,
          endTime: 200,
          order: 'desc'
        };
        eventGeneratorUtils.getAggregatedMetricHistory(mockParameters, function(error, result) {
          expect(error).to.not.be.null;
          expect(error).to.deep.equal({ statusCode: 500, message: 'Error getting aggregated_metrics from database' })
          done();
        });
      });
    });

  });
});
