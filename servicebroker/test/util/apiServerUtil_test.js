'use strict';

var request = require('supertest');
var expect = require('chai').expect;
var fs = require('fs');
var path = require('path');
var settings = require(path.join(__dirname, '../../lib/config/setting.js'))((JSON.parse(
    fs.readFileSync(path.join(__dirname, '../../config/settings.json'), 'utf8'))));
var nock = require('nock');
var apiServerUtils = require('../../lib/util/apiServerUtil')(settings);
var apiServerURI = settings.apiserver.uri;

describe('ApiServer Utility functions', function () {

    context('Attach policy', function () {
        it('should successfully attach policy', function (done) {
            nock(apiServerURI)
                .put('/v1/apps/12345/policy')
                .reply(201);
            apiServerUtils.attachPolicy('12345', { 'policy': 'valid_policy' }, function (error, response) {
                expect(error).to.be.null;
                expect(response.statusCode).to.equal(201);
                done();
            });
        });

        it('should fail to attach policy due to policy validation failure', function (done) {
            nock(apiServerURI)
                .put('/v1/apps/12345/policy')
                .reply(400);
            apiServerUtils.attachPolicy('12345', 'invalid_policy', function (error, response) {
                expect(error).to.be.null;
                expect(response.statusCode).to.equal(400);
                done();
            });
        });

        it('should fail to attach policy due to request timeout', function (done) {
            nock(apiServerURI)
                .put('/v1/apps/12345/policy')
                .socketDelay(20000)
                .reply(201);
            apiServerUtils.attachPolicy('12345', { 'policy': 'valid_policy' }, function (error) {
                expect(error).to.not.be.null;
                expect(error.code).to.equal('ESOCKETTIMEDOUT');
                done();
            });
        });

        it('should fail to attach policy due to internal server error in apiserver', function (done) {
            nock(apiServerURI)
                .put('/v1/apps/12345/policy')
                .reply(500);
            apiServerUtils.attachPolicy('12345', { 'policy': 'valid_policy' }, function (error, response) {
                expect(error).to.be.null;
                expect(response.statusCode).to.equal(500);
                done();
            });
        });
    });

    context('Detach policy', function () {
        it('should successfully detach policy', function (done) {
            nock(apiServerURI)
                .delete('/v1/apps/12345/policy')
                .reply(204);
            apiServerUtils.detachPolicy('12345', function (error, response) {
                expect(error).to.be.null;
                expect(response.statusCode).to.equal(204);
                done();
            });
        });

        it('should fail to detach policy due to request timeout', function (done) {
            nock(apiServerURI)
                .delete('/v1/apps/12345/policy')
                .socketDelay(20000)
                .reply(204);
            apiServerUtils.detachPolicy('12345', function (error) {
                expect(error).to.not.be.null;
                expect(error.code).to.equal('ESOCKETTIMEDOUT');
                done();
            });
        });

        it('should fail to detach policy due to internal server error in apiserver', function (done) {
            nock(apiServerURI)
                .delete('/v1/apps/12345/policy')
                .reply(500);
            apiServerUtils.detachPolicy('12345', function (error, response) {
                expect(error).to.be.null;
                expect(response.statusCode).to.equal(500);
                done();
            });
        });
    });

});

