'use strict';
var expect = require('chai').expect;
var models = require('../../lib/models')();
var serviceInstance = require('../../lib/models')().service_instance;

var fake_serviceId = "test_service";
var fake_orgId = "test_org";
var fake_spaceId = "test_space";
var fake_orgId2 = "test_org2";
var fake_spaceId2 = "test_space2";
var fake_condition = {
  'serviceId': fake_serviceId,
  'orgId': fake_orgId,
  'spaceId': fake_spaceId,
  where: { 'serviceId': fake_serviceId, 'orgId': fake_orgId, 'spaceId': fake_spaceId }
};
var fake_condition2 = {
  'serviceId': fake_serviceId,
  'orgId': fake_orgId2,
  'spaceId': fake_spaceId2,
  where: { 'serviceId': fake_serviceId, 'orgId': fake_orgId2, 'spaceId': fake_spaceId2 }
};

describe('Service_instance Model Definition Test Suite', function() {

  before(function() {
    serviceInstance.sequelize.sync();
    serviceInstance.truncate({ cascade: true });
  });

  beforeEach(function() {
    serviceInstance.truncate({ cascade: true });
  });

  it('Should create a new service instance successfully with the 1st findOrCreate', function(done) {
    serviceInstance.findOrCreate(fake_condition)
      .then(function(result) {
        expect(result[0].serviceId).to.equal(fake_condition.serviceId);
        expect(result[1]).to.equal(true);
        done();
      }) 
      .catch (function(error) {
          console.log ("error in findOrCreate: " + error.message);
          done();
     });
  });


  it('Should find the existing service instance successfully with the 2nd findOrCreate', function(done) {
    serviceInstance.findOrCreate(fake_condition)
      .then(function(result) {
        serviceInstance.findOrCreate(fake_condition)
          .then(function(result) {
            expect(result[0].serviceId).to.equal(fake_condition.serviceId);
            expect(result[1]).to.equal(false);
            done();
          });
      });
  });

  it('Should throw UniqueConstraintError with conflict record', function(done) {
    serviceInstance.findOrCreate(fake_condition)
      .then(function(result) {
        serviceInstance.findOrCreate(fake_condition2)
          .then(function(result) {})
          .catch(function(error) {
            expect(error).to.not.be.null;
            expect(error instanceof serviceInstance.sequelize.UniqueConstraintError);
            done();
          });
      });
  });


});