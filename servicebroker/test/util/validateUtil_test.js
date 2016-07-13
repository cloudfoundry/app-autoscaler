'use strict';

var path = require('path');
var expect = require('chai').expect;
var validateUtil = require(path.join(__dirname, '../../lib/util/validateUtil.js'));
var testNullObject, testUndefinedObject;
var testObject, keys;
describe('validateUtil Test Suite', function() {
  context('validate(testObject, keys)', function() {
    context('when testObject is null', function() {
      it('should retrun false', function() {
        expect(validateUtil.validate(null)).to.equal(false);
      });
    });
    context('when testObject is not defined', function() {
      it('should retrun false', function() {
        expect(validateUtil.validate(testObject)).to.equal(false);;
      });
    });
    context('when keys is not a array', function() {
      beforeEach(function() {
        testObject = { "key": "value" };
        keys = {};

      });
      it('should retrun false', function() {
        expect(validateUtil.validate(testObject, keys)).to.equal(false);;
      });
    });

    context('when testObject[key] is null', function() {
      beforeEach(function() {
        testObject = { "key": null };
        keys = ["key"];

      });
      it('should retrun false', function() {
        expect(validateUtil.validate(testObject, keys)).to.equal(false);;
      });
    });
    context('when testObject[key] is not defined', function() {
      beforeEach(function() {
        testObject = {};
        keys = ["key"];

      });
      it('should retrun false', function() {
        expect(validateUtil.validate(testObject, keys)).to.equal(false);;
      });
    });
    context('when testObject and  keys are both valid', function() {
      beforeEach(function() {
        testObject = { "key": "value" };
        keys = ["key"];

      });
      it('should retrun true', function() {
        expect(validateUtil.validate(testObject, keys)).to.equal(true);
      });
    });

  });
});