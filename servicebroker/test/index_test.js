'use strict';

var supertest = require("supertest");
var should = require("should");
var fs = require('fs');
var path = require('path');

// This agent refers to PORT where program is runninng.
var catalog = JSON.parse(fs.readFileSync(path.join(__dirname, '../config/catalog.json'), 'utf8'));

// UNIT test begin

describe("RESTful API test suite", function() {
  var server;
  beforeEach(function() {
    server = require(path.join(__dirname, '../lib/index.js'), { bustCache: true });
  });

  afterEach(function(done) {
    server.close(done);
  });

  it("should return catalog json", function(done) {
    supertest(server)
      .get("/v2/catalog")
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
      .expect(404);
    done();
  })

});