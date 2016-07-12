'use strict';

var expect = require("chai").expect;
var logger = require('../../../lib/log/logger');
var _ = require('underscore');

describe('Model Initialization', function () {
  it('returns the Policy model', function () {
    var models = require('../../../lib/models')();
    logger.log('Policy model loaded successfully');
    expect(models.policy_json).to.be.ok;
    expect(_.has(models, 'policy_json')).to.be.equal(true);
    expect(_.has(models, 'sequelize')).to.be.equal(true);
    expect(_.keys(models).length).to.be.equal(2);
  });
});
