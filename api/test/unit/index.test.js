'use strict';

//var expect = require('expect.js');
var expect = require("chai").expect;
var logger = require('../../lib/log/logger');

describe('Model Initialization', function () {
  it('returns the Policy model', function () {
    var models = require('../../lib/models');
    logger.log('Policy model loaded successfully');
    expect(models.policy_json).to.be.ok;
  });
});
