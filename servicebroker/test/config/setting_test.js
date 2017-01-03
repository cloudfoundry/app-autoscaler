'use strict';

var path = require('path');
var expect = require('chai').expect;
var configSetting = require(path.join(__dirname, '../../lib/config/setting.js'));
var defaultConfig = {
  "port": 80,
  "username": "username",
  "password": "password",
  "db": {
    "maxConnections": 10,
    "minConnections": 0,
    "idleTimeout": 1000,
    "uri": "postgres://postgres@server:80/dbname"
  },
  "apiServerUri": "http://apiserveruri",
  "httpRequestTimeout": 5000,
  "tls": {
    "keyFile": "keyFilePath",
    "certFile": "certFilePath",
    "caCertFile": "caCertFilePath"
  }
}

var settingTmp = {};
var settings;

describe('config setting Test Suite', function() {
  beforeEach(function() {
    settings = configSetting(defaultConfig)
  });

  it('Should contain the default configuration', function() {
    expect(settings.port).to.equal(defaultConfig.port);
    expect(settings.username).to.equal(defaultConfig.username);
    expect(settings.password).to.equal(defaultConfig.password);
    expect(settings.apiServerUri).to.equal(defaultConfig.apiServerUri);
    expect(settings.db.uri).to.equal(defaultConfig.db.uri);
    expect(settings.db.server).to.equal('postgres://postgres@server:80');
    expect(settings.db.name).to.equal('dbname');
    expect(settings.db.maxConnections).to.equal(defaultConfig.db.maxConnections);
    expect(settings.db.minConnections).to.equal(defaultConfig.db.minConnections);
    expect(settings.db.idleTimeout).to.equal(defaultConfig.db.idleTimeout);
    expect(settings.httpRequestTimeout).to.equal(defaultConfig.httpRequestTimeout);
    expect(settings.tls.keyFile).to.equal(defaultConfig.tls.keyFile);
    expect(settings.tls.certFile).to.equal(defaultConfig.tls.certFile);
    expect(settings.tls.caCertFile).to.equal(defaultConfig.tls.caCertFile);
  });

  describe('validate', function() {
    context('When setting is correct', function() {
      it('Should return true', function() {
        expect(settings.validate().valid).to.equal(true);
      })
    });

    context('Validate port', function() {
      context('When port is null', function() {
        it('Should return false', function() {
          settings.port = null;
          expect(settings.validate().valid).to.equal(false);
        })
      });
      context('When port is undefined', function() {
        it('Should return false', function() {
          delete settings.port
          expect(settings.validate().valid).to.equal(false);
        })
      });
      context('When port is not an integer', function() {
        it('Should return false', function() {
          settings.port = "80";
          expect(settings.validate().valid).to.equal(false);
        })
      });
      context('When the port is out of range', function() {
        it('Should return false', function() {
          settings.port = 70000
          expect(settings.validate().valid).to.equal(false);
        })
      });
    });

    context('Validate username', function() {
      context('When username is null', function() {
        it('Should return false', function() {
          settings.username = null;
          expect(settings.validate().valid).to.equal(false);
        })
      });
      context('When username is undefined', function() {
        it('Should return false', function() {
          delete settings.username
          expect(settings.validate().valid).to.equal(false);
        })
      });
    });

    context('Validate password', function() {
      context('When password is null', function() {
        it('Should return false', function() {
          settings.password = null
          expect(settings.validate().valid).to.equal(false);
        })
      });
      context('When password is undefined', function() {
        it('Should return false', function() {
          delete settings.password
          expect(settings.validate().valid).to.equal(false);
        })
      });
    });
    context('Validate db.maxConnections', function() {
      context('When db.maxConnections is null', function() {
        it('Should return false', function() {
          settings.db.maxConnections = null;
          expect(settings.validate().valid).to.equal(false);
        })
      });
      context('When db.maxConnections is undefined', function() {
        it('Should return false', function() {
          delete settings.db.maxConnections
          expect(settings.validate().valid).to.equal(false);
        })
      });
      context('When db.maxConnections is not an integer', function() {
        it('Should return false', function() {
          settings.db.maxConnections = "10";
          expect(settings.validate().valid).to.equal(false);
        })
      });
      context('When the db.maxConnections is out of range', function() {
        it('Should return false', function() {
          settings.db.maxConnections = -10;
          expect(settings.validate().valid).to.equal(false);
        })
      });
    });

    context('Validate db.minConnections', function() {
      context('When minConnections is null', function() {
        it('Should return false', function() {
          settings.db.minConnections = null;
          expect(settings.validate().valid).to.equal(false);
        })
      });
      context('When db.minConnections is undefined', function() {
        it('Should return false', function() {
          delete settings.db.minConnections
          expect(settings.validate().valid).to.equal(false);
        })
      });
      context('When db.minConnections is not an integer', function() {
        it('Should return false', function() {
          settings.db.minConnections = "10";
          expect(settings.validate().valid).to.equal(false);
        })
      });
      context('When the db.minConnections is out of range', function() {
        it('Should return false', function() {
          settings.db.minConnections = -10;
          expect(settings.validate().valid).to.equal(false);
        })
      });
    });

    context('Validate db.idleTimeout', function() {
      context('When db.idleTimeout is null', function() {
        it('Should return false', function() {
          settings.db.idleTimeout = null;
          expect(settings.validate().valid).to.equal(false);
        })
      });
      context('When db.idleTimeout is undefined', function() {
        it('Should return false', function() {
          delete settings.db.idleTimeout
          expect(settings.validate().valid).to.equal(false);
        })
      });
      context('When db.idleTimeout is not an integer', function() {
        it('Should return false', function() {
          settings.db.idleTimeout = "1000";
          expect(settings.validate().valid).to.equal(false);
        })
      });
      context('When the db.idleTimeout is out of range', function() {
        it('Should return false', function() {
          settings.db.idleTimeout = -1000;
          expect(settings.validate().valid).to.equal(false);
        })
      });
    });
    context('Validate db.uri', function() {
      context('When db.uri is null', function() {
        it('Should return false', function() {
          settings.db.uri = null
          expect(settings.validate().valid).to.equal(false);
        })
      });
      context('When db.uri is undefined', function() {
        it('Should return false', function() {
          delete settings.db.uri
          expect(settings.validate().valid).to.equal(false);
        })
      });
    });

    context('Validate apiServerUri', function() {
      context('When apiServerUri is null', function() {
        it('Should return false', function() {
          settings.apiServerUri = null
          expect(settings.validate().valid).to.equal(false);
        })
      });
      context('When apiServerUri is undefined', function() {
        it('Should return false', function() {
          delete settings.apiServerUri
          expect(settings.validate().valid).to.equal(false);
        })
      });
    });
  });

  context('db.uri', function() {
    it('Should filter the last slash', function() {
      var dbSetting = configSetting({ db: { uri: defaultConfig.db.uri + '/' } }).db;
      expect(dbSetting.uri).to.equal(defaultConfig.db.uri);
      expect(dbSetting.server).to.equal("postgres://postgres@server:80");
      expect(dbSetting.name).to.equal("dbname");
    });

    context('When the db.uri is mixed case', function() {
      it('Should be lowercased', function() {
        var dbSetting = configSetting({ db: { uri: defaultConfig.db.uri.toUpperCase() } }).db;
        expect(dbSetting.uri).to.equal(defaultConfig.db.uri);
      });
    });
  });

  context('apiServerUri', function() {
    it('Should filter the last slash', function() {
      var apiSetting = configSetting({ apiServerUri: defaultConfig.apiServerUri + '/' }).apiServerUri;
      expect(apiSetting).to.equal(defaultConfig.apiServerUri);
    });

    context('When the apiServerUri is upper case', function() {
      it('Should be lowercased', function() {
        var apiSetting = configSetting({ apiServerUri: defaultConfig.apiServerUri.toUpperCase() }).apiServerUri;
        expect(apiSetting).to.equal(defaultConfig.apiServerUri);
      });
    });
  });

  context('Validate httpRequestTimeout', function() {
      context('When httpRequestTimeout is null', function() {
        it('Should return false', function() {
          settings.httpRequestTimeout = null;
          expect(settings.validate().valid).to.equal(false);
        })
      });
      context('When httpRequestTimeout is undefined', function() {
        it('Should return false', function() {
          delete settings.httpRequestTimeout
          expect(settings.validate().valid).to.equal(false);
        })
      });
      context('When httpRequestTimeout is not an integer', function() {
        it('Should return false', function() {
          settings.httpRequestTimeout = "1000";
          expect(settings.validate().valid).to.equal(false);
        })
      });
      context('When the httpRequestTimeout is out of range', function() {
        it('Should return false', function() {
          settings.httpRequestTimeout = -1
          expect(settings.validate().valid).to.equal(false);
        })
      });
  });

  context('Validate tls', function(){
    context('When tls is null', function(){
      it('Should return false',function(){
        settings.tls = null;
        expect(settings.validate().valid).to.equal(false)
      });
    });
    context('When tls is undefined', function(){
      it('Should return false',function(){
        delete settings.tls;
        expect(settings.validate().valid).to.equal(false)
      });
    });
    context('When tls is not an object', function(){
      it('Should return false',function(){
        settings.tls = "notobject";
        expect(settings.validate().valid).to.equal(false)
      });
    });
  });

  context('Validate tls.keyFile', function(){
    context('When tls.keyFile is null', function(){
      it('Should return false',function(){
        settings.tls.keyFile = null;
        expect(settings.validate().valid).to.equal(false)
      });
    });
    context('When tls.keyFile is undefined', function(){
      it('Should return false',function(){
        delete settings.tls.keyFile;
        expect(settings.validate().valid).to.equal(false)
      });
    });
  });

  context('Validate tls.certFile', function(){
    context('When tls.certFile is null', function(){
      it('Should return false',function(){
        settings.tls.certFile = null;
        expect(settings.validate().valid).to.equal(false)
      });
    });
    context('When tls.certFile is undefined', function(){
      it('Should return false',function(){
        delete settings.tls.certFile;
        expect(settings.validate().valid).to.equal(false)
      });
    });
  });

  context('Validate tls.caCertFile', function(){
    context('When tls.caCertFile is null', function(){
      it('Should return false',function(){
        settings.tls.caCertFile = null;
        expect(settings.validate().valid).to.equal(false)
      });
    });
    context('When tls.caCertFile is undefined', function(){
      it('Should return false',function(){
        delete settings.tls.caCertFile;
        expect(settings.validate().valid).to.equal(false)
      });
    });
  });
});
