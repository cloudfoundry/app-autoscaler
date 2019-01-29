'use strict';

process.env.NODE_TLS_REJECT_UNAUTHORIZED = '0';
var request = require('supertest');
var NodeCache = require('node-cache');
var expect = require('chai').expect;
var fs = require('fs');
var path = require('path');
var settings = require(path.join(__dirname, '../../../lib/config/setting.js'))((JSON.parse(
    fs.readFileSync(path.join(__dirname, '../../../config/settings.json'), 'utf8'))));
var relativePath = path.relative(process.cwd(), path.join(__dirname, '../../../../test-certs'));
var testSetting = require(path.join(__dirname, '../test.helper.js'))(relativePath, settings);
var API = require('../../../app.js');
var app;
var publicApp;
var healthApp;
var servers;
var credentialCache = new NodeCache();
var models = require('../../../lib/models')(settings.db);
var credentials = models.credentials;


describe('Custom Metrics Credential Management', function() {
  before(function(done) {
    servers = API(testSetting, credentialCache, function() {});
    app = servers.internalServer;
    publicApp = servers.publicServer;
    healthApp = servers.healthServer;
    done();
  });
  after(function(done) {
    app.close(function() {
      publicApp.close(function(){
        healthApp.close(done);
      });
    });
  })

  beforeEach(function(done) {
    credentials.truncate({
      cascade: true
    }).then(function(result) {
      done();
    });
  });

  context('create credentials when credentials does not exists', function() {
    it('should return 201', function(done) {
      request(app)
        .post('/v1/apps/12345/credentials')
        .end(function(error, result) {
          expect(error).to.equal(null);
          expect(result.statusCode).to.equal(201);
          credentials.count({
            where: {
              id: '12345'
            }
          }).then(function(countRes) {
            expect(countRes).to.equal(1);
            done();
          });
        });
    });
  });

  context('update credentials when credentials already exists', function() {
    beforeEach(function(done) {
      request(app)
      .post('/v1/apps/12345/credentials')
      .end(function(error, result) {
        expect(error).to.equal(null);
        expect(result.statusCode).to.equal(201);
        done();
      });
    });
    it('should return 200', function(done) {
      request(app)
      .post('/v1/apps/12345/credentials')
      .end(function(error, result) {
        expect(error).to.equal(null);
        expect(result.statusCode).to.equal(200);
        credentials.count({
          where: {
            id: '12345'
          }
        }).then(function(countRes) {
          expect(countRes).to.equal(1);
          done();
        });
      });
    });
  });

  context('delete credentials when credentials does not exists', function() {
    it('should return 404', function(done) {
      request(app)
      .delete('/v1/apps/12345/credentials')
      .end(function(error, result) {
        expect(error).to.equal(null);
        expect(result.statusCode).to.equal(404);
        done();
      });
    });
  });

  context('delete credentials when credentials exists', function() {
    beforeEach(function(done) {
      request(app)
      .post('/v1/apps/12345/credentials')
      .end(function(error, result) {
        done();
      });
    });
    it('should return 200', function(done) {
      request(app)
      .delete('/v1/apps/12345/credentials')
      .end(function(error, result) {
        expect(error).to.equal(null);
        expect(result.statusCode).to.equal(200);
        credentials.count({
          where: {
            id: '12345'
          }
        }).then(function(countRes) {
          expect(countRes).to.equal(0);
          done();
        });
      });
    });
  });

  context('validate credentials', function() {
    var username, password;
    beforeEach(function(done) {
      request(app)
      .post('/v1/apps/12345/credentials')
      .end(function(error, result) {
        username = result.body.username;
        password = result.body.password;
        expect(error).to.equal(null);
        expect(result.statusCode).to.equal(201);
        credentialCache.flushAll();
        done();
      });
    });
    it('should return 200 while returning non-cached credentials', function(done) {
      request(app)
      .post('/v1/apps/12345/credentials/validate')
      .query({
        'username': username,
        'password': password
      })
      .end(function(error, result) {
        expect(error).to.equal(null);
        expect(result.statusCode).to.equal(200);
        expect(result.body.isValid).to.equal(true);
        done();
      });
    });
    it('should return 200 while returning cached credentials', function(done) {
      request(app)
      .post('/v1/apps/12345/credentials/validate')
      .query({
        'username': username,
        'password': password
      })
      .end(function(error, result) {
        request(app)
        .post('/v1/apps/12345/credentials/validate')
        .query({
          'username': username,
          'password': password
        })
        .end(function(error, result) {
          expect(error).to.equal(null);
          expect(result.statusCode).to.equal(200);
          expect(result.body.isValid).to.equal(true);
          expect(credentialCache.get('12345')).to.not.be.undefined;
          done();
        });
      });
    });

    it('should return 200 for invalid username', function(done) {
      request(app)
      .post('/v1/apps/12345/credentials/validate')
      .query({
        'username': 'username1',
        'password': password
      })
      .end(function(error, result) {
        expect(error).to.equal(null);
        expect(result.statusCode).to.equal(200);
        expect(result.body.isValid).to.equal(false);
        done();
      });
    });

    it('should return 404 if credential does not exist', function(done) {
      request(app)
      .post('/v1/apps/123456/credentials/validate')
      .query({
        'username': username,
        'password': password
      })
      .end(function(error, result) {
        expect(error).to.equal(null);
        expect(result.statusCode).to.equal(404);
        done();
      });
    });

    it('should return 500 if query parameters not available', function(done) {
      request(app)
      .post('/v1/apps/12345/credentials/validate')
      .end(function(error, result) {
        expect(error).to.equal(null);
        expect(result.statusCode).to.equal(500);
        expect(result.body.error).to.equal('insufficient query parameters');
        done();
      });
    });

    it('should return 500 if required parameter not sent', function(done) {
      request(app)
      .post('/v1/apps/12345/credentials/validate')
      .query({
        'username': username
      })
      .end(function(error, result) {
        expect(error).to.equal(null);
        expect(result.statusCode).to.equal(500);
        expect(result.body.error).to.equal('insufficient query parameters');
        done();
      });
    });
  });
});
