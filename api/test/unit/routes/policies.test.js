'use strict';

var request = require('supertest');  
var expect = require('chai').expect;
var fs = require('fs');
var app = require('../../../app.js');
var logger = require('../../../lib/log/logger');

describe('Routing Policy Creation', function() {
  var fakePolicy;
  before(function(done) {
    this.policy = require('../../../lib/models')().policy_json;
    var _this = this;
    this.policy.sequelize.sync().then(function(success) {
      _this.policy.truncate().then(function(result) {
        fakePolicy = JSON.parse(fs.readFileSync(__dirname+'/../fakePolicy.json', 'utf8'));
        done();
      });  
    }, function(error) {
      logger.error('Failed to setup database for test');
      done(error);
    });
  });
  it('should create a policy for app id 12345', function(done) {
    request(app)
    .put('/v1/apps/12345/policy')
    .send(fakePolicy)
    .end(function(error,result) {
      expect(result.statusCode).to.equal(201);
      expect(result.body.success).to.equal(true);
      expect(result.body.error).to.be.null;
      expect(result.body.result.policy_json).eql(fakePolicy);
      done();
    });
  });

  it('should fail to create another policy for app id 12345', function(done) {
    request(app)
    .put('/v1/apps/12345/policy')
    .send(fakePolicy)
    .end(function(error,result) {
      expect(result.statusCode).to.equal(400);
      expect(result.body.success).to.equal(false);
      expect(result.body.result).to.be.null;
      expect(result.body.error).to.have.deep.property('name').equal('SequelizeUniqueConstraintError');
      expect(result.body.error).to.have.deep.property('message').equal('Validation error');
      done();
    });
  });
});


