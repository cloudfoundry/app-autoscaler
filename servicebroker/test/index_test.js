'use strict';

var supertest = require("supertest");
var should = require("should");
var fs = require('fs');
var path = require('path');
var uuid = require('uuid');

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
      .expect(200)
      .end(function(err, res) {
        res.status.should.equal(200);
        JSON.stringify(res.body).should.equal(JSON.stringify(catalog));
        done();
      });
  });

  it("should return 201 when add a new service instance", function(done) {
    var serviceId = uuid.v4();
    var orgId = uuid.v4();
    var spaceId = uuid.v4();
    supertest(server)
      .put("/v2/service_instances/" + serviceId)
      .set("Authorization", "Basic " + auth)
      .send({ "organization_guid": orgId, "space_guid": spaceId })
      .expect('Content-Type', /json/)
      .expect(201, {
        dashboard_url: ''
      }, done);
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

  it("should return 401 when no auth info provided", function(done) {
    supertest(server)
      .get("/v2/catalog")
      .expect(401, done);
  });

  it("should return 401 when incorrect user/password provided", function(done) {
    supertest(server)
      .get("/v2/catalog")
      .set("Authorization", "Basic " + new Buffer("incorrectuser:incorrectpassword").toString('base64') )
      .expect(401, done);
  });

  it("should return 401 when incorrect user provided", function(done) {
    supertest(server)
      .get("/v2/catalog")
      .set("Authorization", "Basic " + new Buffer("incorrectuser" + ":" + settings.password).toString('base64') )
      .expect(401, done);

  });

  it("should return 401 when incorrect password provided", function(done) {
    supertest(server)
      .get("/v2/catalog")
      .set("Authorization", "Basic " + new Buffer(settings.user + ":" + "incorrectpassword").toString('base64'))
      .expect(401, done);

  });



});