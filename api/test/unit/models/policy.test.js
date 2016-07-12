'use strict';
var expect = require('chai').expect;
var fs = require('fs');
var logger = require('../../../lib/log/logger');

describe('Create a Policy', function() {
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
  it('Should create a policy for app id 99999 successfully', function() {
    return this.policy.create({ 'policy_json':fakePolicy, 'app_id': '99999' })
    .then(function(policy) {
      expect(policy.policy_json).to.deep.equal(fakePolicy);
    });
  });
  it('Should fail to create a policy with duplicate app_id 99999', function() {
    return this.policy.create({ 'policy_json':fakePolicy, 'app_id': '99999' })
    .catch(function(error) {
      expect(error).to.not.be.null;
      expect(error).to.have.deep.property('name').equal('SequelizeUniqueConstraintError');
      expect(error).to.have.deep.property('message').equal('Validation error');
    });
  });
});
