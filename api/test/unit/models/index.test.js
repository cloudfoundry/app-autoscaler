'use strict';

var expect = require("chai").expect;
var logger = require('../../../lib/log/logger');
var fs = require('fs');
var path = require('path');
var settings = require(path.join(__dirname, '../../../lib/config/setting.js'))((JSON.parse(
  fs.readFileSync(path.join(__dirname, '../../../config/settings.json'), 'utf8'))));
var _ = require('underscore');

describe('Model Initialization', function () {
  var models;

  before(function(){
    models = require('../../../lib/models')(settings.db);
  })

  it('returns the Policy model', function () {
    expect(models.policy_json).to.be.ok;
    expect(_.has(models, 'policy_json')).to.be.equal(true);
    expect(_.has(models, 'sequelize')).to.be.equal(true);
    expect(_.keys(models).length).to.be.equal(3);
  });

  it('returns the sequelize instance with default connection pool', function () {
    expect(models.policy_json.sequelize).to.be.ok;
    expect(_.keys(models.policy_json.sequelize.options.pool).length).to.be.equal(3);
    expect(models.policy_json.sequelize.options.pool.max).to.be.equal(10);
    expect(models.policy_json.sequelize.options.pool.min).to.be.equal(0);
    expect(models.policy_json.sequelize.options.pool.idle).to.be.equal(1000);
  });
});
