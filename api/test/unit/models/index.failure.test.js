'use strict';

var expect = require("chai").expect;
var logger = require('../../../lib/log/logger');
var fs = require('fs');
var path = require('path');
var settings = require(path.join(__dirname, '../../../lib/config/setting.js'))((JSON.parse(
  fs.readFileSync(path.join(__dirname, '../../../config/settings.json'), 'utf8'))));
describe('Test Model Load Failures', function () {
  beforeEach(function(){
  	settings.db.uri = "postgres://postgres@127.0.0.1:5432/wrong-db-name";
  });
  it('Should fail for a invalid DB_URI', function (done) {
    var models = require('../../../lib/models')(settings.db,function (error) {
    	logger.error ('Error while establishing connection', error);
    	expect(error).to.not.be.null;
    	done();
    });
  });
});
