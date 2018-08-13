'use strict';
var expect = require('chai').expect;

var fs = require('fs');
var path = require('path');
var settings = require(path.join(__dirname, '../../lib/config/setting.js'))((JSON.parse(
  fs.readFileSync(path.join(__dirname, '../../config/settings.json'), 'utf8'))));
var models = require('../../lib/models')(settings.db);
var credentials = models.credentials;
var binding = models.binding;

var bindingId = "test_binding";
var userName = "test_username";
var password = "test_password";
var serviceInstanceId = "test_serviceinstance";
var appId = "test_app";


var credential_condition = {
  'bindingId': bindingId, 
  'username': userName,
  'password': password
};

var binding_condition = {
  'serviceInstanceId': serviceInstanceId,
  'appId': appId,
  'bindingId': bindingId
};

var where_condition = { where: { 'username': userName} };


describe('Credential Model Definition Test Suite', function() {

  beforeEach(function(done) {
    return credentials.truncate({ cascade: true }).then(function(result) {
      binding.truncate({ cascade: true }).then(function(result) {
        binding.create(binding_condition).then(function(result) {
          done();
        });
      });
    });


  });

  context(' create and find', function() {
    context('when there is no record ', function() {
      it('should create a new credentials record successfully with the 1st create', function() {
        return credentials.create(credential_condition)
          .then(function(result) {
            expect(result.username).to.equal(credential_condition.username);
          });
      });
    });

    context('when a credential already exists ', function() {
      beforeEach(function() {
        return credentials.create(credential_condition).then(function(result) {});
      });

      it('should find the existing credentials successfully', function() {
        return credentials.findAll(where_condition)
          .then(function(result) {
            expect(result[0].username).to.equal(credential_condition.username);
          });
      });

      it('should throw UniqueConstraintError for conflict record with the 2nd create', function() {
        return credentials.create(credential_condition)
          .then(function(result) {
            expect(result).to.not.exist;
          }).catch(function(error) {
            expect(error).to.not.be.null;
            expect(error instanceof credentials.sequelize.UniqueConstraintError);
          })
      });
    });

  });


  context(' findById ', function() {
    context('when there is no record ', function() {
      it('should find NULL with nonexist credentials', function() {
        return credentials.findById(userName)
          .then(function(instance) {
            expect(instance).to.be.null;
          })
      });
    });

    context('when credentials already exists ', function() {
      beforeEach(function() {
        return credentials.create(credential_condition).then(function(result) {});
      });

      it('should find the credentials', function() {
        return credentials.findById(credential_condition.username)
          .then(function(instance) {
            expect(instance.username).to.equal(credential_condition.username);
          })
      });

    });
  });

  context('destroy credentials ', function() {
    context('when there is no record ', function() {
      it('should return 0', function() {
        return credentials.destroy({
          where: {
            username: credential_condition.username
          }
        }).then(function(count) {
          expect(count).to.equal(0);
        })
      });
    });

    context('when an credentials exists ', function() {
      beforeEach(function(done) {
        return credentials.create(credential_condition).then(function(result) {
          done();
        }).catch(function(error) {
          done(error);
        });
      });

      it('should delete credentials successfully', function() {
        return credentials.destroy({
          where: {
            username: credential_condition.username
          }
        }).then(function(count) {
          expect(count).to.equal(1);
          return credentials.findById(userName)
            .then(function(instance) {
              expect(instance).to.be.null;
            })
        })
      });

    });
  });



});
