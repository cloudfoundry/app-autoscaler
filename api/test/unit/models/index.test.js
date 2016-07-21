'use strict';

var expect = require("chai").expect;
var logger = require('../../../lib/log/logger');
var _ = require('underscore');

describe('Model Initialization', function () {
  var models;

  before(function(){
    models = require('../../../lib/models')();
  })

  it('returns the Policy model', function () {
    expect(models.policy_json).to.be.ok;
    expect(_.has(models, 'policy_json')).to.be.equal(true);
    expect(_.has(models, 'sequelize')).to.be.equal(true);
    expect(_.keys(models).length).to.be.equal(2);
  });

  it('returns the sequelize instance with default connection pool', function () {
    expect(models.policy_json.sequelize).to.be.ok;
    expect(_.keys(models.policy_json.sequelize.options.pool).length).to.be.equal(3);
    expect(models.policy_json.sequelize.options.pool.max).to.be.equal(10);
    expect(models.policy_json.sequelize.options.pool.min).to.be.equal(0);
    expect(models.policy_json.sequelize.options.pool.idle).to.be.equal(1000);
  });
});
