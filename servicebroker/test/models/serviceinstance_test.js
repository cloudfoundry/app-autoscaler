'use strict';
var expect = require('chai').expect;
var Sequelize = require('sequelize');

var fs = require('fs');
var path = require('path');
var settings = require(path.join(__dirname, '../../lib/config/setting.js'))((JSON.parse(
  fs.readFileSync(path.join(__dirname, '../../config/settings.json'), 'utf8'))));
var models = require('../../lib/models')(settings.db);
var serviceInstance = models.service_instance;

var serviceInstanceId = "test_serviceinstance";
var orgId = "test_org";
var spaceId = "test_space";
var orgId2 = "test_org2";
var spaceId2 = "test_space2";
var condition = {
  'serviceInstanceId': serviceInstanceId,
  'orgId': orgId,
  'spaceId': spaceId,
  where: { 'serviceInstanceId': serviceInstanceId, 'orgId': orgId, 'spaceId': spaceId }
};
var condition2 = {
  'serviceInstanceId': serviceInstanceId,
  'orgId': orgId2,
  'spaceId': spaceId2,
  where: { 'serviceInstanceId': serviceInstanceId, 'orgId': orgId2, 'spaceId': spaceId2 }
};

describe('Service_instance Model Definition Test Suite', function() {

  beforeEach(function() {
    return serviceInstance.truncate({ cascade: true });
  });


  context('findOrCreate', function() {
    context('when there is no record ', function() {
      it('should create a new service instance successfully with the 1st findOrCreate', function() {
        return serviceInstance.findOrCreate(condition)
          .then(function(result) {
            expect(result[0].serviceInstanceId).to.equal(condition.serviceInstanceId);
            expect(result[1]).to.equal(true);
          })
      });
    });

    context('when an instance already exists ', function() {
      beforeEach(function() {
        return serviceInstance.findOrCreate(condition);
      });

      it('should find the existing service instance successfully with the 2nd findOrCreate', function() {
        return serviceInstance.findOrCreate(condition)
          .then(function(result) {
            expect(result[0].serviceInstanceId).to.equal(condition.serviceInstanceId);
            expect(result[1]).to.equal(false);
          });
      });

      it('should throw UniqueConstraintError for conflict record with the 2nd findOrCreate', function() {
        return serviceInstance.findOrCreate(condition2)
          .then(function(result) {
            expect(result).to.not.exist;
          }).catch(function(error) {
            expect(error).to.not.be.null;
            expect(error instanceof Sequelize.UniqueConstraintError)
          })
      });
    });

  });


  context('findByPk', function() {
    context('when there is no record ', function() {
      it('should find NULL with nonexist service instance', function() {
        return serviceInstance.findByPk(serviceInstanceId)
          .then(function(instance) {
            expect(instance).to.be.null;
          })
      });
    });

    context('when an instance already exists ', function() {
      beforeEach(function() {
        return serviceInstance.findOrCreate(condition);
      });

      it('should find an instance', function() {
        return serviceInstance.findByPk(condition.serviceInstanceId)
          .then(function(instance) {
            expect(instance.serviceInstanceId).to.equal(condition.serviceInstanceId);
          })
      });

    });
  });

  context('destroy', function() {
    context('when there is no record ', function() {
      it('should return 0 with nonexist service instance', function() {
        return serviceInstance.destroy({
          where: {
            serviceInstanceId: condition.serviceInstanceId
          }
        }).then(function(count) {
          expect(count).to.equal(0);
        })
      });
    });

    context('when an instance already exists ', function() {
      beforeEach(function() {
        return serviceInstance.findOrCreate(condition);
      });

      it('should delete an instance', function() {
        return serviceInstance.destroy({
          where: {
            serviceInstanceId: condition.serviceInstanceId
          }
        }).then(function(count) {
          expect(count).to.equal(1);
          return serviceInstance.findByPk(serviceInstanceId)
            .then(function(instance) {
              expect(instance).to.be.null;
            })
        })
      });

    });
  });
});
