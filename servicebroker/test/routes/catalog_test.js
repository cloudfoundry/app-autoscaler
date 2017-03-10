'use strict';
var supertest = require("supertest");
var expect = require('chai').expect;

var fs = require('fs');
var path = require('path');
var BrokerServer = require(path.join(__dirname, '../../lib/server.js'));
var configFilePath = path.join(__dirname, '../../config/settings.json');
var settings = require(path.join(__dirname, '../../lib/config/setting.js'))((JSON.parse(
  fs.readFileSync(configFilePath, 'utf8'))));
var auth = new Buffer(settings.username + ":" + settings.password).toString('base64');

describe('getCatalog RESTful API', function() {
  var server;
  beforeEach(function() {
    server = BrokerServer(configFilePath);
  });

  afterEach(function(done) {
    server.close(done);
  });

  it("should return catalog json", function(done) {
    supertest(server)
      .get("/v2/catalog")
      .set("Authorization", "Basic " + auth)
      .expect(200)
      .expect("Content-type", /json/)
      .end(function(err, res) {
        expect(JSON.stringify(res.body)).to.be.equal(JSON.stringify(settings.services));
        done();
      });
  });

});
