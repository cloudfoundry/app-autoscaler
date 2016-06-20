'use strict';

var supertest = require("supertest");
var should = require("should");
var fs = require('fs');
var path = require('path');

// This agent refers to PORT where program is runninng.
var catalog = JSON.parse(fs.readFileSync(path.join(__dirname, '../config/catalog.json'), 'utf8'));
var settings = JSON.parse(
  fs.readFileSync(path.join(__dirname, '../config/settings.json'), 'utf8'));
var auth = new Buffer(settings.user + ":" + settings.password).toString('base64')

// UNIT test begin
describe("RESTful API test suite", function() {
  var server;
  beforeEach(function() {
    delete require.cache[require.resolve('../lib/index.js')];
    server = require(path.join(__dirname, '../lib/index.js'));
  });

  afterEach(function(done) {
    server.close(done);
  });

  it("should return catalog json", function(done) {
    supertest(server)
      .get("/v2/catalog")
      .set("Authorization", "Basic " + auth)
      .expect("Content-type", /json/)
      .expect(200) // THis is HTTP response
      .end(function(err, res) {
        // HTTP status should be 200
        res.status.should.equal(200);
        JSON.stringify(res.body).length.should.equal(JSON.stringify(catalog).length);
        res.body.services[0].id.should.equal(catalog.services[0].id);
        res.body.services[0].name.should.equal(catalog.services[0].name);
        res.body.services[0].description.should.equal(catalog.services[0].description);
        res.body.services[0].plans[0].id.should.equal(catalog.services[0].plans[0].id);
        res.body.services[0].plans[0].name.should.equal(catalog.services[0].plans[0].name);
        res.body.services[0].plans[0].description.should.equal(catalog.services[0].plans[0].description);
        done();
      });
  });

  it("should resturn 404 when path is invalid", function(done) {
    supertest(server)
      .get("/v2/catalogs")
      .set("Authorization", "Basic " + auth)
      .expect(404)
      .end(function(err, res) {
        if (err) return done(err);
        done();
      });
  });

  it("should return 401 when no auth info provided", function(done) {
    supertest(server)
      .get("/v2/catalog")
      .expect(401, done);
  });

  it("should return 401 when incorrect auth format provided", function(done) {
    supertest(server)
      .get("/v2/catalog")
      .set("Authorization", "invalidauth"  )
      .expect(401, done);
  });

  it("should return 401 when auth info is not encoded with base64", function(done) {
    supertest(server)
      .get("/v2/catalog")
      .set("Authorization", "Basic " + settings.user + ":" + settings.password )
      .expect(401, done);
  });

  it("should return 401 when incorrect user/password provided", function(done) {
    supertest(server)
      .get("/v2/catalog")
      .set("Authorization", "Basic " + new Buffer("incorrectuser:incorrectpassword").toString('base64') )
      .expect(401, done);
  });

  it("should return 401 when incorrect password provided", function(done) {
    supertest(server)
      .get("/v2/catalog")
      .set("Authorization", "Basic " + new Buffer(settings.user+ ":" + "incorrectpassword").toString('base64') )
      .expect(401, done);

  });

  it("should return 401 when incorrect basic auth format provided", function(done) {
    supertest(server)
      .get("/v2/catalog")
      .set("Authorization", "Basic " + new Buffer(settings.user + settings.password).toString('base64') )
      .expect(401, done);
  });


});