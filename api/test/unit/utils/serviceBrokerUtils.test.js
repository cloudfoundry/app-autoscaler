'use strict';

var request = require('supertest');
var expect = require('chai').expect;
var fs = require('fs');
var path = require('path');
var settings = require(path.join(__dirname, '../../../lib/config/setting.js'))((JSON.parse(
  fs.readFileSync(path.join(__dirname, '../../../config/settings.json'), 'utf8'))));
var logger = require('../../../lib/log/logger');
var nock = require('nock');

var serviceBrokerUtils = require('../../../lib/utils/serviceBrokerUtils')(settings.serviceOffering.serviceBroker);
var serviceBrokerUri = settings.serviceOffering.serviceBroker.uri;
var theAppId = 'an-app-id';

describe('serviceBroker Utility functions', function() {
  context('Check binding', function() {
    beforeEach(function() {
      nock.cleanAll();
    });
    context('serviceBroker returns 200 with binding', function() {
      it('should return 200', function(done) {
        nock(serviceBrokerUri)
          .get(/\/v1\/apps\/.+\/service_bindings/)
          .reply(200, {
            "binding": {
              "bindingId": "an-binding-id",
              "appId": "an-app-id",
              "serviceInstanceId": "an-service-instance-id"
            }
          });
        var mockParameters = {
          appId: theAppId
        };
        serviceBrokerUtils.checkBinding(mockParameters, function(error, result) {
          expect(error).to.be.null;
          expect(result.statusCode).to.equal(200);
          done();
        });
      });
    });
    context('serviceBroker returns 200 with null binding', function() {
      it('should return 404', function(done) {
        nock(serviceBrokerUri)
          .get(/\/v1\/apps\/.+\/service_bindings/)
          .reply(200, { "binding": null });
        var mockParameters = {
          appId: theAppId
        };
        serviceBrokerUtils.checkBinding(mockParameters, function(error, result) {
          expect(error).to.be.null;
          expect(result.statusCode).to.equal(404);
          done();
        });
      });
    });
    context('there is error when requesting serviceBroker ', function() {
      var mockError = {
        'message': 'Error in requests serviceBroker',
        'details': 'fake body'
      };
      it('should return 500', function(done) {
        nock(serviceBrokerUri)
          .get(/\/v1\/apps\/.+\/service_bindings/)
          .replyWithError(mockError);
        var mockParameters = {
          appId: theAppId
        };
        serviceBrokerUtils.checkBinding(mockParameters, function(error, result) {
          expect(error).to.not.be.null;
          expect(error.statusCode).to.equal(500);
          expect(error).to.deep.equal(mockError);
          done();
        });
      });
    });

    

    context('serviceBroker returns 500 ', function() {
      it('should return 500', function(done) {
        nock(serviceBrokerUri)
          .get(/\/v1\/apps\/.+\/service_bindings/)
          .reply(500);
        var mockParameters = {
          appId: theAppId
        };
        serviceBrokerUtils.checkBinding(mockParameters, function(error, result) {
          expect(error).to.not.be.null;
          expect(error.statusCode).to.equal(500);
          done();
        });
      });
    });

    context('serviceBroker returns 400', function() {
      it('should return 400', function(done) {
        nock(serviceBrokerUri)
          .get(/\/v1\/apps\/.+\/service_bindings/)
          .reply(400, {});
        var mockParameters = {
          appId: theAppId
        };
        serviceBrokerUtils.checkBinding(mockParameters, function(error, result) {
          expect(error).to.not.be.null;
          expect(error.statusCode).to.equal(400);
          done();
        });
      });
    });
  });
});