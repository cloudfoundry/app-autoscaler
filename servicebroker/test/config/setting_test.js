'use strict';

var path = require('path');
var expect = require('chai').expect;
var fs = require('fs');
var configSetting = require(path.join(__dirname, '../../lib/config/setting.js'));
var fakeDBCAPath = path.join(__dirname, '../../../test-certs/autoscaler-ca.crt');
var defaultConfig;

var settingTmp = {};
var settings;

describe('config setting Test Suite', function() {
  beforeEach(function() {
    defaultConfig = {
      "port": 88,
      "publicPort": 80,
      "healthPort": 8081,
      "username": "username",
      "password": "password",
      "enableCustomMetrics": true,
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
      "publicTls": {
        "keyFile": "keyFilePath",
        "certFile": "certFilePath",
        "caCertFile": "caCertFilePath"
      },
      "tls": {
        "keyFile": "keyFilePath_internal",
        "certFile": "certFilePath_internal",
        "caCertFile": "caCertFilePath"
      },
      "serviceCatalogPath" : "catalogPath",
      "schemaValidationPath" : "schemaPath",
      "dashboardRedirectUri": "https://dashboard-redirect-uri-settings.example.com",
      "customMetricsUrl": "https://metrics.example.com/v1/metrics"
    };
    settings = configSetting(defaultConfig);
  });

  it('Should contain the default configuration', function() {
    expect(settings.port).to.equal(defaultConfig.port);
    expect(settings.publicPort).to.equal(defaultConfig.publicPort);
    expect(settings.healthPort).to.equal(defaultConfig.healthPort);

    expect(settings.username).to.equal(defaultConfig.username);
    expect(settings.password).to.equal(defaultConfig.password);
    expect(settings.enableCustomMetrics).to.equal(true);

    expect(settings.db.uri).to.equal(defaultConfig.db.uri);
    expect(settings.db.maxConnections).to.equal(defaultConfig.db.maxConnections);
    expect(settings.db.minConnections).to.equal(defaultConfig.db.minConnections);
    expect(settings.db.idleTimeout).to.equal(defaultConfig.db.idleTimeout);
    expect(settings.httpRequestTimeout).to.equal(defaultConfig.httpRequestTimeout);

    expect(settings.tls.keyFile).to.equal(defaultConfig.tls.keyFile);
    expect(settings.tls.certFile).to.equal(defaultConfig.tls.certFile);
    expect(settings.tls.caCertFile).to.equal(defaultConfig.tls.caCertFile);

    expect(settings.publicTls.keyFile).to.equal(defaultConfig.publicTls.keyFile);
    expect(settings.publicTls.certFile).to.equal(defaultConfig.publicTls.certFile);
    expect(settings.publicTls.caCertFile).to.equal(defaultConfig.publicTls.caCertFile);

    expect(settings.apiserver.uri).to.equal(defaultConfig.apiserver.uri);
    expect(settings.apiserver.tls.keyFile).to.equal(defaultConfig.apiserver.tls.keyFile);
    expect(settings.apiserver.tls.caCertFile).to.equal(defaultConfig.apiserver.tls.caCertFile);
    expect(settings.apiserver.tls.certFile).to.equal(defaultConfig.apiserver.tls.certFile);

    expect(settings.serviceCatalogPath).to.equal(defaultConfig.serviceCatalogPath);
    expect(settings.schemaValidationPath).to.equal(defaultConfig.schemaValidationPath);
    expect(settings.dashboardRedirectUri).to.equal(defaultConfig.dashboardRedirectUri);
    expect(settings.customMetricsUrl).to.equal(defaultConfig.customMetricsUrl);
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
          expect(settings.validate().message).to.equal("port is required");
        })
      });
      context('When port is undefined', function() {
        it('Should return false', function() {
          delete settings.port;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("port is required");
        })
      });
      context('When port is not an integer', function() {
        it('Should return false', function() {
          settings.port = "80";
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("port must be a number");
        })
      });
      context('When the port is out of range', function() {
        it('Should return false', function() {
          settings.port = 70000;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("value of port must between 1 and 65535");
        })
      });
    });

    context('Validate publicPort', function() {
      context('When publicPort is null', function() {
        it('Should return false', function() {
          settings.publicPort = null;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("publicPort is required");
        })
      });
      context('When publicPort is undefined', function() {
        it('Should return false', function() {
          delete settings.publicPort;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("publicPort is required");
        })
      });
      context('When publicPort is not an integer', function() {
        it('Should return false', function() {
          settings.publicPort = "80";
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("publicPort must be a number");
        })
      });
      context('When the publicPort is out of range', function() {
        it('Should return false', function() {
          settings.publicPort = 70000;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("value of publicPort must between 1 and 65535");
        })
      });
    });

    context('Validate healthPort', function() {
      context('When healthPort is null', function() {
        it('Should return false', function() {
          settings.healthPort = null;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("healthPort is required");
        })
      });
      context('When healthPort is undefined', function() {
        it('Should return false', function() {
          delete settings.healthPort;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("healthPort is required");
        })
      });
      context('When healthPort is not an integer', function() {
        it('Should return false', function() {
          settings.healthPort = "80";
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("healthPort must be a number");
        })
      });
      context('When the healthPort is out of range', function() {
        it('Should return false', function() {
          settings.healthPort = 70000;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("value of healthPort must between 0 and 65535");
        })
      });
    });

    context('Validate internal port, public port and health port', function() {
      context('When publicPort is equal to internal port', function() {
        it('Should return false', function() {
          settings.publicPort = 3002;
          settings.port = 3002;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("internal port, public port and health port should be different");
        })
      });
      context('When publicPort is equal to health port', function() {
        it('Should return false', function() {
          settings.publicPort = 3002;
          settings.healthPort = 3002;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("internal port, public port and health port should be different");
        })
      });
      context('When internal port is equal to health port', function() {
        it('Should return false', function() {
          settings.port = 3002;
          settings.healthPort = 3002;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("internal port, public port and health port should be different");
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
          delete settings.username;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("username is required");
        })
      });
      context('When username is not a string', function() {
        it('Should return false', function() {
          settings.username = 1234;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("username must be a string");
        })
      });
    });

    context('Validate password', function() {
      context('When password is null', function() {
        it('Should return false', function() {
          settings.password = null;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("password is required");
        })
      });
      context('When password is undefined', function() {
        it('Should return false', function() {
          delete settings.password;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("password is required");
        })
      });
      context('When password is not a string', function() {
        it('Should return false', function() {
          settings.password = 12345
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("password must be a string");
        })
      });
    });
    context('Validate enableCustomMetrics', function() {
      context('When enableCustomMetrics is null', function() {
        it('Should return false', function() {
          settings.enableCustomMetrics = null;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("enableCustomMetrics is required");
        })
      });
      context('When enableCustomMetrics is undefined', function() {
        it('Should return false', function() {
          delete settings.enableCustomMetrics;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("enableCustomMetrics is required");
        })
      });
      context('When enableCustomMetrics is not a boolean', function() {
        it('Should return false', function() {
          settings.enableCustomMetrics = 1234;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("enableCustomMetrics must be a boolean");
        })
      });
    });
    context('Validate db.maxConnections', function() {
      context('When db.maxConnections is null', function() {
        it('Should return false', function() {
          settings.db.maxConnections = null;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("db.maxConnections is required");
        })
      });
      context('When db.maxConnections is undefined', function() {
        it('Should return false', function() {
          delete settings.db.maxConnections;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("db.maxConnections is required");
        })
      });
      context('When db.maxConnections is not an integer', function() {
        it('Should return false', function() {
          settings.db.maxConnections = "10";
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("db.maxConnections must be a number");
        })
      });
      context('When the db.maxConnections is out of range', function() {
        it('Should return false', function() {
          settings.db.maxConnections = -10;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("db.maxConnections must be greater than 0");
        })
      });
    });

    context('Validate db.minConnections', function() {
      context('When db.minConnections is null', function() {
        it('Should return false', function() {
          settings.db.minConnections = null;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("db.minConnections is required");
        })
      });
      context('When db.minConnections is undefined', function() {
        it('Should return false', function() {
          delete settings.db.minConnections;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("db.minConnections is required");
        })
      });
      context('When db.minConnections is not an integer', function() {
        it('Should return false', function() {
          settings.db.minConnections = "10";
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("db.minConnections must be a number");
        })
      });
      context('When the db.minConnections is out of range', function() {
        it('Should return false', function() {
          settings.db.minConnections = -10;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("db.minConnections must be greater than or equal to 0");
        })
      });
    });

    context('Validate db.idleTimeout', function() {
      context('When db.idleTimeout is null', function() {
        it('Should return false', function() {
          settings.db.idleTimeout = null;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("db.idleTimeout is required");
        })
      });
      context('When db.idleTimeout is undefined', function() {
        it('Should return false', function() {
          delete settings.db.idleTimeout;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("db.idleTimeout is required");
        })
      });
      context('When db.idleTimeout is not an integer', function() {
        it('Should return false', function() {
          settings.db.idleTimeout = "1000";
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("db.idleTimeout must be a number");
        })
      });
      context('When the db.idleTimeout is out of range', function() {
        it('Should return false', function() {
          settings.db.idleTimeout = -1000;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("db.idleTimeout must be greater than 0");
        })
      });
    });
    context('Validate db.uri', function() {
      context('When db.uri is null', function() {
        it('Should return false', function() {
          settings.db.uri = null;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("db.uri is required");
        })
      });
      context('When db.uri is undefined', function() {
        it('Should return false', function() {
          delete settings.db.uri;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("db.uri is required");
        })
      });
      context('When db.uri is not a string', function() {
        it('Should return false', function() {
          settings.db.uri = 12345;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("db.uri must be a string");
        })
      });
    });
  });

  context('db.uri', function() {
    it('Should filter the last slash', function() {
      var dbSetting = configSetting({ db: { uri: defaultConfig.db.uri + '/' },apiserver: { uri: defaultConfig.apiserver.uri, tls: defaultConfig.apiserver.tls } }).db;
      expect(dbSetting.uri).to.equal(defaultConfig.db.uri);
    });

  });
  context('dbUri ssl', function(){
    context('When there is no ssl setting in db.uri', function() {
      it('Should return null ssl', function() {
        var dbSetting = configSetting({ db: { uri: defaultConfig.db.uri }, apiserver: { uri: defaultConfig.apiserver.uri, tls: defaultConfig.apiserver.tls } }).db;
        expect(dbSetting.uri).to.equal(defaultConfig.db.uri);
        expect(dbSetting.ssl).to.be.null;
      });
    });

    context('When sslmode is disable', function() {
      it('Should return null ssl', function() {
        var dbSetting = configSetting({ db: { uri: defaultConfig.db.uri }, apiserver: { uri: defaultConfig.apiserver.uri, tls: defaultConfig.apiserver.tls } }).db;
        expect(dbSetting.uri).to.equal(defaultConfig.db.uri);
        expect(dbSetting.ssl).to.be.null;
      });
    });

    context('When sslmode is prefer', function() {
      it('Should return ssl with rejectUnauthorized=false', function() {
        var dbSetting = configSetting({ db: { uri: defaultConfig.db.uri + '?sslmode=prefer' }, apiserver: { uri: defaultConfig.apiserver.uri, tls: defaultConfig.apiserver.tls } }).db;
        expect(dbSetting.uri).to.equal(defaultConfig.db.uri + '?sslmode=prefer');
        expect(dbSetting.ssl).to.deep.equal({rejectUnauthorized: false});
      });
    });
    context('When sslmode is require', function() {
      it('Should return ssl with rejectUnauthorized=false', function() {
        var dbSetting = configSetting({ db: { uri: defaultConfig.db.uri + '?sslmode=require' }, apiserver: { uri: defaultConfig.apiserver.uri, tls: defaultConfig.apiserver.tls } }).db;
        expect(dbSetting.uri).to.equal(defaultConfig.db.uri + '?sslmode=require');
        expect(dbSetting.ssl).to.deep.equal({rejectUnauthorized: false});
      });
    });
    context('When sslmode is verify-ca', function() {
      context('When sslrootcert is not provided', function() {
        it('Should return ssl with rejectUnauthorized=true', function() {
          var dbSetting = configSetting({ db: { uri: defaultConfig.db.uri + '?sslmode=verify-ca' }, apiserver: { uri: defaultConfig.apiserver.uri, tls: defaultConfig.apiserver.tls } }).db;
          expect(dbSetting.uri).to.equal(defaultConfig.db.uri + '?sslmode=verify-ca');
          expect(dbSetting.ssl).to.deep.equal({ rejectUnauthorized: true });
        });
      });

      context('When sslrootcert is provided', function() {
        it('Should return ssl with rejectUnauthorized=true', function() {
          var dbSetting = configSetting({ db: { uri: defaultConfig.db.uri + '?sslmode=verify-ca&sslrootcert=' + fakeDBCAPath }, apiserver: { uri: defaultConfig.apiserver.uri, tls: defaultConfig.apiserver.tls } }).db;
          expect(dbSetting.uri).to.equal(defaultConfig.db.uri + '?sslmode=verify-ca&sslrootcert=' + fakeDBCAPath);
          expect(dbSetting.ssl).to.deep.equal({ rejectUnauthorized: true, ca: fs.readFileSync(fakeDBCAPath) });
        });
      });
    });

    context('When sslmode is verify-full', function() {
      context('When sslrootcert is not provided', function() {
        it('Should return ssl with rejectUnauthorized=true', function() {
          var dbSetting = configSetting({ db: { uri: defaultConfig.db.uri + '?sslmode=verify-full' }, apiserver: { uri: defaultConfig.apiserver.uri, tls: defaultConfig.apiserver.tls } }).db;
          expect(dbSetting.uri).to.equal(defaultConfig.db.uri + '?sslmode=verify-full');
          expect(dbSetting.ssl).to.deep.equal({ rejectUnauthorized: true });
        });
      });

      context('When sslrootcert is provided', function() {
        it('Should return ssl with rejectUnauthorized=true', function() {
          var dbSetting = configSetting({ db: { uri: defaultConfig.db.uri + '?sslmode=verify-full&sslrootcert=' + fakeDBCAPath }, apiserver: { uri: defaultConfig.apiserver.uri, tls: defaultConfig.apiserver.tls } }).db;
          expect(dbSetting.uri).to.equal(defaultConfig.db.uri + '?sslmode=verify-full&sslrootcert=' + fakeDBCAPath);
          expect(dbSetting.ssl).to.deep.equal({ rejectUnauthorized: true, ca: fs.readFileSync(fakeDBCAPath) });
        });
      });
    });
  })
  context('apiserver.uri', function() {
    it('Should filter the last slash', function() {
      var apiSetting = configSetting({ apiserver : { uri: defaultConfig.apiserver.uri + '/' }}).apiserver;
      expect(apiSetting.uri).to.equal(defaultConfig.apiserver.uri);
    });

    context('When the apiserver.uri is upper case', function() {
      it('Should be lowercased', function() {
        var apiSetting = configSetting({ apiserver : { uri: defaultConfig.apiserver.uri.toUpperCase() }}).apiserver;
        expect(apiSetting.uri).to.equal(defaultConfig.apiserver.uri);
      });
    });
  });

  context('Validate apiserver.uri', function() {
      context('When apiserver.uri is null', function() {
        it('Should return false', function() {
          settings.apiserver.uri = null;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("apiserver.uri is required");
        })
      });
      context('When apiserver.uri is undefined', function() {
        it('Should return false', function() {
          delete settings.apiserver.uri;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("apiserver.uri is required");
        })
      });
      context('When apiserver.uri is not a string', function() {
        it('Should return false', function() {
          settings.apiserver.uri = 1234;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("apiserver.uri must be a string");
        })
      });
    });

  context('Validate httpRequestTimeout', function() {
      context('When httpRequestTimeout is null', function() {
        it('Should return false', function() {
          settings.httpRequestTimeout = null;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("httpRequestTimeout is required");
        })
      });
      context('When httpRequestTimeout is undefined', function() {
        it('Should return false', function() {
          delete settings.httpRequestTimeout;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("httpRequestTimeout is required");
        })
      });
      context('When httpRequestTimeout is not an integer', function() {
        it('Should return false', function() {
          settings.httpRequestTimeout = "1000";
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("httpRequestTimeout must be a number");
        })
      });
      context('When the httpRequestTimeout is out of range', function() {
        it('Should return false', function() {
          settings.httpRequestTimeout = -1;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("value of httpRequestTimeout must be greater than 0");
        })
      });
  });

  context('Validate tls', function(){
    context('When tls is null', function(){
      it('Should return true',function(){
        settings.tls = null;
        expect(settings.validate().valid).to.equal(true);
      });
    });
    context('When tls is undefined', function(){
      it('Should return true',function(){
        delete settings.tls;
        expect(settings.validate().valid).to.equal(true);
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
    context('When tls.keyFile is not a string', function(){
      it('Should return false',function(){
        settings.tls.keyFile = 1234;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("tls.keyFile must be a string");
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
    context('When tls.certFile is not a string', function(){
      it('Should return false',function(){
        settings.tls.certFile = 1234;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("tls.certFile must be a string");
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
    context('When tls.caCertFile is not a string', function(){
      it('Should return false',function(){
        settings.tls.caCertFile = 1234;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("tls.caCertFile must be a string");
      });
    });
  });

  context('Validate publicTls', function(){
    context('When publicTls is null', function(){
      it('Should return true',function(){
        settings.publicTls = null;
        expect(settings.validate().valid).to.equal(true);
      });
    });
    context('When publicTls is undefined', function(){
      it('Should return true',function(){
        delete settings.publicTls;
        expect(settings.validate().valid).to.equal(true);
      });
    });
    context('When publicTls is not an object', function(){
      it('Should return false',function(){
        settings.publicTls = "notobject";
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("publicTls must be an object");
      });
    });
  });

  context('Validate publicTls.keyFile', function(){
    context('When publicTls.keyFile is null', function(){
      it('Should return false',function(){
        settings.publicTls.keyFile = null;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("publicTls.keyFile is required");
      });
    });
    context('When publicTls.keyFile is undefined', function(){
      it('Should return false',function(){
        delete settings.publicTls.keyFile;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("publicTls.keyFile is required");
      });
    });
    context('When publicTls.keyFile is not a string', function(){
      it('Should return false',function(){
        settings.publicTls.keyFile = 1234;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("publicTls.keyFile must be a string");
      });
    });
  });

  context('Validate publicTls.certFile', function(){
    context('When publicTls.certFile is null', function(){
      it('Should return false',function(){
        settings.publicTls.certFile = null;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("publicTls.certFile is required");
      });
    });
    context('When publicTls.certFile is undefined', function(){
      it('Should return false',function(){
        delete settings.publicTls.certFile;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("publicTls.certFile is required");
      });
    });
    context('When publicTls.certFile is not a string', function(){
      it('Should return false',function(){
        settings.publicTls.certFile = 1234;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("publicTls.certFile must be a string");
      });
    });
  });

  context('Validate publicTls.caCertFile', function(){
    context('When publicTls.caCertFile is null', function(){
      it('Should return false',function(){
        settings.publicTls.caCertFile = null;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("publicTls.caCertFile is required");
      });
    });
    context('When publicTls.caCertFile is undefined', function(){
      it('Should return false',function(){
        delete settings.publicTls.caCertFile;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("publicTls.caCertFile is required");
      });
    });
    context('When publicTls.caCertFile is not a string', function(){
      it('Should return false',function(){
        settings.publicTls.caCertFile = 1234;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("publicTls.caCertFile must be a string");
      });
    });
  });

context('Validate apiserver client tls.keyFile', function(){
    context('When apiserver client tls.keyFile is null', function(){
      it('Should return false',function(){
        settings.apiserver.tls.keyFile = null;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("apiserver.tls.keyFile is required");
      });
    });
    context('When apiserver client tls.keyFile is undefined', function(){
      it('Should return false',function(){
        delete settings.apiserver.tls.keyFile;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("apiserver.tls.keyFile is required");
      });
    });
    context('When apiserver client tls.keyFile is not a string', function(){
      it('Should return false',function(){
        settings.apiserver.tls.keyFile = 1234;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("apiserver.tls.keyFile must be a string");
      });
    });
  });

  context('Validate apiserver client tls.certFile', function(){
    context('When apiserver client tls.certFile is null', function(){
      it('Should return false',function(){
        settings.apiserver.tls.certFile = null;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("apiserver.tls.certFile is required");
      });
    });
    context('When apiserver client tls.certFile is undefined', function(){
      it('Should return false',function(){
        delete settings.apiserver.tls.certFile;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("apiserver.tls.certFile is required");
      });
    });
    context('When apiserver client tls.certFile is not a string', function(){
      it('Should return false',function(){
        settings.apiserver.tls.certFile = 1234;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("apiserver.tls.certFile must be a string");
      });
    });
  });

  context('Validate apiserver client tls.caCertFile', function(){
    context('When apiserver client tls.caCertFile is null', function(){
      it('Should return false',function(){
        settings.apiserver.tls.caCertFile = null;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("apiserver.tls.caCertFile is required");
      });
    });
    context('When apiserver client tls.caCertFile is undefined', function(){
      it('Should return false',function(){
        delete settings.apiserver.tls.caCertFile;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("apiserver.tls.caCertFile is required");
      });
    });
    context('When apiserver client tls.caCertFile is not a string', function(){
      it('Should return false',function(){
        settings.apiserver.tls.caCertFile = 1234;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("apiserver.tls.caCertFile must be a string");
      });
    });
  });

  context('Validate apiserver client tls', function(){
    context('When apiserver client tls is null', function(){
      it('Should return true',function(){
        settings.apiserver.tls = null;
        expect(settings.validate().valid).to.equal(true);
      });
    });
    context('When apiserver client tls  is undefined', function(){
      it('Should return true',function(){
        delete settings.apiserver.tls;
        expect(settings.validate().valid).to.equal(true);
      });
    });
    context('When apiserver client tls is not an object', function(){
      it('Should return false',function(){
        settings.apiserver.tls = "notobject";
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("apiserver.tls must be an object");
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
    context('When catalogPath is not a string', function(){
      it('Should return false',function(){
        settings.serviceCatalogPath = 1234;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("serviceCatalogPath must be a string");
      });
    });
  });

  context('Validate Dashboard Redirect Uri', function(){
    context('When dashboardRedirectUri is null', function(){
      it('Should return true',function(){
        settings.dashboardRedirectUri = null;
        expect(settings.validate().valid).to.equal(true);
      });
    });
    context('When dashboardRedirectUri is undefined', function(){
      it('Should return true',function(){
        delete settings.dashboardRedirectUri;
        expect(settings.validate().valid).to.equal(true);
      });
    });
    context('When dashboardRedirectUri is not a string', function(){
      it('Should return false',function(){
        settings.dashboardRedirectUri = 1234;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("dashboardRedirectUri must be a string");
      });
    });
  });

  context('Validate Custom Metrics Forwarder Uri', function(){
    context('When customMetricsUrl is null', function(){
      it('Should return true',function(){
        settings.customMetricsUrl = null;
        expect(settings.validate().valid).to.equal(true);
      });
    });
    context('When customMetricsUrl is undefined', function(){
      it('Should return true',function(){
        delete settings.customMetricsUrl;
        expect(settings.validate().valid).to.equal(true);
      });
    });
    context('When customMetricsUrl is not a string', function(){
      it('Should return false',function(){
        settings.customMetricsUrl = 1234;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("customMetricsUrl must be a string");
      });
    });
  });

});
