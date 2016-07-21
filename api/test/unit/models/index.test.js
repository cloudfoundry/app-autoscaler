'use strict';

var expect = require("chai").expect;
var logger = require('../../../lib/log/logger');
var _ = require('underscore');

describe('Model Initialization', function () {
  var models;
  before(function(done){
    models = require('../../../lib/models')();
    logger.log('Policy model loaded successfully');
    done();
  })
  it('returns the Policy model', function () {
    expect(models.policy_json).to.be.ok;
    expect(_.has(models, 'policy_json')).to.be.equal(true);
    expect(_.has(models, 'sequelize')).to.be.equal(true);
    expect(_.keys(models).length).to.be.equal(2);
  });
  it('returns the sequelize instance with default connection pool', function () {
    expect(models.policy_json.sequelize).to.be.ok;
    expect(models.policy_json.sequelize.options.maxConcurrentQueries).to.be.equal(50);
    expect(_.keys(models.policy_json.sequelize.options.pool).length).to.be.equal(3);
    expect(models.policy_json.sequelize.options.pool.maxConnections).to.be.equal(10);
    expect(models.policy_json.sequelize.options.pool.minConnections).to.be.equal(0);
    expect(models.policy_json.sequelize.options.pool.maxIdleTime).to.be.equal(1000);
  });
});
