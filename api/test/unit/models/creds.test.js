'use strict';
var expect = require('chai').expect;
var fs = require('fs');
var path = require('path');
var Sequelize = require('sequelize');
var settings = require(path.join(__dirname, '../../../lib/config/setting.js'))((JSON.parse(
    fs.readFileSync(path.join(__dirname, '../../../config/settings.json'), 'utf8'))));
var credentials = require('../../../lib/models')(settings.db).credentials;


var userName = 'test_username';
var password = 'test_password';
var appId = 'test_app';


var credentialCondition = {
  'id': appId,
  'username': userName,
  'password': password
};

var whereCondition = {
  where: {
    'id': appId
  }
};


describe('Credential Model Test Suite', function() {
    
  context('Test create credentials', function() {
    beforeEach(function() {
      return credentials.truncate();
    });

    context('when there is no credentials ', function() {
      it('should create a new credentials record successfully', function() {
        return credentials.create(credentialCondition)
            .then(function(result) {
              expect(result.id).to.equal(credentialCondition.id);
            });
      });
    });
    context('when a credential already exists ', function() {
      beforeEach(function() {
        return credentials.create(credentialCondition).then(function(result) {});
      });

      it('should find the existing credentials successfully', function() {
        return credentials.findAll(whereCondition)
        .then(function(result) {
          expect(result[0].id).to.equal(credentialCondition.id);
        });
      });

      it('should throw UniqueConstraintError for conflict record with the 2nd create', function() {
        return credentials.create(credentialCondition)
        .then(function(result) {
          expect(result).to.not.exist;
        }).catch(function(error) {
            expect(error).to.not.be.null;
            expect(error instanceof Sequelize.UniqueConstraintError);
        });
      });

      it('should update the existing credentials on 2nd create', function() {
        return credentials.upsert(credentialCondition)
          .then(function(result) {
            expect(result).to.be.false;
          }).catch(function(error) {
            expect(error).to.not.be.null;
          });
      });
    });
  });
  context('Test destroy credentials ', function() {
    beforeEach(function() {
      return credentials.truncate();
    });
    context('when there is no record ', function() {
      it('should return 0', function() {
        return credentials.destroy({
          where: {
            id: credentialCondition.id
          }
        }).then(function(count) {
          expect(count).to.equal(0);
        })
      });
    });
    context('when a credentials exists ', function() {
      beforeEach(function(done) {
        credentials.create(credentialCondition).then(function(result) {
          done();
        }).catch(function(error) {
          done(error);
        });
      });
      it('should delete credentials successfully', function() {
        return credentials.destroy({
          where: {
            username: credentialCondition.username
          }
        }).then(function(count) {
          expect(count).to.equal(1);
          return credentials.findByPk(userName)
          .then(function(instance) {
          expect(instance).to.be.null;
          });
        });
      });
    });
  });
});
