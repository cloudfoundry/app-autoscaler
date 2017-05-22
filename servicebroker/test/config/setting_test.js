'use strict';

var path = require('path');
var expect = require('chai').expect;
var configSetting = require(path.join(__dirname, '../../lib/config/setting.js'));
var defaultConfig;

var settingTmp = {};
var settings;

describe('config setting Test Suite', function() {
  beforeEach(function() {
    defaultConfig = {
      "port": 80,
      "username": "username",
      "password": "password",
      "db": {
        "maxConnections": 10,
        "minConnections": 0,
        "idleTimeout": 1000,
        "uri": "postgres://postgres@server:80/dbname"
      },
      "apiserver": {
        "uri": "https://autoscaler.boshlite.com",
        "tls": {
          "keyFile":"../test-certs/api.key",
          "certFile":"../test-certs/api.crt",
          "caCertFile":"../test-certs/autoscaler-ca.crt"
          }
      },
      "httpRequestTimeout": 5000,
      "tls": {
        "keyFile": "keyFilePath",
        "certFile": "certFilePath",
        "caCertFile": "caCertFilePath"
      },
      "serviceCatalogPath" : "catalogPath"
    };
    settings = configSetting(defaultConfig);
  });

  it('Should contain the default configuration', function() {
    expect(settings.port).to.equal(defaultConfig.port);
    expect(settings.username).to.equal(defaultConfig.username);
    expect(settings.password).to.equal(defaultConfig.password);
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
    expect(settings.apiserver.uri).to.equal(defaultConfig.apiserver.uri);
    expect(settings.apiserver.tls.keyFile).to.equal(defaultConfig.apiserver.tls.keyFile);
    expect(settings.apiserver.tls.caCertFile).to.equal(defaultConfig.apiserver.tls.caCertFile);
    expect(settings.apiserver.tls.certFile).to.equal(defaultConfig.apiserver.tls.certFile);
    expect(settings.serviceCatalogPath).to.equal(defaultConfig.serviceCatalogPath);
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
          expect(settings.validate().message).to.equal("The port must be a number");
        })
      });
      context('When port is undefined', function() {
        it('Should return false', function() {
          delete settings.port
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("The port must be a number");
        })
      });
      context('When port is not an integer', function() {
        it('Should return false', function() {
          settings.port = "80";
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("The port must be a number");
        })
      });
      context('When the port is out of range', function() {
        it('Should return false', function() {
          settings.port = 70000
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("The value of port must between 0 and 65536");
        })
      });
    });

    context('Validate username', function() {
      context('When username is null', function() {
        it('Should return false', function() {
          settings.username = null;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("username is required");
        })
      });
      context('When username is undefined', function() {
        it('Should return false', function() {
          delete settings.username
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("username is required");
        })
      });
    });

    context('Validate password', function() {
      context('When password is null', function() {
        it('Should return false', function() {
          settings.password = null
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("password is required");
        })
      });
      context('When password is undefined', function() {
        it('Should return false', function() {
          delete settings.password
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("password is required");
        })
      });
    });
    context('Validate db.maxConnections', function() {
      context('When db.maxConnections is null', function() {
        it('Should return false', function() {
          settings.db.maxConnections = null;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("maxConnections is required");
        })
      });
      context('When db.maxConnections is undefined', function() {
        it('Should return false', function() {
          delete settings.db.maxConnections
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("maxConnections is required");
        })
      });
      context('When db.maxConnections is not an integer', function() {
        it('Should return false', function() {
          settings.db.maxConnections = "10";
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("maxConnections is required");
        })
      });
      context('When the db.maxConnections is out of range', function() {
        it('Should return false', function() {
          settings.db.maxConnections = -10;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("maxConnections must be greater than 0");
        })
      });
    });

    context('Validate db.minConnections', function() {
      context('When minConnections is null', function() {
        it('Should return false', function() {
          settings.db.minConnections = null;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("minConnections is required");
        })
      });
      context('When db.minConnections is undefined', function() {
        it('Should return false', function() {
          delete settings.db.minConnections
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("minConnections is required");
        })
      });
      context('When db.minConnections is not an integer', function() {
        it('Should return false', function() {
          settings.db.minConnections = "10";
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("minConnections is required");
        })
      });
      context('When the db.minConnections is out of range', function() {
        it('Should return false', function() {
          settings.db.minConnections = -10;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("minConnections must be greater than 0");
        })
      });
    });

    context('Validate db.idleTimeout', function() {
      context('When db.idleTimeout is null', function() {
        it('Should return false', function() {
          settings.db.idleTimeout = null;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("idleTimeout is required");
        })
      });
      context('When db.idleTimeout is undefined', function() {
        it('Should return false', function() {
          delete settings.db.idleTimeout
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("idleTimeout is required");
        })
      });
      context('When db.idleTimeout is not an integer', function() {
        it('Should return false', function() {
          settings.db.idleTimeout = "1000";
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("idleTimeout is required");
        })
      });
      context('When the db.idleTimeout is out of range', function() {
        it('Should return false', function() {
          settings.db.idleTimeout = -1000;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("idleTimeout must be greater than 0");
        })
      });
    });
    context('Validate db.uri', function() {
      context('When db.uri is null', function() {
        it('Should return false', function() {
          settings.db.uri = null
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("dbUri is required");
        })
      });
      context('When db.uri is undefined', function() {
        it('Should return false', function() {
          delete settings.db.uri
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("dbUri is required");
        })
      });
    });
  });

  context('db.uri', function() {
    it('Should filter the last slash', function() {
      var dbSetting = configSetting({ db: { uri: defaultConfig.db.uri + '/' },apiserver: { uri: defaultConfig.apiserver.uri, tls: defaultConfig.apiserver.tls } }).db;
      expect(dbSetting.uri).to.equal(defaultConfig.db.uri);
      expect(dbSetting.server).to.equal("postgres://postgres@server:80");
      expect(dbSetting.name).to.equal("dbname");
    });

    context('When the db.uri is mixed case', function() {
      it('Should be lowercased', function() {
        var dbSetting = configSetting({ db: { uri: defaultConfig.db.uri.toUpperCase() },apiserver: { uri: defaultConfig.apiserver.uri, tls: defaultConfig.apiserver.tls } }).db;
        expect(dbSetting.uri).to.equal(defaultConfig.db.uri);
      });
    });
  });

  context('apiserver uri', function() {
    it('Should filter the last slash', function() {
      var apiSetting = configSetting({ apiserver : { uri: defaultConfig.apiserver.uri + '/' }}).apiserver;
      expect(apiSetting.uri).to.equal(defaultConfig.apiserver.uri);
    });

    context('When the apiserver uri is upper case', function() {
      it('Should be lowercased', function() {
        var apiSetting = configSetting({ apiserver : { uri: defaultConfig.apiserver.uri.toUpperCase() }}).apiserver;
        expect(apiSetting.uri).to.equal(defaultConfig.apiserver.uri);
      });
    });
  });

  context('Validate apiserver uri', function() {
      context('When apiserver uri is null', function() {
        it('Should return false', function() {
          settings.apiserver.uri = null
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("apiserver uri is required");
        })
      });
      context('When apiserver uri is undefined', function() {
        it('Should return false', function() {
          delete settings.apiserver.uri
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("apiserver uri is required");
        })
      });
    });

  context('Validate httpRequestTimeout', function() {
      context('When httpRequestTimeout is null', function() {
        it('Should return false', function() {
          settings.httpRequestTimeout = null;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("The httpRequestTimeout must be a number");
        })
      });
      context('When httpRequestTimeout is undefined', function() {
        it('Should return false', function() {
          delete settings.httpRequestTimeout
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("The httpRequestTimeout must be a number");
        })
      });
      context('When httpRequestTimeout is not an integer', function() {
        it('Should return false', function() {
          settings.httpRequestTimeout = "1000";
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("The httpRequestTimeout must be a number");
        })
      });
      context('When the httpRequestTimeout is out of range', function() {
        it('Should return false', function() {
          settings.httpRequestTimeout = -1
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("The value of httpRequestTimeout must be greater than 0");
        })
      });
  });

  context('Validate tls', function(){
    context('When tls is null', function(){
      it('Should return false',function(){
        settings.tls = null;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("tls is required");
      });
    });
    context('When tls is undefined', function(){
      it('Should return false',function(){
        delete settings.tls;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("tls is required");
      });
    });
    context('When tls is not an object', function(){
      it('Should return false',function(){
        settings.tls = "notobject";
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("tls must be an object");
      });
    });
  });

  context('Validate tls.keyFile', function(){
    context('When tls.keyFile is null', function(){
      it('Should return false',function(){
        settings.tls.keyFile = null;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("tls.keyFile is required");
      });
    });
    context('When tls.keyFile is undefined', function(){
      it('Should return false',function(){
        delete settings.tls.keyFile;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("tls.keyFile is required");
      });
    });
  });

  context('Validate tls.certFile', function(){
    context('When tls.certFile is null', function(){
      it('Should return false',function(){
        settings.tls.certFile = null;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("tls.certFile is required");
      });
    });
    context('When tls.certFile is undefined', function(){
      it('Should return false',function(){
        delete settings.tls.certFile;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("tls.certFile is required");
      });
    });
  });

  context('Validate tls.caCertFile', function(){
    context('When tls.caCertFile is null', function(){
      it('Should return false',function(){
        settings.tls.caCertFile = null;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("tls.caCertFile is required");
      });
    });
    context('When tls.caCertFile is undefined', function(){
      it('Should return false',function(){
        delete settings.tls.caCertFile;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("tls.caCertFile is required");
      });
    });
  });

