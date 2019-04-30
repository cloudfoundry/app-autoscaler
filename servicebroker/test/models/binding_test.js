'use strict';
var expect = require('chai').expect;
var Sequelize = require('sequelize');
var fs = require('fs');
var path = require('path');
var settings = require(path.join(__dirname, '../../lib/config/setting.js'))((JSON.parse(
  fs.readFileSync(path.join(__dirname, '../../config/settings.json'), 'utf8'))));
var models = require('../../lib/models')(settings.db);
var serviceInstance = models.service_instance;
var binding = models.binding;

var serviceInstanceId = "test_serviceinstance";
var orgId = "test_org";
var spaceId = "test_space";
var appId = "test_app";
var bindingId = "test_binding";

var appId2 = "test_app2";
var bindingId2 = "test_binding2";

var service_condition = {
  'serviceInstanceId': serviceInstanceId,
  'orgId': orgId,
  'spaceId': spaceId,
  where: { 'serviceInstanceId': serviceInstanceId, 'orgId': orgId, 'spaceId': spaceId },

};
var binding_condition = {
  'serviceInstanceId': serviceInstanceId,
  'appId': appId,
  'bindingId': bindingId
};
var where_condition = { where: { 'serviceInstanceId': serviceInstanceId, 'appId': appId, 'bindingId': bindingId } };
var binding_condition2 = {
  'serviceInstanceId': serviceInstanceId,
  'appId': appId2,
  'bindingId': bindingId
};

describe('Binding Model Definition Test Suite', function() {

  beforeEach(function(done) {
    binding.truncate({ cascade: true }).then(function(result) {
      serviceInstance.truncate({ cascade: true }).then(function(result) {
        serviceInstance.create(service_condition).then(function(result) {
          done();
        });
      });
    });


  });

  context(' create and find', function() {
    context('when there is no record ', function() {
      it('should create a new binding record successfully with the 1st create', function() {
        return binding.create(binding_condition)
          .then(function(result) {
            expect(result.bindingId).to.equal(binding_condition.bindingId);
          });
      });
    });

    context('when an binding already exists ', function() {
      beforeEach(function() {
        return binding.create(binding_condition).then(function(result) {});
      });

      it('should find the existing binding successfully', function() {
        return binding.findAll(where_condition)
          .then(function(result) {
            expect(result[0].bindingId).to.equal(binding_condition.bindingId);
          });
      });

      it('should throw UniqueConstraintError for conflict record with the 2nd create', function() {
        return binding.create(binding_condition2)
          .then(function(result) {
            expect(result).to.not.exist;
          }).catch(function(error) {
            expect(error).to.not.be.null;
            expect(error instanceof Sequelize.UniqueConstraintError);
          })
      });
    });

  });


  context(' findById ', function() {
    context('when there is no record ', function() {
      it('should find NULL with nonexist service instance', function() {
        return binding.findByPk(bindingId)
          .then(function(instance) {
            expect(instance).to.be.null;
          })
      });
    });

    context('when an instance already exists ', function() {
      beforeEach(function() {
        return binding.create(binding_condition).then(function(result) {});
      });

      it('should find an instance', function() {
        return binding.findByPk(binding_condition.bindingId)
          .then(function(instance) {
            expect(instance.bindingId).to.equal(binding_condition.bindingId);
          })
      });

    });
  });

  context('destroy ', function() {
    context('when there is no record ', function() {
      it('should return 0', function() {
        return binding.destroy({
          where: {
            bindingId: binding_condition.bindingId
          }
        }).then(function(count) {
          expect(count).to.equal(0);
        })
      });
    });

    context('when an instance already exists ', function() {
      beforeEach(function(done) {
        binding.create(binding_condition).then(function(result) {
          done();
        }).catch(function(error) {
          done(error);
        });
      });

      it('should delete an instance', function() {
        return binding.destroy({
          where: {
            bindingId: binding_condition.bindingId
          }
        }).then(function(count) {
          expect(count).to.equal(1);
          return binding.findByPk(bindingId)
            .then(function(instance) {
              expect(instance).to.be.null;
            })
        })
      });

    });
  });



});
