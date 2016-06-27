'use strict';

var supertest = require("supertest");
var should = require("should");
var fs = require('fs');
var path = require('path');
var uuid = require('uuid');

// This agent refers to PORT where program is runninng.
var settings = JSON.parse(
  fs.readFileSync(path.join(__dirname, '../config/settings.json'), 'utf8'));
var auth = new Buffer(settings.username + ":" + settings.password).toString('base64')

describe("Invalid path for RESTful API", function() {
  var server;
  before(function() {
    delete require.cache[require.resolve('../lib/index.js')];
    server = require(path.join(__dirname, '../lib/index.js'));
  });

  after(function(done) {
    server.close(done);
  });


  it("should resturn 404 when path is invalid", function(done) {
    supertest(server)
      .get("/v2/invalidpath")
      .set("Authorization", "Basic " + auth)
      .expect(404)
      .end(function(err, res) {
        if (err) return done(err);
        done();
      });
  });

});

describe("Auth for RESTful API", function() {
  var server;
  before(function() {
    delete require.cache[require.resolve('../lib/index.js')];
    server = require(path.join(__dirname, '../lib/index.js'));
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