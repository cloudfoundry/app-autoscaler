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
var scalingEngineUtils = require('../../../lib/utils/scalingEngineUtils')(settings.scalingEngine);
var scalingEngineUri = settings.scalingEngine.uri;

var theAppId = 'an-app-id'
describe('ScalingEngine Utility functions', function() {
  context('Get scaling histories', function() {
    beforeEach(function() {
      nock.cleanAll();
    });
    context('all parameters are valid', function() {
      var histories = [
        { 'app_id': theAppId, 'timestamp': 150, 'scaling_type': 1, 'status': 1, 'old_instances': 2, 'new_instances': 4, 'reason': 'a reason', 'message': '', 'error': '' },
        { 'app_id': theAppId, 'timestamp': 120, 'scaling_type': 0, 'status': 0, 'old_instances': 2, 'new_instances': 4, 'reason': 'a reason', 'message': '', 'error': '' },
      ]
      it('should get the scaling histories', function(done) {
        nock(scalingEngineUri)
          .get(/\/v1\/apps\/.+\/scaling_histories/)
          .reply(200, histories);
        var mockParameters = {
          appId: theAppId,
          startTime: 100,
          endTime: 200,
          order: 'desc'
        };
        scalingEngineUtils.getScalingHistory(mockParameters, function(error, result) {
          expect(error).to.be.null;
          expect(result.statusCode).to.equal(200);
          expect(result.body).to.deep.equal(histories);
          done();
        });
      });
    });

    context('there is error when requesting scalingEngine ', function() {
      var mockError = {
        'message': 'Error in requests scalingEngine',
        'details': 'fake body'
      };
      it('should fail to get the scaling histories', function(done) {
        nock(scalingEngineUri)
          .get(/\/v1\/apps\/.+\/scaling_histories/)
          .replyWithError(mockError);
        var mockParameters = {
          appId: theAppId,
          startTime: 100,
          endTime: 200,
          order: 'desc'
        };
        scalingEngineUtils.getScalingHistory(mockParameters, function(error, result) {
          expect(error).to.not.be.null;
          expect(error).to.deep.equal(mockError);
          done();
        });
      });
    });

    context('start-time is not number', function() {
      var mockBody = { code: 'Bad-Request', message: 'Error parsing start time' };
      it('should fail to get the scaling histories', function(done) {
        nock(scalingEngineUri)
          .get(/\/v1\/apps\/.+\/scaling_histories/)
          .reply(400, mockBody);
        var mockParameters = {
          appId: theAppId,
          startTime: 'not-integer',
          endTime: 200,
          order: 'desc'
        };
        scalingEngineUtils.getScalingHistory(mockParameters, function(error, result) {
          expect(error).to.not.be.null;
          expect(error).to.deep.equal({ statusCode: 400, message: 'Error parsing start time' })
          done();
        });
      });
    });

    context('end-time is not number', function() {
      var mockBody = { code: 'Bad-Request', message: 'Error parsing end time' };
      it('should fail to get the scaling histories', function(done) {
        nock(scalingEngineUri)
          .get(/\/v1\/apps\/.+\/scaling_histories/)
          .reply(400, mockBody);
        var mockParameters = {
          appId: theAppId,
          startTime: 100,
          endTime: 'not-integer',
          order: 'desc'
        };
        scalingEngineUtils.getScalingHistory(mockParameters, function(error, result) {
          expect(error).to.not.be.null;
          expect(error).to.deep.equal({ statusCode: 400, message: 'Error parsing end time' })
          done();
        });
      });
    });

    context('order is not desc or asc', function() {
      var mockBody = { code: 'Bad-Request', message: 'Incorrect order parameter in query string, the value can only be desc or asc' };
      it('should fail to get the scaling histories', function(done) {
        nock(scalingEngineUri)
          .get(/\/v1\/apps\/.+\/scaling_histories/)
          .reply(400, mockBody);
        var mockParameters = {
          appId: theAppId,
          startTime: 100,
          endTime: 200,
          order: 'not-desc-asc'
        };
        scalingEngineUtils.getScalingHistory(mockParameters, function(error, result) {
          expect(error).to.not.be.null;
          expect(error).to.deep.equal({ statusCode: 400, message: 'Incorrect order parameter in query string, the value can only be desc or asc' })
          done();
        });
      });
    });

    context('internal error in scalingEngine', function() {
      var mockBody = { code: 'Interal-Server-Error', message: 'Error getting scaling histories from database' };
      it('should fail to get the scaling histories', function(done) {
        nock(scalingEngineUri)
          .get(/\/v1\/apps\/.+\/scaling_histories/)
          .reply(500, mockBody);
        var mockParameters = {
          appId: theAppId,
          startTime: 100,
          endTime: 200,
          order: 'desc'
        };
        scalingEngineUtils.getScalingHistory(mockParameters, function(error, result) {
          expect(error).to.not.be.null;
          expect(error).to.deep.equal({ statusCode: 500, message: 'Error getting scaling histories from database' })
          done();
        });
      });
    });

  });
});
