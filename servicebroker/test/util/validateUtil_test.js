'use strict';

var path = require('path');
var expect = require('chai').expect;
var validateUtil = require(path.join(__dirname, '../../lib/util/validateUtil.js'));
var testNullObject, testUndefinedObject;
var testObject, keys;
describe('validateUtil Test Suite', function() {
  context('isNull()', function() {
    context('when testObject is null', function() {
      beforeEach(function(done) {
        testNullObject = null;
        done();
      });
      it('should return true', function() {
        expect(validateUtil.isNull(testNullObject)).to.equal(true);
      });
    });

    context('when testNullObject is not null', function() {
      beforeEach(function(done) {
        testNullObject = 'has value';
        done();
      });
      it('should return false', function() {
        expect(validateUtil.isNull(testNullObject)).to.equal(false);
      });
    });
  });

  context('isUndefined()', function() {
    context('when testUndefinedObject is not defined', function() {
      it('should return true', function() {
        expect(validateUtil.isUndefined(testUndefinedObject)).to.equal(true);
      });
    });
    context('when testUndefinedObject is defined', function() {
      beforeEach(function(done) {
        testUndefinedObject = 'has value';
        done();
      });
      it('should return false', function() {
        expect(validateUtil.isUndefined(testUndefinedObject)).to.equal(false);
      });
    });
  });
  
  context('validate(testObject, keys)', function() {
    context('when testObject is null', function() {
      it('should retrun false', function() {
        expect(validateUtil.validate(null).valid).to.equal(false);
      });
    });
    context('when testObject is not defined', function() {
      it('should retrun false', function() {
        expect(validateUtil.validate(testObject).valid).to.equal(false);;
      });
    });
    context('when keys is not a array', function() {
      beforeEach(function(done) {
        testObject = { "key": "value" };
        keys = {};
        done();
      });
      it('should retrun false', function() {
        expect(validateUtil.validate(testObject, keys).valid).to.equal(false);;
      });
    });

    context('when testObject[key] is null', function() {
      beforeEach(function(done) {
        testObject = { "key": null };
        keys = ["key"];
        done();
      });
      it('should retrun false', function() {
        expect(validateUtil.validate(testObject,keys).valid).to.equal(false);;
      });
    });
    context('when testObject[key] is not defined', function() {
      beforeEach(function(done) {
        testObject = {};
        keys = ["key"];
        done();
      });
      it('should retrun false', function() {
        expect(validateUtil.validate(testObject,keys).valid).to.equal(false);;
      });
    });
    context('when testObject and  keys are both valid', function() {
      beforeEach(function(done) {
        testObject = { "key": "value" };
        keys = ["key"];
        done();
      });
      it('should retrun true', function() {
        expect(validateUtil.validate(testObject, keys).valid).to.equal(true);
      });
    });

  });
});