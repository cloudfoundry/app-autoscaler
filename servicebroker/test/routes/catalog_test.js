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
var catalog = JSON.parse(fs.readFileSync(path.join(__dirname, '../../config/catalog.json'), 'utf8'));

describe('getCatalog RESTful API', function() {
  var servers, publicServer, internalServer;
  beforeEach(function() {
    servers = BrokerServer(settings, catalog, function(){});
    publicServer = servers.publicServer;
    internalServer = servers.internalServer;
  });

  after(function(done) {
    publicServer.close(function(){
      internalServer.close(done);
    })
  });

  it("should return catalog json", function(done) {

    supertest(publicServer)
      .get("/v2/catalog")
      .set("Authorization", "Basic " + auth)
      .expect(200)
      .expect("Content-type", /json/)
      .end(function(err, res) {
        expect(JSON.stringify(res.body)).to.be.equal(JSON.stringify(catalog));
        done();
      });
  });

});
