'use strict';
var expect = require('chai').expect;
var serviceInstance = require('../../lib/models')().service_instance;
var binding = require('../../lib/models')().binding;

var fake_serviceInstanceId = "test_serviceinstance";
var fake_orgId = "test_org";
var fake_spaceId = "test_space";
var fake_appId = "test_app";
var fake_bindingId = "test_binding";

var fake_appId2 = "test_app2";
var fake_bindingId2 = "test_binding2";

var fake_service_condition = {
  'serviceInstanceId': fake_serviceInstanceId,
  'orgId': fake_orgId,
  'spaceId': fake_spaceId,
  where: { 'serviceInstanceId': fake_serviceInstanceId, 'orgId': fake_orgId, 'spaceId': fake_spaceId },

};
var fake_binding_condition = {
  'serviceInstanceId': fake_serviceInstanceId,
  'appId': fake_appId,
  'bindingId': fake_bindingId,
  'timestamp': new Date().getTime() 
};
var where_condition = {where: { 'serviceInstanceId': fake_serviceInstanceId, 'appId': fake_appId, 'bindingId': fake_bindingId }};
var fake_binding_condition2 = {
  'serviceInstanceId': fake_serviceInstanceId,
  'appId': fake_appId2,
  'bindingId': fake_bindingId,
  'timestamp': new Date().getTime() 
};

describe('Binding Model Definition Test Suite', function() {

  before(function(done) {
    serviceInstance.sequelize.sync().then(function(result) {
      binding.sequelize.sync().then(function(result) {
        done();
      }).catch(function(error1) {
        console.log("Failed to sync model binding, error: " + error1);
        done(error1);
      });
    }).catch(function(error) {
      console.log("Failed to sync model serviceInstance, error: " + error);
      done(error);
    });
  });

  beforeEach(function(done) {
    return binding.truncate({ cascade: true }).then(function(result) {
      serviceInstance.truncate({ cascade: true }).then(function(result) {
        serviceInstance.findOrCreate(fake_service_condition).then(function(result) {
          done();
        }).catch(function(error1) {
          done(error1);
        });
      }).catch(function(error2) {
        done(error2);
      });
    }).catch(function(error3) {
      done(error3);
    });


  });

  context(' create and find', function() {
    context('when there is no record ', function() {
      it('should create a new binding record successfully with the 1st create', function() {
        return binding.create(fake_binding_condition)
          .then(function(result) {
            expect(result[0].bindingId).to.equal(fake_binding_condition.bindingId);
            expect(result[1]).to.equal(true);
          }).catch(function(err) {

          });
      });
    });

    context('when an binding already exists ', function() {
      beforeEach(function(done) {
        return binding.create(fake_binding_condition).then(function(result) {
          done();
        }).catch(function(error) {
          done(error);
        });
      });

      it('should find the existing binding successfully', function() {
        return binding.findAll(where_condition)
          .then(function(result) {
            expect(result[0].bindingId).to.equal(fake_binding_condition.bindingId);
          });
      });

      it('should throw UniqueConstraintError for conflict record with the 2nd create', function() {
        return binding.create(fake_binding_condition2)
          .then(function(result) {
            expect(result).to.not.exist;
          }).catch(function(error) {
            expect(error).to.not.be.null;
            expect(error instanceof binding.sequelize.UniqueConstraintError);
          })
      });
    });

  });


  context(' findById ', function() {
    context('when there is no record ', function() {
      it('should find NULL with nonexist service instance', function() {
        return binding.findById(fake_bindingId)
          .then(function(instance) {
            expect(instance).to.be.null;
          })
      });
    });

    context('when an instance already exists ', function() {
      beforeEach(function(done) {
        return binding.create(fake_binding_condition).then(function(result) {
          done();
        }).catch(function(error) {
          done(error);
        });
      });

      it('should find an instance', function() {
        return binding.findById(fake_binding_condition.bindingId)
          .then(function(instance) {
            expect(instance.bindingId).to.equal(fake_binding_condition.bindingId);
          })
      });

    });
  });

  context(' destory ', function() {
    context('when there is no record ', function() {
      it('should return 0 with nonexist binding', function() {
        return binding.destroy({
          where: {
            bindingId: fake_binding_condition.bindingId
          }
        }).then(function(count) {
          expect(count).to.equal(0);
        })
      });
    });

    context('when an instance already exists ', function() {
      beforeEach(function(done) {
        return binding.create(fake_binding_condition).then(function(result) {
          done();
        }).catch(function(error) {
          done(error);
        });
      });

      it('should delete an instance', function() {
        return binding.destroy({
          where: {
            bindingId: fake_binding_condition.bindingId
          }
        }).then(function(count) {
          expect(count).to.equal(1);
          return binding.findById(fake_bindingId)
            .then(function(instance) {
              expect(instance).to.be.null;
            })
        })
      });

    });
  });



});