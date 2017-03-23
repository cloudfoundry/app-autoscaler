'use strict';
var expect = require('chai').expect;
var fs = require('fs');
var path = require('path');
var moment = require('moment');
var uuidV4 = require('uuid/v4');
var logger = require('../../../lib/log/logger');
var settings = require(path.join(__dirname, '../../../lib/config/setting.js'))((JSON.parse(
  fs.readFileSync(path.join(__dirname, '../../../config/settings.json'), 'utf8'))));
var policy = require('../../../lib/models')(settings.db).policy_json;

describe('Create a Policy', function() {
  var fakePolicy;

  before(function() {
    fakePolicy = JSON.parse(fs.readFileSync(__dirname+'/../fakePolicy.json', 'utf8'));
  });

  context ('Test create policies', function () {
    beforeEach(function() {
      return policy.truncate();
    });

    it('Should create a policy for app id 99999 successfully', function() {
      var guid = uuidV4();
      return policy.create({ 'policy_json':fakePolicy, 'app_id': '99999', 'guid': guid })
      .then(function(policy) {
        expect(policy.policy_json).to.deep.equal(fakePolicy);
        expect(policy).to.have.property('guid', guid);
      });
    });

    it('Should fail to create a policy with duplicate app_id 99999', function(done) {
      policy.create({ 'policy_json':fakePolicy, 'app_id': '99999', 'guid': uuidV4() })
        .then(function(savedPolicy) {
        policy.create({ 'policy_json':fakePolicy, 'app_id': '99999', 'guid': uuidV4() })
        .catch(function(error) {
            expect(error).to.not.be.null;
            expect(error).to.have.deep.property('name').equal('SequelizeUniqueConstraintError');
            expect(error).to.have.deep.property('message').equal('Validation error');
            done();
        });
      });
    });    
  });
  
  context ('Test create and update policies with updated_at checks', function () {
    beforeEach(function() {
      return policy.truncate();
    });

    it('Create a policy with app_id 99999 and then updates the policy for the same', function(done) {
      var initialGuid = uuidV4();
      policy.create({ 'policy_json':fakePolicy, 'app_id': '99999', 'guid': initialGuid })
        .then(function(createdPolicy) {
          expect(createdPolicy).to.have.property('guid', initialGuid);
          setTimeout(function(){
            var updatedGuid = uuidV4();
            policy.update({ 'policy_json':fakePolicy, 'app_id': '99999', 'guid': updatedGuid },{ where: { app_id: '99999' } ,returning:true})
              .then(function(result) {
                var updatedPolicy = result[1][0];
                expect(updatedPolicy.policy_json).to.deep.equal(fakePolicy);
                expect(updatedPolicy).to.have.property('guid', updatedGuid);
                done();
            });
          }, 1000);
          // Adding a delay of 1 second before updating the policy for the same app_id 99999 to check the trigger update 
      });
    });
  });
});
