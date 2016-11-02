'use strict';

var supertest = require("supertest");
var uuid = require('uuid');

var fs = require('fs');
var path = require('path');
var BrokerServer = require(path.join(__dirname, '../lib/server.js'));
var configFilePath = path.join(__dirname, '../config/settings.json');
var settings = require(path.join(__dirname, '../lib/config/setting.js'))((JSON.parse(
  fs.readFileSync(configFilePath, 'utf8'))));
var auth = new Buffer(settings.username + ":" + settings.password).toString('base64')

describe("Invalid path for RESTful API", function() {
  var server;
  before(function() {
    server = BrokerServer(configFilePath);
  });

  after(function(done) {
    server.close(done);
  });


  it("should resturn 404 when path is invalid", function(done) {
    supertest(server)
      .get("/v2/invalidpath")
      .set("Authorization", "Basic " + auth)
      .expect(404, done);
  });

});

describe("Auth for RESTful API", function() {
  var server;
  before(function() {
    server = BrokerServer(configFilePath);
  });

  after(function(done) {
    server.close(done);
  });

  it("should return 401 when no auth info provided", function(done) {
    supertest(server)
      .get("/v2/catalog")
      .expect(401, done);
  });

  it("should return 401 when incorrect user/password provided", function(done) {
    supertest(server)
      .get("/v2/catalog")
      .set("Authorization", "Basic " + new Buffer("incorrectuser:incorrectpassword").toString('base64'))
      .expect(401, done);
  });

  it("should return 401 when incorrect user provided", function(done) {
    supertest(server)
      .get("/v2/catalog")
      .set("Authorization", "Basic " + new Buffer("incorrectuser" + ":" + settings.password).toString('base64'))
      .expect(401, done);

  });

  it("should return 401 when incorrect password provided", function(done) {
    supertest(server)
      .get("/v2/catalog")
      .set("Authorization", "Basic " + new Buffer(settings.username + ":" + "incorrectpassword").toString('base64'))
      .expect(401, done);

  });

});
