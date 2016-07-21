'use strict';
var expect = require('chai').expect;
var fs = require('fs');
var logger = require('../../../lib/log/logger');
var policy = require('../../../lib/models')().policy_json;

describe('Create a Policy', function() {
  var fakePolicy;
  before(function(done) {
    policy.sequelize.sync({force:true}).then(function(success) {
        fakePolicy = JSON.parse(fs.readFileSync(__dirname+'/../fakePolicy.json', 'utf8'));
        done();
    }, function(error) {
      logger.error('Failed to setup database for test',error);
      done(error);
    });
  });
  it('Should create a policy for app id 99999 successfully', function() {
    return policy.create({ 'policy_json':fakePolicy, 'app_id': '99999' })
    .then(function(policy) {
      expect(policy.policy_json).to.deep.equal(fakePolicy);
    });
  });
  it('Should fail to create a policy with duplicate app_id 99999', function() {
    return policy.create({ 'policy_json':fakePolicy, 'app_id': '99999' })
    .catch(function(error) {
      expect(error).to.not.be.null;
      expect(error).to.have.deep.property('name').equal('SequelizeUniqueConstraintError');
      expect(error).to.have.deep.property('message').equal('Validation error');
    });
  });
  after(function(done){
    policy.drop().then(function(result) {
        done();
      },function(error){
      logger.error('Failed to clean up database after test',error);
      done(error);
    });
  });
});
