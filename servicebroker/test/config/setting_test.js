'use strict';

var path = require('path');
var expect = require('chai').expect;
var setting = {
  "port": "testPort",
  "username": "username",
  "password": "password",
  "dbUri": "postgress://postgres@server:port/dbname",
  "apiServerUri": "http://apiserveruri"
}
describe('config setting Test Suite', function() {
  it('Should get port correctly', function() {
    var settingObj = require(path.join(__dirname, '../../lib/config/setting.js'))(setting);
    expect(settingObj.port).to.equal(setting.port);
  });
  it('Should get username and password correctly', function() {
    var settingObj = require(path.join(__dirname, '../../lib/config/setting.js'))(setting);
    expect(settingObj.username).to.equal(setting.username);
    expect(settingObj.password).to.equal(setting.password);
  });
  context('Parse dbUri', function() {
    it('Should parse dbUri correctly', function() {
      var dbSetting = require(path.join(__dirname, '../../lib/config/setting.js'))(setting).db;
      expect(dbSetting.uri).to.equal(setting.dbUri);
      expect(dbSetting.server).to.equal("postgress://postgres@server:port");
      expect(dbSetting.name).to.equal("dbname");
    });

    it('Should filter the last slash when parse dbUri', function() {
      var settingTmp = {};
      settingTmp.port = setting.port;
      settingTmp.username = setting.username;
      settingTmp.password = setting.password;
      settingTmp.dbUri = setting.dbUri + '/';
      settingTmp.apiServerUri = setting.apiServerUri;

      var dbSetting = require(path.join(__dirname, '../../lib/config/setting.js'))(settingTmp).db;
      expect(dbSetting.uri).to.equal(setting.dbUri);
      expect(dbSetting.server).to.equal("postgress://postgres@server:port");
      expect(dbSetting.name).to.equal("dbname");
    });

    it('Should handle the upper case when parse dbUri', function() {
      var settingTmp = {};
      settingTmp.port = setting.port;
      settingTmp.username = setting.username;
      settingTmp.password = setting.password;
      settingTmp.dbUri = setting.dbUri.toUpperCase();
      settingTmp.apiServerUri = setting.apiServerUri;

      var dbSetting = require(path.join(__dirname, '../../lib/config/setting.js'))(setting).db;
      expect(dbSetting.uri).to.equal(setting.dbUri);
      expect(dbSetting.server).to.equal("postgress://postgres@server:port");
      expect(dbSetting.name).to.equal("dbname");
    });
  });
  context('Parse apiServerUri', function() {
    it('Should parse apiServerUri correctly', function() {
      var apiServerUriParsed = require(path.join(__dirname, '../../lib/config/setting.js'))(setting).apiServerUri;
      expect(apiServerUriParsed).to.equal(setting.apiServerUri);
    });

    it('Should filter the last slash when parse apiServerUri', function() {
      var settingTmp = {};
      settingTmp.port = setting.port;
      settingTmp.username = setting.username;
      settingTmp.password = setting.password;
      settingTmp.dbUri = setting.dbUri;
      settingTmp.apiServerUri = setting.apiServerUri + '/';

      var apiServerUriParsed = require(path.join(__dirname, '../../lib/config/setting.js'))(settingTmp).apiServerUri;
      expect(apiServerUriParsed).to.equal(setting.apiServerUri);
    });

    it('Should handle the upper case when parse apiServerUri', function() {
      var settingTmp = {};
      settingTmp.port = setting.port;
      settingTmp.username = setting.username;
      settingTmp.password = setting.password;
      settingTmp.dbUri = setting.dbUri;
      settingTmp.apiServerUri = setting.apiServerUri.toUpperCase();

      var apiServerUriParsed = require(path.join(__dirname, '../../lib/config/setting.js'))(setting).apiServerUri;
      expect(apiServerUriParsed).to.equal(setting.apiServerUri);
    });
  });



});