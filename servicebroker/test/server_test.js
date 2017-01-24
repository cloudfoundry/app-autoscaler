'use strict';

var supertest = require("supertest");
var uuid = require('uuid');
var sinon = require('sinon');
var fs = require('fs');
var path = require('path');
var BrokerServer = require(path.join(__dirname, '../lib/server.js'));
var configFilePath = path.join(__dirname, '../config/settings.json');
var settings = require(path.join(__dirname, '../lib/config/setting.js'))((JSON.parse(
  fs.readFileSync(configFilePath, 'utf8'))));
var auth = new Buffer(settings.username + ":" + settings.password).toString('base64')
process.env.NODE_TLS_REJECT_UNAUTHORIZED = '0';

var app, sandbox, shutdownStub, processExitStub, forceShutdownStub;

before(function () {
  process.removeAllListeners('SIGUSR2');
  process.removeAllListeners('SIGINT');
})

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

describe('Gracefully Shutting down', function () {
  beforeEach(function () {
    app = BrokerServer(configFilePath);
    sandbox = sinon.sandbox.create();
    shutdownStub = sandbox.stub(app, 'shutdown');
    processExitStub = sandbox.stub(process, 'exit');
  })

  afterEach(function (done) {
    sandbox.restore();
    app.close(done);
  })

  it('receives SIGINT signal', function (done) {
    process.once('SIGINT', function () {
      sinon.assert.calledOnce(shutdownStub);
      done();
    });
    process.kill(process.pid, 'SIGINT');
  })

  it('receives SIGUSR2 signal', function (done) {
    process.once('SIGUSR2', function () {
      sinon.assert.calledOnce(shutdownStub);
      done();
    });
    process.kill(process.pid, 'SIGUSR2');
  })
})

describe('Forcefully Shutting down', function () {
  beforeEach(function () {
    app = BrokerServer(configFilePath);
    sandbox = sinon.sandbox.create({ useFakeTimers: true });
    shutdownStub = sandbox.stub(app, 'shutdown', function () {
      setTimeout(function () { }, 15 * 1000);
    });
    forceShutdownStub = sandbox.stub(app, 'forceShutdown');
    processExitStub = sandbox.stub(process, 'exit');
  })

  afterEach(function (done) {
    sandbox.restore();
    app.close(done);
  })

  it('receives SIGINT signal', function (done) {
    process.once('SIGINT', function () {
      sinon.assert.notCalled(forceShutdownStub);
      sandbox.clock.tick(11 * 1000);
      sinon.assert.calledOnce(forceShutdownStub);
      done();
    });
    process.kill(process.pid, 'SIGINT');
  })

  it('receives SIGUSR2 signal', function (done) {
    process.once('SIGUSR2', function () {
      sinon.assert.notCalled(forceShutdownStub);
      sandbox.clock.tick(11 * 1000);
      sinon.assert.calledOnce(forceShutdownStub);
      done();
    });
    process.kill(process.pid, 'SIGUSR2');
  })
})

