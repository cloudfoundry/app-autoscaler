'use strict';

process.env.NODE_TLS_REJECT_UNAUTHORIZED = '0';
var path = require('path');

var sinon = require('sinon');
var path = require('path');
var API = require('../../app.js');
var app, sandbox, shutdownStub, processExitStub, forceShutdownStub;

before(function () {
  process.removeAllListeners('SIGUSR2');
  process.removeAllListeners('SIGINT');
})

describe('Gracefully Shutting down', function () {
  beforeEach(function () {
    app = API(path.join(__dirname, '../../config/settings.json'));
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
    app = API(path.join(__dirname, '../../config/settings.json'));
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