context('Validate apiserver client tls.keyFile', function(){
    context('When apiserver client tls.keyFile is null', function(){
      it('Should return false',function(){
        settings.apiserver.tls.keyFile = null;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("apiserver tls.keyFile is required");
      });
    });
    context('When apiserver client tls.keyFile is undefined', function(){
      it('Should return false',function(){
        delete settings.apiserver.tls.keyFile;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("apiserver tls.keyFile is required");
      });
    });
  });

  context('Validate apiserver client tls.certFile', function(){
    context('When apiserver client tls.certFile is null', function(){
      it('Should return false',function(){
        settings.apiserver.tls.certFile = null;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("apiserver tls.certFile is required");
      });
    });
    context('When apiserver client tls.certFile is undefined', function(){
      it('Should return false',function(){
        delete settings.apiserver.tls.certFile;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("apiserver tls.certFile is required");
      });
    });
  });

  context('Validate apiserver client tls.caCertFile', function(){
    context('When apiserver client tls.caCertFile is null', function(){
      it('Should return false',function(){
        settings.apiserver.tls.caCertFile = null;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("apiserver tls.caCertFile is required");
      });
    });
    context('When apiserver client tls.caCertFile is undefined', function(){
      it('Should return false',function(){
        delete settings.apiserver.tls.caCertFile;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("apiserver tls.caCertFile is required");
      });
    });
  });

  context('Validate apiserver client tls', function(){
    context('When apiserver client tls is null', function(){
      it('Should return false',function(){
        settings.apiserver.tls = null;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("apiserver tls is required");
      });
    });
    context('When apiserver client tls  is undefined', function(){
      it('Should return false',function(){
        delete settings.apiserver.tls;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("apiserver tls is required");
      });
    });
    context('When apiserver client tls is not an object', function(){
      it('Should return false',function(){
        settings.apiserver.tls = "notobject";
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("apiserver tls must be an object");
      });
    });
});

context('Validate Service Catalog', function(){
    context('When catalogPath is null', function(){
      it('Should return false',function(){
        settings.serviceCatalogPath = null;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("serviceCatalogPath is required");
      });
    });
    context('When catalogPath is undefined', function(){
      it('Should return false',function(){
        delete settings.serviceCatalogPath;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("serviceCatalogPath is required");
      });
    });
  });

});
