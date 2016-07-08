'use strict';
var expect = require('chai').expect;
var serviceInstance = require('../../lib/models')().service_instance;

var fake_serviceInstanceId = "test_serviceinstance";
var fake_orgId = "test_org";
var fake_spaceId = "test_space";
var fake_orgId2 = "test_org2";
var fake_spaceId2 = "test_space2";
var fake_condition = {
  'serviceInstanceId': fake_serviceInstanceId,
  'orgId': fake_orgId,
  'spaceId': fake_spaceId,
  where: { 'serviceInstanceId': fake_serviceInstanceId, 'orgId': fake_orgId, 'spaceId': fake_spaceId }
};
var fake_condition2 = {
  'serviceInstanceId': fake_serviceInstanceId,
  'orgId': fake_orgId2,
  'spaceId': fake_spaceId2,
  where: { 'serviceInstanceId': fake_serviceInstanceId, 'orgId': fake_orgId2, 'spaceId': fake_spaceId2 }
};

describe('Service_instance Model Definition Test Suite', function() {

  before(function(done) {
    serviceInstance.sequelize.sync().then(function(result){done();}).catch(function(error){
      console.log("Failed to sync model serviceInstance, error: " + error);
    });
  });

  beforeEach(function() {
    serviceInstance.truncate({ cascade: true });
  });


  context('findOrCreate', function() {
    context('when there is no record ', function() {
      it('should create a new service instance successfully with the 1st findOrCreate', function() {
        return serviceInstance.findOrCreate(fake_condition)
          .then(function(result) {
            expect(result[0].serviceInstanceId).to.equal(fake_condition.serviceInstanceId);
            expect(result[1]).to.equal(true);
          })
      });
    });

    context('when an instance already exists ', function() {
      beforeEach(function() {
        return serviceInstance.findOrCreate(fake_condition);
      });

      it('should find the existing service instance successfully with the 2nd findOrCreate', function() {
        return serviceInstance.findOrCreate(fake_condition)
          .then(function(result) {
            expect(result[0].serviceInstanceId).to.equal(fake_condition.serviceInstanceId);
            expect(result[1]).to.equal(false);
          });
      });

      it('should throw UniqueConstraintError for conflict record with the 2nd findOrCreate', function() {
        return serviceInstance.findOrCreate(fake_condition2)
          .then(function(result) {
            expect(result).to.not.exist;
          }).catch(function(error) {
            expect(error).to.not.be.null;
            expect(error instanceof serviceInstance.sequelize.UniqueConstraintError);
          })
      });
    });

  });


  context('findById', function() {
    context('when there is no record ', function() {
      it('should find NULL with nonexist service instance', function() {
        return serviceInstance.findById(fake_serviceInstanceId)
          .then(function(instance) {
            expect(instance).to.be.null;
          })
      });
    });

    context('when an instance already exists ', function() {
      beforeEach(function() {
        return serviceInstance.findOrCreate(fake_condition);
      });

      it('should find an instance', function() {
        return serviceInstance.findById(fake_condition.serviceInstanceId)
          .then(function(instance) {
            expect(instance.serviceInstanceId).to.equal(fake_condition.serviceInstanceId);
          })
      });

    });
  });

  context('destroy', function() {
    context('when there is no record ', function() {
      it('should return 0 with nonexist service instance', function() {
        return serviceInstance.destroy({
          where: {
            serviceInstanceId: fake_condition.serviceInstanceId
          }
        }).then(function(count) {
          expect(count).to.equal(0);
        })
      });
    });

    context('when an instance already exists ', function() {
      beforeEach(function() {
        return serviceInstance.findOrCreate(fake_condition);
      });

      it('should delete an instance', function() {
        return serviceInstance.destroy({
          where: {
            serviceInstanceId: fake_condition.serviceInstanceId
          }
        }).then(function(count) {
          expect(count).to.equal(1);
          return serviceInstance.findById(fake_serviceInstanceId)
            .then(function(instance) {
              expect(instance).to.be.null;
            })
        })
      });

    });
  });
});
