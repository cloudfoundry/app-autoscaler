'use strict';

var path = require('path');
var expect = require('chai').expect;
var configSetting = require(path.join(__dirname, '../../lib/config/setting.js'));
var setting = {
  "port": "testPort",
  "username": "username",
  "password": "password",
  "dbUri": "postgress://postgres@server:port/dbname",
  "apiServerUri": "http://apiserveruri"
}
var settingTmp = {};
var settingObj, dbSetting;
var apiServerUriParsed;
var settingUndefined;
describe('config setting Test Suite', function() {
  beforeEach(function(done) {
    settingObj = require(path.join(__dirname, '../../lib/config/setting.js'))(setting);
    done();
  });
  it('Should get port correctly', function() {
    expect(settingObj.port).to.equal(setting.port);
  });
  it('Should get username and password correctly', function() {
    expect(settingObj.username).to.equal(setting.username);
    expect(settingObj.password).to.equal(setting.password);
  });
  context('Get setting', function(){
    context('when setting is null',function(){
      beforeEach(function(done){
        settingTmp = null;
        done();
      });
      it('Should return null',function(){
        var settingRead = configSetting(settingTmp);
        expect(settingRead).to.be.null;
      });
    });
    context('when setting is undefined',function(){
      it('Should return null',function(){
        var settingRead = configSetting(settingUndefined);
        expect(settingRead).to.be.null;
      });
    });
    context('when some field(s) of setting is null',function(){
      beforeEach(function(done){
        settingTmp = {};
        settingTmp.port = null;
        settingTmp.username = setting.username;
        settingTmp.password = setting.password;
        settingTmp.dbUri = setting.dbUri;
        settingTmp.apiServerUri = setting.apiServerUri;
        done();
      });
      it('Should return null',function(){
        var settingRead = configSetting(settingTmp);
        expect(settingRead).to.be.null;
      });
    });
    context('when some neccessary field(s) of setting is undefined',function(){
      beforeEach(function(done){
        settingTmp = {};
        //port is undefined
        settingTmp.username = setting.username;
        settingTmp.password = setting.password;
        settingTmp.dbUri = setting.dbUri;
        settingTmp.apiServerUri = setting.apiServerUri;
        done();
      });
      it('Should return null',function(){
        var settingRead = configSetting(settingTmp);
        expect(settingRead).to.be.null;
      });
    });
  });
  context('Parse dbUri', function() {
    it('Should parse dbUri correctly', function() {
      var dbSetting = settingObj.db;
      expect(dbSetting.uri).to.equal(setting.dbUri);
      expect(dbSetting.server).to.equal("postgress://postgres@server:port");
      expect(dbSetting.name).to.equal("dbname");
    });

    context('When parse dbUri', function() {

      beforeEach(function(done) {
        settingTmp = {};
        settingTmp.port = setting.port;
        settingTmp.username = setting.username;
        settingTmp.password = setting.password;
        settingTmp.dbUri = setting.dbUri + '/';
        settingTmp.apiServerUri = setting.apiServerUri;
        dbSetting = configSetting(settingTmp).db;
        done();
      });
      it('Should filter the last slash', function() {

        expect(dbSetting.uri).to.equal(setting.dbUri);
        expect(dbSetting.server).to.equal("postgress://postgres@server:port");
        expect(dbSetting.name).to.equal("dbname");
      });

    });

    context('When parse dbUri is upper case', function() {

      beforeEach(function(done) {
        var settingTmp = {};
        settingTmp.port = setting.port;
        settingTmp.username = setting.username;
        settingTmp.password = setting.password;
        settingTmp.dbUri = setting.dbUri.toUpperCase();
        settingTmp.apiServerUri = setting.apiServerUri;
        dbSetting = configSetting(setting).db;
        done();
      });
      it('Should handle correctly ', function() {
        expect(dbSetting.uri).to.equal(setting.dbUri);
        expect(dbSetting.server).to.equal("postgress://postgres@server:port");
        expect(dbSetting.name).to.equal("dbname");
      });

    });
  });
  context('Parse apiServerUri', function() {
    beforeEach(function(done) {
      apiServerUriParsed = configSetting(setting).apiServerUri;
      done();
    });
    it('Should parse apiServerUri correctly', function() {
      expect(apiServerUriParsed).to.equal(setting.apiServerUri);
    });

    context('When parse apiServerUri', function() {
      beforeEach(function(done) {
        settingTmp = {};
        settingTmp.port = setting.port;
        settingTmp.username = setting.username;
        settingTmp.password = setting.password;
        settingTmp.dbUri = setting.dbUri;
        settingTmp.apiServerUri = setting.apiServerUri + '/';
        apiServerUriParsed = configSetting(settingTmp).apiServerUri;
        done();
      });
      it('Should filter the last slash ', function() {
        expect(apiServerUriParsed).to.equal(setting.apiServerUri);
      });
    });

    context('When parse dbUri is upper case', function() {
      beforeEach(function(done) {
        settingTmp = {};
        settingTmp.port = setting.port;
        settingTmp.username = setting.username;
        settingTmp.password = setting.password;
        settingTmp.dbUri = setting.dbUri;
        settingTmp.apiServerUri = setting.apiServerUri.toUpperCase();
        apiServerUriParsed = configSetting(setting).apiServerUri;
        done();
      });
      it('Should handle correctly', function() {
        expect(apiServerUriParsed).to.equal(setting.apiServerUri);
      });
    });
  });
});