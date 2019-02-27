'use strict';

var expect = require('chai').expect;
var fs = require('fs');
var path = require('path');
var NodeCache = require('node-cache');
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
var HttpStatus = require('http-status-codes');
var models = require('../../../lib/models')(settings.db);
var credHelper = require('../../../lib/routes/credentialHelper')(models,credentialCache,settings.cacheTTL);
var credentials = models.credentials;

describe('Credential Management helper ', function() {

  before(function() {
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

  beforeEach(function(done) {
    credentials.truncate({
      cascade: true
    }).then(function(result) {
      done();
    });
  });

  context('Create credentials', function() {
    it('should create credentials for app id 12345', function(done) {
      var mockRequest = {
        body: {},
        params: {
          'app_id': '12345'
        }
      };

      credHelper.createOrUpdateCredentials(mockRequest, function(error, result) {
        expect(result.statusCode).to.equal(HttpStatus.CREATED);
        expect(error).to.be.null;
        done();
      });
    });
    
    it('should create credentials for app id 12345', function(done) {
      var mockRequest = {
        body: {},
        params: {
          'app_id': '12345'
        }
      };
      credHelper.createOrUpdateCredentials(mockRequest, function(error, result) {
        expect(result.statusCode).to.equal(HttpStatus.CREATED);
        expect(error).to.be.null;
        done();
      });
    });
  });

  context('Update credentials', function() {
    var mockRequest = {
      body: {},
      params: {
        'app_id': '12345'
      }
    };

    beforeEach(function(done) {
      credHelper.createOrUpdateCredentials(mockRequest, function(error, result) {
        expect(result.statusCode).to.equal(HttpStatus.CREATED);
        expect(error).to.be.null;
        done();
      });
    });

    it('should update credentials for app id 12345', function(done) {
      credHelper.createOrUpdateCredentials(mockRequest, function(error, result) {
        expect(result.statusCode).to.equal(HttpStatus.OK);
        expect(error).to.be.null;
        done();
      });
    });
  });

  context('Delete credentials', function() {
    var mockRequest = {
      body: {},
      params: {
        'app_id': '12345'
      }
    };

    beforeEach(function(done) {
      credHelper.createOrUpdateCredentials(mockRequest, function(error, result) {
        expect(result.statusCode).to.equal(HttpStatus.CREATED);
        expect(error).to.be.null;
        done();
      });
    });

    it('should delete credentials for app id 12345', function(done) {
      credHelper.deleteCredentials(mockRequest, function(error, result) {
        expect(result.statusCode).to.equal(HttpStatus.OK);
        expect(error).to.be.null;
        done();
      });
    });

    it('should return 404 while deleting non-existing credentials for app id 123456', function(done) {
      var newMockRequest = {
        body: {},
        params: {
          'app_id': '123456'
        }
      };
      credHelper.deleteCredentials(newMockRequest, function(error, result) {
        expect(error.statusCode).to.equal(HttpStatus.NOT_FOUND);
        expect(error.message).to.equal('No custom metrics credentials exists with application');
        done();
      });
    });
  });

  context('Validate credentials', function() {
    var mockCreateRequest = {
      body: {},
      params: {
        'app_id': '12345'
      }
    };

    var username;
    var password;

    beforeEach(function(done) {
      credHelper.createOrUpdateCredentials(mockCreateRequest, function(error, result) {
        expect(result.statusCode).to.equal(HttpStatus.CREATED);
        expect(error).to.be.null;
        username = result.username;
        password = result.password;
        done();
      });
    });

    it('should validate credentials for app id 12345', function(done) {
      var mockRequest = {
        body: {},
        params: {
          'app_id': '12345'
        },
        query: {
          'username': username,
          'password': password
        }
      };
      credHelper.validateCredentials(mockRequest, function(error, result) {
        expect(result.statusCode).to.equal(HttpStatus.OK);
        expect(error).to.be.null;
        expect(result.isValid).to.equal(true);
        done();
      });
    });

    it('should invalidate wrong username credentials for app id 12345', function(done) {
      var mockRequest = {
        body: {},
        params: {
          'app_id': '12345'
        },
        query: {
          'username': 'username1',
          'password': password
        }
      };
      credHelper.validateCredentials(mockRequest, function(error, result) {
        expect(result.statusCode).to.equal(HttpStatus.OK);
        expect(error).to.be.null;
        expect(result.isValid).to.equal(false);
        done();
      });
    });
  });
});
