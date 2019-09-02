'use strict';

var path = require('path');
var expect = require('chai').expect;
var fs = require('fs');
var configSetting = require(path.join(__dirname, '../../../lib/config/setting.js'));
var fakeDBCAPath = path.join(__dirname, '../../../../test-certs/autoscaler-ca.crt');
var defaultConfig, defaultConfigTemplate;
var settings;

describe('config setting Test Suite', function() {
  beforeEach(function() {
    defaultConfigTemplate = {
      "port": 8080,
      "publicPort": 8081,
      "healthPort": 8083,
      "infoFilePath": "../api/config/info.json",
      "cfApi": "api.bosh-lite.com",
      "cfClientId": "admin",
      "cfClientSecret": "admin-secret",
      "skipSSLValidation": false,
      "cacheTTL": 200,
      "db": {
        "maxConnections": 10,
        "minConnections": 0,
        "idleTimeout": 1000,
        "uri": "postgres://postgres@server:80/dbname",
      },
      "scheduler": {
        "uri": "schedulerUri",
        "tls": {
          "keyFile": "keyFilePath",
          "certFile": "certFilePath",
          "caCertFile": "caCertFilePath"
        }
      },
      "scalingEngine": {
        "uri": "scalingEngineUri",
        "tls": {
          "keyFile": "keyFilePath",
          "certFile": "certFilePath",
          "caCertFile": "caCertFilePath"
        }
      },
      "metricsCollector": {
        "uri": "metricsCollectorUri",
        "tls": {
          "keyFile": "keyFilePath",
          "certFile": "certFilePath",
          "caCertFile": "caCertFilePath"
        }
      },
      "eventGenerator": {
        "uri": "eventGeneratorUri",
        "tls": {
          "keyFile": "keyFilePath",
          "certFile": "certFilePath",
          "caCertFile": "caCertFilePath"
        }
      },
      "serviceOffering": {
        "enabled": true,
        "serviceBroker": {
          "uri": "serviceBrokerUri",
          "tls": {
            "keyFile": "keyFilePath",
            "certFile": "certFilePath",
            "caCertFile": "caCertFilePath"
          }
        },
      },
      "tls": {
        "keyFile": "keyFilePath",
        "certFile": "certFilePath",
        "caCertFile": "caCertFilePath"
      },
      "publicTls": {
        "keyFile": "keyFilePath",
        "certFile": "certFilePath",
        "caCertFile": "caCertFilePath"
      },
      "minBreachDurationSecs": 1,
      "minCoolDownSecs": 1,
      "httpClientTimeout": 5000
    }
    //setting.js will modifty defaultConfigTemplate, here clone defaultConfigTemplate to defaultConfig first
    defaultConfig = JSON.parse(JSON.stringify(defaultConfigTemplate))
    settings = configSetting(defaultConfigTemplate);
  });

  it('Should contain the default configuration', function() {
    expect(settings.port).to.equal(defaultConfig.port);

    expect(settings.publicPort).to.equal(defaultConfig.publicPort);
    expect(settings.healthPort).to.equal(defaultConfig.healthPort);

    expect(settings.cfApi).to.equal("https://" + defaultConfig.cfApi.toLowerCase());
    expect(settings.cfClientId).to.equal(defaultConfig.cfClientId);
    expect(settings.cfClientSecret).to.equal(defaultConfig.cfClientSecret);
    expect(settings.infoFilePath).to.equal(defaultConfig.infoFilePath);
    expect(settings.skipSSLValidation).to.equal(false);
    expect(settings.cacheTTL).to.equal(defaultConfig.cacheTTL);

    expect(settings.db.maxConnections).to.equal(defaultConfig.db.maxConnections);
    expect(settings.db.minConnections).to.equal(defaultConfig.db.minConnections);
    expect(settings.db.idleTimeout).to.equal(defaultConfig.db.idleTimeout);
    expect(settings.db.uri).to.equal(defaultConfig.db.uri);

    expect(settings.tls.keyFile).to.equal(defaultConfig.tls.keyFile);
    expect(settings.tls.certFile).to.equal(defaultConfig.tls.certFile);
    expect(settings.tls.caCertFile).to.equal(defaultConfig.tls.caCertFile);

    expect(settings.publicTls.keyFile).to.equal(defaultConfig.publicTls.keyFile);
    expect(settings.publicTls.certFile).to.equal(defaultConfig.publicTls.certFile);
    expect(settings.publicTls.caCertFile).to.equal(defaultConfig.publicTls.caCertFile);

    expect(settings.scheduler.uri).to.equal("https://" + defaultConfig.scheduler.uri.toLowerCase());
    expect(settings.scheduler.tls.keyFile).to.equal(defaultConfig.scheduler.tls.keyFile);
    expect(settings.scheduler.tls.caCertFile).to.equal(defaultConfig.scheduler.tls.caCertFile);
    expect(settings.scheduler.tls.certFile).to.equal(defaultConfig.scheduler.tls.certFile);

    expect(settings.scalingEngine.uri).to.equal("https://" + defaultConfig.scalingEngine.uri.toLowerCase());
    expect(settings.scalingEngine.tls.keyFile).to.equal(defaultConfig.scalingEngine.tls.keyFile);
    expect(settings.scalingEngine.tls.caCertFile).to.equal(defaultConfig.scalingEngine.tls.caCertFile);
    expect(settings.scalingEngine.tls.certFile).to.equal(defaultConfig.scalingEngine.tls.certFile);

    expect(settings.metricsCollector.uri).to.equal("https://" + defaultConfig.metricsCollector.uri.toLowerCase());
    expect(settings.metricsCollector.tls.keyFile).to.equal(defaultConfig.metricsCollector.tls.keyFile);
    expect(settings.metricsCollector.tls.caCertFile).to.equal(defaultConfig.metricsCollector.tls.caCertFile);
    expect(settings.metricsCollector.tls.certFile).to.equal(defaultConfig.metricsCollector.tls.certFile);

    expect(settings.eventGenerator.uri).to.equal("https://" + defaultConfig.eventGenerator.uri.toLowerCase());
    expect(settings.eventGenerator.tls.keyFile).to.equal(defaultConfig.eventGenerator.tls.keyFile);
    expect(settings.eventGenerator.tls.caCertFile).to.equal(defaultConfig.eventGenerator.tls.caCertFile);
    expect(settings.eventGenerator.tls.certFile).to.equal(defaultConfig.eventGenerator.tls.certFile);

    expect(settings.serviceOffering.enabled).to.equal(true);
    expect(settings.serviceOffering.serviceBroker.uri).to.equal("https://" + defaultConfig.serviceOffering.serviceBroker.uri.toLowerCase());
    expect(settings.serviceOffering.serviceBroker.tls.keyFile).to.equal(defaultConfig.serviceOffering.serviceBroker.tls.keyFile);
    expect(settings.serviceOffering.serviceBroker.tls.caCertFile).to.equal(defaultConfig.serviceOffering.serviceBroker.tls.caCertFile);
    expect(settings.serviceOffering.serviceBroker.tls.certFile).to.equal(defaultConfig.serviceOffering.serviceBroker.tls.certFile);

    expect(settings.httpClientTimeout).to.equal(defaultConfig.httpClientTimeout)
  });

  describe('URL handling', function() {
    context('cfapi.uri', function() {
      it("Should add http if no protocol", function() {
        var apiSetting = configSetting({ cfApi: defaultConfig.cfApi, db: { uri: defaultConfig.db.uri }, scheduler: { uri: defaultConfig.scheduler.uri, tls: defaultConfig.scheduler.tls }, scalingEngine: { uri: defaultConfig.scalingEngine.uri, tls: defaultConfig.scalingEngine.tls }, metricsCollector: { uri: defaultConfig.metricsCollector.uri, tls: defaultConfig.metricsCollector.tls }, serviceOffering: { enabled: defaultConfig.serviceOffering.enabled, serviceBroker: { uri: defaultConfig.serviceOffering.serviceBroker.uri, tls: defaultConfig.serviceOffering.serviceBroker.tls } } });
        expect(apiSetting.cfApi).to.equal("https://" + defaultConfig.cfApi);
      });
      it("Should filter the last slash", function() {
        var apiSetting = configSetting({ cfApi: defaultConfig.cfApi + "/", db: { uri: defaultConfig.db.uri }, scheduler: { uri: defaultConfig.scheduler.uri, tls: defaultConfig.scheduler.tls }, scalingEngine: { uri: defaultConfig.scalingEngine.uri, tls: defaultConfig.scalingEngine.tls }, metricsCollector: { uri: defaultConfig.metricsCollector.uri, tls: defaultConfig.metricsCollector.tls }, serviceOffering: { enabled: true, serviceBroker: { uri: defaultConfig.serviceOffering.serviceBroker.uri, tls: defaultConfig.serviceOffering.serviceBroker.tls } } });
        expect(apiSetting.cfApi).to.equal("https://" + defaultConfig.cfApi);
      });
    });

    context('db.uri', function() {
      it('Should filter the last slash', function() {
        var dbSetting = configSetting({ db: { uri: defaultConfig.db.uri + '/' }, scheduler: { uri: defaultConfig.scheduler.uri, tls: defaultConfig.scheduler.tls }, scalingEngine: { uri: defaultConfig.scalingEngine.uri, tls: defaultConfig.scalingEngine.tls }, metricsCollector: { uri: defaultConfig.metricsCollector.uri, tls: defaultConfig.metricsCollector.tls }, serviceOffering: { enabled: true, serviceBroker: { uri: defaultConfig.serviceOffering.serviceBroker.uri, tls: defaultConfig.serviceOffering.serviceBroker.tls } } }).db;
        expect(dbSetting.uri).to.equal(defaultConfig.db.uri);
      });
    });

    context('scheduler.uri', function() {

      it('Should add http if no protocol', function() {
        var apiSetting = configSetting({ scheduler: { uri: defaultConfig.scheduler.uri }, scalingEngine: { uri: defaultConfig.scalingEngine.uri, tls: defaultConfig.scalingEngine.tls }, metricsCollector: { uri: defaultConfig.metricsCollector.uri, tls: defaultConfig.metricsCollector.tls }, eventGenerator: { uri: defaultConfig.eventGenerator.uri }, serviceOffering: { enabled: true, serviceBroker: { uri: defaultConfig.serviceOffering.serviceBroker.uri, tls: defaultConfig.serviceOffering.serviceBroker.tls } } }).scheduler;
        expect(apiSetting.uri).to.equal("https://" + defaultConfig.scheduler.uri.toLowerCase());
      });

      it('Should filter the last slash', function() {
        var apiSetting = configSetting({ scheduler: { uri: defaultConfig.scheduler.uri + '/' }, scalingEngine: { uri: defaultConfig.scalingEngine.uri, tls: defaultConfig.scalingEngine.tls }, metricsCollector: { uri: defaultConfig.metricsCollector.uri, tls: defaultConfig.metricsCollector.tls }, eventGenerator: { uri: defaultConfig.eventGenerator.uri }, serviceOffering: { enabled: true, serviceBroker: { uri: defaultConfig.serviceOffering.serviceBroker.uri, tls: defaultConfig.serviceOffering.serviceBroker.tls } } }).scheduler;
        expect(apiSetting.uri).to.equal("https://" + defaultConfig.scheduler.uri.toLowerCase());
      });

      context('When the scheduler.uri is upper case', function() {
        it('Should be lowercased', function() {
          var apiSetting = configSetting({ scheduler: { uri: defaultConfig.scheduler.uri.toUpperCase() }, scalingEngine: { uri: defaultConfig.scalingEngine.uri, tls: defaultConfig.scalingEngine.tls }, metricsCollector: { uri: defaultConfig.metricsCollector.uri, tls: defaultConfig.metricsCollector.tls }, eventGenerator: { uri: defaultConfig.eventGenerator.uri }, serviceOffering: { enabled: true, serviceBroker: { uri: defaultConfig.serviceOffering.serviceBroker.uri, tls: defaultConfig.serviceOffering.serviceBroker.tls } } }).scheduler;
          expect(apiSetting.uri).to.equal("https://" + defaultConfig.scheduler.uri.toLowerCase());
        });
      });
    });
    context('scalingEngine uri', function() {

      it('Should add http if no protocol', function() {
        var apiSetting = configSetting({ scalingEngine: { uri: defaultConfig.scalingEngine.uri }, scheduler: { uri: defaultConfig.scheduler.uri, tls: defaultConfig.scheduler.tls }, metricsCollector: { uri: defaultConfig.metricsCollector.uri, tls: defaultConfig.metricsCollector.tls }, eventGenerator: { uri: defaultConfig.eventGenerator.uri }, serviceOffering: { enabled: true, serviceBroker: { uri: defaultConfig.serviceOffering.serviceBroker.uri, tls: defaultConfig.serviceOffering.serviceBroker.tls } } }).scalingEngine;
        expect(apiSetting.uri).to.equal("https://" + defaultConfig.scalingEngine.uri.toLowerCase());
      });

      it('Should filter the last slash', function() {
        var apiSetting = configSetting({ scalingEngine: { uri: defaultConfig.scalingEngine.uri + '/' }, scheduler: { uri: defaultConfig.scheduler.uri, tls: defaultConfig.scheduler.tls }, metricsCollector: { uri: defaultConfig.metricsCollector.uri, tls: defaultConfig.metricsCollector.tls }, eventGenerator: { uri: defaultConfig.eventGenerator.uri }, serviceOffering: { enabled: true, serviceBroker: { uri: defaultConfig.serviceOffering.serviceBroker.uri, tls: defaultConfig.serviceOffering.serviceBroker.tls } } }).scalingEngine;
        expect(apiSetting.uri).to.equal("https://" + defaultConfig.scalingEngine.uri.toLowerCase());
      });

      context('When the scalingEngine uri is upper case', function() {
        it('Should be lowercased', function() {
          var apiSetting = configSetting({ scalingEngine: { uri: defaultConfig.scalingEngine.uri.toUpperCase() }, scheduler: { uri: defaultConfig.scheduler.uri, tls: defaultConfig.scheduler.tls }, metricsCollector: { uri: defaultConfig.metricsCollector.uri, tls: defaultConfig.metricsCollector.tls }, eventGenerator: { uri: defaultConfig.eventGenerator.uri }, serviceOffering: { enabled: true, serviceBroker: { uri: defaultConfig.serviceOffering.serviceBroker.uri, tls: defaultConfig.serviceOffering.serviceBroker.tls } } }).scalingEngine;
          expect(apiSetting.uri).to.equal("https://" + defaultConfig.scalingEngine.uri.toLowerCase());
        });
      });
    });
    context('metricsCollector uri', function() {

      it('Should add http if no protocol', function() {
        var apiSetting = configSetting({ metricsCollector: { uri: defaultConfig.metricsCollector.uri }, scheduler: { uri: defaultConfig.scheduler.uri, tls: defaultConfig.scheduler.tls }, scalingEngine: { uri: defaultConfig.scalingEngine.uri, tls: defaultConfig.scalingEngine.tls }, eventGenerator: { uri: defaultConfig.eventGenerator.uri }, serviceOffering: { enabled: true, serviceBroker: { uri: defaultConfig.serviceOffering.serviceBroker.uri, tls: defaultConfig.serviceOffering.serviceBroker.tls } } }).metricsCollector;
        expect(apiSetting.uri).to.equal("https://" + defaultConfig.metricsCollector.uri.toLowerCase());
      });

      it('Should filter the last slash', function() {
        var apiSetting = configSetting({ metricsCollector: { uri: defaultConfig.metricsCollector.uri + '/' }, scheduler: { uri: defaultConfig.scheduler.uri, tls: defaultConfig.scheduler.tls }, scalingEngine: { uri: defaultConfig.scalingEngine.uri, tls: defaultConfig.scalingEngine.tls }, eventGenerator: { uri: defaultConfig.eventGenerator.uri }, serviceOffering: { enabled: true, serviceBroker: { uri: defaultConfig.serviceOffering.serviceBroker.uri, tls: defaultConfig.serviceOffering.serviceBroker.tls } } }).metricsCollector;
        expect(apiSetting.uri).to.equal("https://" + defaultConfig.metricsCollector.uri.toLowerCase());
      });

      context('When the metricsCollector uri is upper case', function() {
        it('Should be lowercased', function() {
          var apiSetting = configSetting({ metricsCollector: { uri: defaultConfig.metricsCollector.uri.toUpperCase() }, scheduler: { uri: defaultConfig.scheduler.uri, tls: defaultConfig.scheduler.tls }, scalingEngine: { uri: defaultConfig.scalingEngine.uri, tls: defaultConfig.scalingEngine.tls }, eventGenerator: { uri: defaultConfig.eventGenerator.uri }, serviceOffering: { enabled: true, serviceBroker: { uri: defaultConfig.serviceOffering.serviceBroker.uri, tls: defaultConfig.serviceOffering.serviceBroker.tls } } }).metricsCollector;
          expect(apiSetting.uri).to.equal("https://" + defaultConfig.metricsCollector.uri.toLowerCase());
        });
      });
    });
    context('eventGenerator uri', function() {

      it('Should add http if no protocol', function() {
        var apiSetting = configSetting({ eventGenerator: { uri: defaultConfig.eventGenerator.uri }, metricsCollector: { uri: defaultConfig.metricsCollector.uri }, scheduler: { uri: defaultConfig.scheduler.uri, tls: defaultConfig.scheduler.tls }, scalingEngine: { uri: defaultConfig.scalingEngine.uri, tls: defaultConfig.scalingEngine.tls }, serviceOffering: { enabled: true, serviceBroker: { uri: defaultConfig.serviceOffering.serviceBroker.uri, tls: defaultConfig.serviceOffering.serviceBroker.tls } } }).eventGenerator;
        expect(apiSetting.uri).to.equal("https://" + defaultConfig.eventGenerator.uri.toLowerCase());
      });

      it('Should filter the last slash', function() {
        var apiSetting = configSetting({ eventGenerator: { uri: defaultConfig.eventGenerator.uri + '/' }, metricsCollector: { uri: defaultConfig.metricsCollector.uri }, scheduler: { uri: defaultConfig.scheduler.uri, tls: defaultConfig.scheduler.tls }, scalingEngine: { uri: defaultConfig.scalingEngine.uri, tls: defaultConfig.scalingEngine.tls }, serviceOffering: { enabled: true, serviceBroker: { uri: defaultConfig.serviceOffering.serviceBroker.uri, tls: defaultConfig.serviceOffering.serviceBroker.tls } } }).eventGenerator;
        expect(apiSetting.uri).to.equal("https://" + defaultConfig.eventGenerator.uri.toLowerCase());
      });

      context('When the eventGenerator uri is upper case', function() {
        it('Should be lowercased', function() {
          var apiSetting = configSetting({ eventGenerator: { uri: defaultConfig.eventGenerator.uri.toUpperCase() }, metricsCollector: { uri: defaultConfig.metricsCollector.uri }, scheduler: { uri: defaultConfig.scheduler.uri, tls: defaultConfig.scheduler.tls }, scalingEngine: { uri: defaultConfig.scalingEngine.uri, tls: defaultConfig.scalingEngine.tls }, serviceOffering: { enabled: true, serviceBroker: { uri: defaultConfig.serviceOffering.serviceBroker.uri, tls: defaultConfig.serviceOffering.serviceBroker.tls } } }).eventGenerator;
          expect(apiSetting.uri).to.equal("https://" + defaultConfig.eventGenerator.uri.toLowerCase());
        });
      });
    });

    context('serviceBroker uri', function() {
      it('Should add http if no protocol', function() {
        var apiSetting = configSetting({ serviceOffering: { enabled: true, serviceBroker: { uri: defaultConfig.serviceOffering.serviceBroker.uri } }, scheduler: { uri: defaultConfig.scheduler.uri, tls: defaultConfig.scheduler.tls }, scalingEngine: { uri: defaultConfig.scalingEngine.uri, tls: defaultConfig.scalingEngine.tls }, metricsCollector: { uri: defaultConfig.metricsCollector.uri, tls: defaultConfig.metricsCollector.tls }, eventGenerator: { uri: defaultConfig.eventGenerator.uri } }).serviceOffering.serviceBroker;
        expect(apiSetting.uri).to.equal("https://" + defaultConfig.serviceOffering.serviceBroker.uri.toLowerCase());
      });

      it('Should filter the last slash', function() {
        var apiSetting = configSetting({ serviceOffering: { enabled: true, serviceBroker: { uri: defaultConfig.serviceOffering.serviceBroker.uri + '/' } }, scheduler: { uri: defaultConfig.scheduler.uri, tls: defaultConfig.scheduler.tls }, scalingEngine: { uri: defaultConfig.scalingEngine.uri, tls: defaultConfig.scalingEngine.tls }, metricsCollector: { uri: defaultConfig.metricsCollector.uri, tls: defaultConfig.metricsCollector.tls }, eventGenerator: { uri: defaultConfig.eventGenerator.uri } }).serviceOffering.serviceBroker;
        expect(apiSetting.uri).to.equal("https://" + defaultConfig.serviceOffering.serviceBroker.uri.toLowerCase());
      });

      context('When the serviceBroker uri is upper case', function() {
        it('Should be lowercased', function() {
          var apiSetting = configSetting({ serviceOffering: { enabled: true, serviceBroker: { uri: defaultConfig.serviceOffering.serviceBroker.uri.toUpperCase() } }, scheduler: { uri: defaultConfig.scheduler.uri, tls: defaultConfig.scheduler.tls }, scalingEngine: { uri: defaultConfig.scalingEngine.uri, tls: defaultConfig.scalingEngine.tls }, metricsCollector: { uri: defaultConfig.metricsCollector.uri, tls: defaultConfig.metricsCollector.tls }, eventGenerator: { uri: defaultConfig.eventGenerator.uri } }).serviceOffering.serviceBroker;
          expect(apiSetting.uri).to.equal("https://" + defaultConfig.serviceOffering.serviceBroker.uri.toLowerCase());
        });
      });
    });
  });
  describe('dbUri ssl', function(){
    context('When there is no ssl setting in db.uri', function() {
      it('Should return null ssl', function() {
        var dbSetting = configSetting({ db: { uri: defaultConfig.db.uri }, scheduler: { uri: defaultConfig.scheduler.uri, tls: defaultConfig.scheduler.tls }, scalingEngine: { uri: defaultConfig.scalingEngine.uri, tls: defaultConfig.scalingEngine.tls }, metricsCollector: { uri: defaultConfig.metricsCollector.uri, tls: defaultConfig.metricsCollector.tls }, serviceOffering: { enabled: true, serviceBroker: { uri: defaultConfig.serviceOffering.serviceBroker.uri, tls: defaultConfig.serviceOffering.serviceBroker.tls } } }).db;
        expect(dbSetting.uri).to.equal(defaultConfig.db.uri);
        expect(dbSetting.ssl).to.be.null;
      });
    });

    context('When sslmode is disable', function() {
      it('Should return null ssl', function() {
        var dbSetting = configSetting({ db: { uri: defaultConfig.db.uri }, scheduler: { uri: defaultConfig.scheduler.uri, tls: defaultConfig.scheduler.tls }, scalingEngine: { uri: defaultConfig.scalingEngine.uri, tls: defaultConfig.scalingEngine.tls }, metricsCollector: { uri: defaultConfig.metricsCollector.uri, tls: defaultConfig.metricsCollector.tls }, serviceOffering: { enabled: true, serviceBroker: { uri: defaultConfig.serviceOffering.serviceBroker.uri, tls: defaultConfig.serviceOffering.serviceBroker.tls } } }).db;
        expect(dbSetting.uri).to.equal(defaultConfig.db.uri);
        expect(dbSetting.ssl).to.be.null;
      });
    });

    context('When sslmode is prefer', function() {
      it('Should return ssl with rejectUnauthorized=false', function() {
        var dbSetting = configSetting({ db: { uri: defaultConfig.db.uri + '?sslmode=prefer' }, scheduler: { uri: defaultConfig.scheduler.uri, tls: defaultConfig.scheduler.tls }, scalingEngine: { uri: defaultConfig.scalingEngine.uri, tls: defaultConfig.scalingEngine.tls }, metricsCollector: { uri: defaultConfig.metricsCollector.uri, tls: defaultConfig.metricsCollector.tls }, serviceOffering: { enabled: true, serviceBroker: { uri: defaultConfig.serviceOffering.serviceBroker.uri, tls: defaultConfig.serviceOffering.serviceBroker.tls } } }).db;
        expect(dbSetting.uri).to.equal(defaultConfig.db.uri + '?sslmode=prefer');
        expect(dbSetting.ssl).to.deep.equal({rejectUnauthorized: false});
      });
    });
    context('When sslmode is require', function() {
      it('Should return ssl with rejectUnauthorized=false', function() {
        var dbSetting = configSetting({ db: { uri: defaultConfig.db.uri + '?sslmode=require' }, scheduler: { uri: defaultConfig.scheduler.uri, tls: defaultConfig.scheduler.tls }, scalingEngine: { uri: defaultConfig.scalingEngine.uri, tls: defaultConfig.scalingEngine.tls }, metricsCollector: { uri: defaultConfig.metricsCollector.uri, tls: defaultConfig.metricsCollector.tls }, serviceOffering: { enabled: true, serviceBroker: { uri: defaultConfig.serviceOffering.serviceBroker.uri, tls: defaultConfig.serviceOffering.serviceBroker.tls } } }).db;
        expect(dbSetting.uri).to.equal(defaultConfig.db.uri + '?sslmode=require');
        expect(dbSetting.ssl).to.deep.equal({rejectUnauthorized: false});
      });
    });
    context('When sslmode is verify-ca', function() {
      context('When sslrootcert is not provided', function() {
        it('Should return ssl with rejectUnauthorized=true', function() {
          var dbSetting = configSetting({ db: { uri: defaultConfig.db.uri + '?sslmode=verify-ca' }, scheduler: { uri: defaultConfig.scheduler.uri, tls: defaultConfig.scheduler.tls }, scalingEngine: { uri: defaultConfig.scalingEngine.uri, tls: defaultConfig.scalingEngine.tls }, metricsCollector: { uri: defaultConfig.metricsCollector.uri, tls: defaultConfig.metricsCollector.tls }, serviceOffering: { enabled: true, serviceBroker: { uri: defaultConfig.serviceOffering.serviceBroker.uri, tls: defaultConfig.serviceOffering.serviceBroker.tls } } }).db;
          expect(dbSetting.uri).to.equal(defaultConfig.db.uri + '?sslmode=verify-ca');
          expect(dbSetting.ssl).to.deep.equal({ rejectUnauthorized: true });
        });
      });

      context('When sslrootcert is provided', function() {
        it('Should return ssl with rejectUnauthorized=true', function() {
          var dbSetting = configSetting({ db: { uri: defaultConfig.db.uri + '?sslmode=verify-ca&sslrootcert=' + fakeDBCAPath }, scheduler: { uri: defaultConfig.scheduler.uri, tls: defaultConfig.scheduler.tls }, scalingEngine: { uri: defaultConfig.scalingEngine.uri, tls: defaultConfig.scalingEngine.tls }, metricsCollector: { uri: defaultConfig.metricsCollector.uri, tls: defaultConfig.metricsCollector.tls }, serviceOffering: { enabled: true, serviceBroker: { uri: defaultConfig.serviceOffering.serviceBroker.uri, tls: defaultConfig.serviceOffering.serviceBroker.tls } } }).db;
          expect(dbSetting.uri).to.equal(defaultConfig.db.uri + '?sslmode=verify-ca&sslrootcert=' + fakeDBCAPath);
          expect(dbSetting.ssl).to.deep.equal({ rejectUnauthorized: true, ca: fs.readFileSync(fakeDBCAPath) });
        });
      });
    });

    context('When sslmode is verify-full', function() {
      context('When sslrootcert is not provided', function() {
        it('Should return ssl with rejectUnauthorized=true', function() {
          var dbSetting = configSetting({ db: { uri: defaultConfig.db.uri + '?sslmode=verify-full' }, scheduler: { uri: defaultConfig.scheduler.uri, tls: defaultConfig.scheduler.tls }, scalingEngine: { uri: defaultConfig.scalingEngine.uri, tls: defaultConfig.scalingEngine.tls }, metricsCollector: { uri: defaultConfig.metricsCollector.uri, tls: defaultConfig.metricsCollector.tls }, serviceOffering: { enabled: true, serviceBroker: { uri: defaultConfig.serviceOffering.serviceBroker.uri, tls: defaultConfig.serviceOffering.serviceBroker.tls } } }).db;
          expect(dbSetting.uri).to.equal(defaultConfig.db.uri + '?sslmode=verify-full');
          expect(dbSetting.ssl).to.deep.equal({ rejectUnauthorized: true });
        });
      });

      context('When sslrootcert is provided', function() {
        it('Should return ssl with rejectUnauthorized=true', function() {
          var dbSetting = configSetting({ db: { uri: defaultConfig.db.uri + '?sslmode=verify-full&sslrootcert=' + fakeDBCAPath }, scheduler: { uri: defaultConfig.scheduler.uri, tls: defaultConfig.scheduler.tls }, scalingEngine: { uri: defaultConfig.scalingEngine.uri, tls: defaultConfig.scalingEngine.tls }, metricsCollector: { uri: defaultConfig.metricsCollector.uri, tls: defaultConfig.metricsCollector.tls }, serviceOffering: { enabled: true, serviceBroker: { uri: defaultConfig.serviceOffering.serviceBroker.uri, tls: defaultConfig.serviceOffering.serviceBroker.tls } } }).db;
          expect(dbSetting.uri).to.equal(defaultConfig.db.uri + '?sslmode=verify-full&sslrootcert=' + fakeDBCAPath);
          expect(dbSetting.ssl).to.deep.equal({ rejectUnauthorized: true, ca: fs.readFileSync(fakeDBCAPath) });
        });
      });
    });
  })
  describe('Validate', function() {
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
          expect(settings.validate().message).to.equal("value of port must be between 1 and 65535");
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
          expect(settings.validate().message).to.equal("value of publicPort must be between 1 and 65535");
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
          expect(settings.validate().message).to.equal("value of healthPort must be between 0 and 65535");
        })
      });
    });

    context('Validate cacheTTL', function() {
      context('When cacheTTL is null', function() {
        it('Should return false', function() {
          settings.cacheTTL = null;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("cacheTTL is required");
        })
      });
      context('When cacheTTL is undefined', function() {
        it('Should return false', function() {
          delete settings.cacheTTL;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("cacheTTL is required");
        })
      });
      context('When cacheTTL is not an integer', function() {
        it('Should return false', function() {
          settings.cacheTTL = "800";
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("cacheTTL must be a number");
        })
      });
    });

    context('Validate internal port, public port and health port', function() {
      context('When publicPort is equal to internal port', function() {
        it('Should return false', function() {
          settings.publicPort = 3002;
          settings.port = 3002;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("internal api port, public api port and health port should be different");
        })
      });
      context('When publicPort is equal to health port', function() {
        it('Should return false', function() {
          settings.publicPort = 3002;
          settings.healthPort = 3002;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("internal api port, public api port and health port should be different");
        })
      });
      context('When internal port is equal to health port', function() {
        it('Should return false', function() {
          settings.port = 3002;
          settings.healthPort = 3002;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("internal api port, public api port and health port should be different");
        })
      });
    });

    context('Validate skipSSLValidation', function() {
      context('When skipSSLValidation is null', function() {
        it('Should return false', function() {
          settings.skipSSLValidation = null;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal('skipSSLValidation is required');
        })
      });
      context('When skipSSLValidation is undefined', function() {
        it('Should return false', function() {
          delete settings.skipSSLValidation;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal('skipSSLValidation is required');
        })
      });
      context('When skipSSLValidation is not a boolean', function() {
        it('Should return false', function() {
          settings.skipSSLValidation = "12345";
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal('skipSSLValidation must be a boolean');
        })
      });
    });

    context('Validate infoFilePath', function() {
      context('When infoFilePath is null', function() {
        it('Should return false', function() {
          settings.infoFilePath = null;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("infoFilePath is required");
        })
      });
      context('When infoFilePath is undefined', function() {
        it('Should return false', function() {
          delete settings.infoFilePath;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("infoFilePath is required");
        })
      });
      context('When infoFilePath is not a string', function() {
        it('Should return false', function() {
          settings.infoFilePath = 12345;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("infoFilePath must be a string");
        })
      });
    });

    context('Validate cfApi', function() {
      context('When cfApi is null', function() {
        it('Should return false', function() {
          settings.cfApi = null;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("cfApi is required");
        })
      });
      context('When cfApi is undefined', function() {
        it('Should return false', function() {
          delete settings.cfApi;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("cfApi is required");
        })
      });
      context('When cfApi is not a string', function() {
        it('Should return false', function() {
          settings.cfApi = 12345;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("cfApi must be a string");
        })
      });
    });

    context('Validate cfClientId', function() {
      context('When cfClientId is null', function() {
        it('Should return false', function() {
          settings.cfClientId = null;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("cfClientId is required");
        })
      });
      context('When cfClientId is undefined', function() {
        it('Should return false', function() {
          delete settings.cfClientId;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("cfClientId is required");
        })
      });
      context('When cfClientId is not a string', function() {
        it('Should return false', function() {
          settings.cfClientId = 12345;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("cfClientId must be a string");
        })
      });
    });

    context('Validate cfClientSecret', function() {
      context('When cfClientSecret is null', function() {
        it('Should return false', function() {
          settings.cfClientSecret = null;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("cfClientSecret is required");
        })
      });
      context('When cfClientSecret is undefined', function() {
        it('Should return false', function() {
          delete settings.cfClientSecret;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("cfClientSecret is required");
        })
      });
      context('When cfClientSecret is not a string', function() {
        it('Should return false', function() {
          settings.cfClientSecret = 12345;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("cfClientSecret must be a string");
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

    context('Validate serviceOffering', function() {
      context('When serviceOffering is null', function() {
        it('Should return false', function() {
          settings.serviceOffering = null
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal('serviceOffering is required');
        })
      });
      context('When serviceOffering is undefined', function() {
        it('Should return false', function() {
          delete settings.serviceOffering;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal('serviceOffering is required');
        })
      });
      context('When serviceOffering is not an object', function() {
        it('Should return false', function() {
          settings.serviceOffering = "12345";
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal('serviceOffering must be an object');
        })
      });
      context('When serviceOffering.enabled is null', function() {
        it('Should return false', function() {
          settings.serviceOffering.enabled = null;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal('serviceOffering.enabled is required');
        })
      });
      context('When serviceOffering.enabled is undefined', function() {
        it('Should return false', function() {
          delete settings.serviceOffering.enabled;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal('serviceOffering.enabled is required');
        })
      });
      context('When serviceOffering.enabled is not boolean', function() {
        it('Should return false', function() {
          settings.serviceOffering.enabled = "12345"
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal('serviceOffering.enabled must be a boolean');
        })
      });
    });

    context('Validate scheduler.uri', function() {
      context('When scheduler.uri is null', function() {
        it('Should return false', function() {
          settings.scheduler.uri = null;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("scheduler.uri is required");
        })
      });
      context('When scheduler.uri is undefined', function() {
        it('Should return false', function() {
          delete settings.scheduler.uri;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("scheduler.uri is required");
        })
      });
      context('When scheduler.uri is not a string', function() {
        it('Should return false', function() {
          settings.scheduler.uri = 1234;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("scheduler.uri must be a string");
        })
      });
    });

    context('Validate scalingEngine uri', function() {
      context('When scalingEngine uri is null', function() {
        it('Should return false', function() {
          settings.scalingEngine.uri = null
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("scalingEngine.uri is required");
        })
      });
      context('When scalingEngine uri is undefined', function() {
        it('Should return false', function() {
          delete settings.scalingEngine.uri
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("scalingEngine.uri is required");
        })
      });
      context('When scalingEngine.uri is not a string', function() {
        it('Should return false', function() {
          settings.scalingEngine.uri = 1234;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("scalingEngine.uri must be a string");
        })
      });
    });

    context('Validate metricsCollector uri', function() {
      context('When metricsCollector uri is null', function() {
        it('Should return false', function() {
          settings.metricsCollector.uri = null
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("metricsCollector.uri is required");
        })
      });
      context('When metricsCollector uri is undefined', function() {
        it('Should return false', function() {
          delete settings.metricsCollector.uri
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("metricsCollector.uri is required");
        })
      });
      context('When metricsCollector.uri is not a string', function() {
        it('Should return false', function() {
          settings.metricsCollector.uri = 1234;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("metricsCollector.uri must be a string");
        })
      });
    });

    context('Validate eventGenerator uri', function() {
      context('When eventGenerator uri is null', function() {
        it('Should return false', function() {
          settings.eventGenerator.uri = null
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("eventGenerator.uri is required");
        })
      });
      context('When eventGenerator uri is undefined', function() {
        it('Should return false', function() {
          delete settings.eventGenerator.uri
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("eventGenerator.uri is required");
        })
      });
      context('When eventGenerator.uri is not a string', function() {
        it('Should return false', function() {
          settings.eventGenerator.uri = 1234;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("eventGenerator.uri must be a string");
        })
      });
    });

    context('Validate serviceBroker uri', function() {
      context('When serviceBroker uri is null', function() {
        it('Should return false', function() {
          settings.serviceOffering.serviceBroker.uri = null
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("serviceOffering.serviceBroker.uri is required");
        })
      });
      context('When serviceBroker uri is undefined', function() {
        it('Should return false', function() {
          delete settings.serviceOffering.serviceBroker.uri
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("serviceOffering.serviceBroker.uri is required");
        })
      });
      context('When serviceBroker.uri is not a string', function() {
        it('Should return false', function() {
          settings.serviceOffering.serviceBroker.uri = 1234;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("serviceOffering.serviceBroker.uri must be a string");
        })
      });
    });

    context('Validate httpClientTimeout', function() {
      context('When httpClientTimeout is null', function() {
        it('Should return false', function() {
          settings.httpClientTimeout = null;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("httpClientTimeout is required");
        })
      });
      context('When httpClientTimeout is undefined', function() {
        it('Should return false', function() {
          delete settings.httpClientTimeout;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("httpClientTimeout is required");
        })
      });
      context('When httpClientTimeout is not an integer', function() {
        it('Should return false', function() {
          settings.httpClientTimeout = "1000";
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("httpClientTimeout must be a number");
        })
      });
      context('When the httpClientTimeout is out of range', function() {
        it('Should return false', function() {
          settings.httpClientTimeout = -1;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("value of httpClientTimeout must be greater than 0");
        })
      });
    });

    context('Validate minBreachDurationSecs', function() {
      context('When minBreachDurationSecs is null', function() {
        it('Should return false', function() {
          settings.minBreachDurationSecs = null;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("minBreachDurationSecs is required");
        })
      });
      context('When minBreachDurationSecs is undefined', function() {
        it('Should return false', function() {
          delete settings.minBreachDurationSecs;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("minBreachDurationSecs is required");
        })
      });
      context('When minBreachDurationSecs is not an integer', function() {
        it('Should return false', function() {
          settings.minBreachDurationSecs = "1000";
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("minBreachDurationSecs must be a number");
        })
      });
      context('When the minBreachDurationSecs is out of range', function() {
        it('Should return false', function() {
          settings.minBreachDurationSecs = -1;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("value of minBreachDurationSecs must be greater than 0");
        })
      });
    });

    context('Validate minCoolDownSecs', function() {
      context('When minCoolDownSecs is null', function() {
        it('Should return false', function() {
          settings.minCoolDownSecs = null;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("minCoolDownSecs is required");
        })
      });
      context('When minCoolDownSecs is undefined', function() {
        it('Should return false', function() {
          delete settings.minCoolDownSecs;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("minCoolDownSecs is required");
        })
      });
      context('When minCoolDownSecs is not an integer', function() {
        it('Should return false', function() {
          settings.minCoolDownSecs = "1000";
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("minCoolDownSecs must be a number");
        })
      });
      context('When the minCoolDownSecs is out of range', function() {
        it('Should return false', function() {
          settings.minCoolDownSecs = -1;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("value of minCoolDownSecs must be greater than 0");
        })
      });
    });

  });


  context('Validate tls', function() {
    context('When tls is null', function() {
      it('Should return true', function() {
        settings.tls = null;
        expect(settings.validate().valid).to.equal(true);
      });
    });
    context('When tls is undefined', function() {
      it('Should return true', function() {
        delete settings.tls;
        expect(settings.validate().valid).to.equal(true);
      });
    });
    context('When tls is not an object', function() {
      it('Should return false', function() {
        settings.tls = "notobject";
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("tls must be an object");
      });
    });
  });

  context('Validate tls.keyFile', function() {
    context('When tls.keyFile is null', function() {
      it('Should return false', function() {
        settings.tls.keyFile = null;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("tls.keyFile is required");
      });
    });
    context('When tls.keyFile is undefined', function() {
      it('Should return false', function() {
        delete settings.tls.keyFile;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("tls.keyFile is required");
      });
    });
    context('When tls.keyFile is not a string', function() {
      it('Should return false', function() {
        settings.tls.keyFile = 1234;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("tls.keyFile must be a string");
      });
    });
  });

  context('Validate tls.certFile', function() {
    context('When tls.certFile is null', function() {
      it('Should return false', function() {
        settings.tls.certFile = null;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("tls.certFile is required");
      });
    });
    context('When tls.certFile is undefined', function() {
      it('Should return false', function() {
        delete settings.tls.certFile;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("tls.certFile is required");
      });
    });
    context('When tls.certFile is not a string', function() {
      it('Should return false', function() {
        settings.tls.certFile = 1234;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("tls.certFile must be a string");
      });
    });
  });

  context('Validate tls.caCertFile', function() {
    context('When tls.caCertFile is null', function() {
      it('Should return false', function() {
        settings.tls.caCertFile = null;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("tls.caCertFile is required");
      });
    });
    context('When tls.caCertFile is undefined', function() {
      it('Should return false', function() {
        delete settings.tls.caCertFile;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("tls.caCertFile is required");
      });
    });
    context('When tls.caCertFile is not a string', function() {
      it('Should return false', function() {
        settings.tls.caCertFile = 1234;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("tls.caCertFile must be a string");
      });
    });
  });

  context('Validate publicTls', function() {
    context('When publicTls is null', function() {
      it('Should return true', function() {
        settings.publicTls = null;
        expect(settings.validate().valid).to.equal(true);
      });
    });
    context('When publicTls is undefined', function() {
      it('Should return true', function() {
        delete settings.publicTls;
        expect(settings.validate().valid).to.equal(true);
      });
    });
    context('When publicTls is not an object', function() {
      it('Should return false', function() {
        settings.publicTls = "notobject";
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("publicTls must be an object");
      });
    });
  });

  context('Validate publicTls.keyFile', function() {
    context('When publicTls.keyFile is null', function() {
      it('Should return false', function() {
        settings.publicTls.keyFile = null;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("publicTls.keyFile is required");
      });
    });
    context('When publicTls.keyFile is undefined', function() {
      it('Should return false', function() {
        delete settings.publicTls.keyFile;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("publicTls.keyFile is required");
      });
    });
    context('When publicTls.keyFile is not a string', function() {
      it('Should return false', function() {
        settings.publicTls.keyFile = 1234;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("publicTls.keyFile must be a string");
      });
    });
  });

  context('Validate publicTls.certFile', function() {
    context('When publicTls.certFile is null', function() {
      it('Should return false', function() {
        settings.publicTls.certFile = null;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("publicTls.certFile is required");
      });
    });
    context('When publicTls.certFile is undefined', function() {
      it('Should return false', function() {
        delete settings.publicTls.certFile;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("publicTls.certFile is required");
      });
    });
    context('When publicTls.certFile is not a string', function() {
      it('Should return false', function() {
        settings.publicTls.certFile = 1234;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("publicTls.certFile must be a string");
      });
    });
  });

  context('Validate publicTls.caCertFile', function() {
    context('When publicTls.caCertFile is null', function() {
      it('Should return false', function() {
        settings.publicTls.caCertFile = null;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("publicTls.caCertFile is required");
      });
    });
    context('When publicTls.caCertFile is undefined', function() {
      it('Should return false', function() {
        delete settings.publicTls.caCertFile;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("publicTls.caCertFile is required");
      });
    });
    context('When publicTls.caCertFile is not a string', function() {
      it('Should return false', function() {
        settings.publicTls.caCertFile = 1234;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("publicTls.caCertFile must be a string");
      });
    });
  });

  context('Validate scheduler client tls.keyFile', function() {
    context('When scheduler client tls.keyFile is null', function() {
      it('Should return false', function() {
        settings.scheduler.tls.keyFile = null;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("scheduler.tls.keyFile is required");
      });
    });
    context('When scheduler client tls.keyFile is undefined', function() {
      it('Should return false', function() {
        delete settings.scheduler.tls.keyFile;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("scheduler.tls.keyFile is required");
      });
    });
    context('When scheduler client tls.keyFile is not a string', function() {
      it('Should return false', function() {
        settings.scheduler.tls.keyFile = 1234;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("scheduler.tls.keyFile must be a string");
      });
    });
  });

  context('Validate scheduler client tls.certFile', function() {
    context('When scheduler client tls.certFile is null', function() {
      it('Should return false', function() {
        settings.scheduler.tls.certFile = null;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("scheduler.tls.certFile is required");
      });
    });
    context('When scheduler client tls.certFile is undefined', function() {
      it('Should return false', function() {
        delete settings.scheduler.tls.certFile;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("scheduler.tls.certFile is required");
      });
    });
    context('When scheduler client tls.certFile is not a string', function() {
      it('Should return false', function() {
        settings.scheduler.tls.certFile = 1234;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("scheduler.tls.certFile must be a string");
      });
    });
  });

  context('Validate scheduler client tls.caCertFile', function() {
    context('When scheduler client tls.caCertFile is null', function() {
      it('Should return false', function() {
        settings.scheduler.tls.caCertFile = null;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("scheduler.tls.caCertFile is required");
      });
    });
    context('When scheduler client tls.caCertFile is undefined', function() {
      it('Should return false', function() {
        delete settings.scheduler.tls.caCertFile;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("scheduler.tls.caCertFile is required");
      });
    });
    context('When scheduler client tls.caCertFile is not a string', function() {
      it('Should return false', function() {
        settings.scheduler.tls.caCertFile = 1234;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("scheduler.tls.caCertFile must be a string");
      });
    });
  });

  context('Validate scheduler client tls', function() {
    context('When scheduler client tls is null', function() {
      it('Should return true', function() {
        settings.scheduler.tls = null;
        expect(settings.validate().valid).to.equal(true);
      });
    });
    context('When scheduler client tls  is undefined', function() {
      it('Should return true', function() {
        delete settings.scheduler.tls;
        expect(settings.validate().valid).to.equal(true);
      });
    });
    context('When scheduler client tls is not an object', function() {
      it('Should return false', function() {
        settings.scheduler.tls = "notobject";
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("scheduler.tls must be an object");
      });
    });
  });

  context('Validate scalingEngine client tls.keyFile', function() {
    context('When scalingEngine client tls.keyFile is null', function() {
      it('Should return false', function() {
        settings.scalingEngine.tls.keyFile = null;
        expect(settings.validate().valid).to.equal(false)
        expect(settings.validate().message).to.equal("scalingEngine.tls.keyFile is required");
      });
    });
    context('When scalingEngine client tls.keyFile is undefined', function() {
      it('Should return false', function() {
        delete settings.scalingEngine.tls.keyFile;
        expect(settings.validate().valid).to.equal(false)
        expect(settings.validate().message).to.equal("scalingEngine.tls.keyFile is required");
      });
    });
    context('When scalingEngine client tls.keyFile is not a string', function() {
      it('Should return false', function() {
        settings.scalingEngine.tls.keyFile = 1234;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("scalingEngine.tls.keyFile must be a string");
      });
    });
  });

  context('Validate scalingEngine client tls.certFile', function() {
    context('When scalingEngine client tls.certFile is null', function() {
      it('Should return false', function() {
        settings.scalingEngine.tls.certFile = null;
        expect(settings.validate().valid).to.equal(false)
        expect(settings.validate().message).to.equal("scalingEngine.tls.certFile is required");
      });
    });
    context('When scalingEngine client tls.certFile is undefined', function() {
      it('Should return false', function() {
        delete settings.scalingEngine.tls.certFile;
        expect(settings.validate().valid).to.equal(false)
        expect(settings.validate().message).to.equal("scalingEngine.tls.certFile is required");
      });
    });
    context('When scalingEngine client tls.certFile is not a string', function() {
      it('Should return false', function() {
        settings.scalingEngine.tls.certFile = 1234;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("scalingEngine.tls.certFile must be a string");
      });
    });
  });

  context('Validate scalingEngine client tls.caCertFile', function() {
    context('When scalingEngine client tls.caCertFile is null', function() {
      it('Should return false', function() {
        settings.scalingEngine.tls.caCertFile = null;
        expect(settings.validate().valid).to.equal(false)
        expect(settings.validate().message).to.equal("scalingEngine.tls.caCertFile is required");
      });
    });
    context('When scalingEngine client tls.caCertFile is undefined', function() {
      it('Should return false', function() {
        delete settings.scalingEngine.tls.caCertFile;
        expect(settings.validate().valid).to.equal(false)
        expect(settings.validate().message).to.equal("scalingEngine.tls.caCertFile is required");
      });
    });
    context('When scalingEngine client tls.caCertFile is not a string', function() {
      it('Should return false', function() {
        settings.scalingEngine.tls.caCertFile = 1234;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("scalingEngine.tls.caCertFile must be a string");
      });
    });
  });

  context('Validate scalingEngine client tls', function() {
    context('When scalingEngine client tls is null', function() {
      it('Should return true', function() {
        settings.scalingEngine.tls = null;
        expect(settings.validate().valid).to.equal(true);
      });
    });
    context('When scalingEngine client tls  is undefined', function() {
      it('Should return true', function() {
        delete settings.scalingEngine.tls;
        expect(settings.validate().valid).to.equal(true);
      });
    });
    context('When scalingEngine client tls is not an object', function() {
      it('Should return false', function() {
        settings.scalingEngine.tls = "notobject";
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("scalingEngine.tls must be an object");
      });
    });
  });

  context('Validate metricsCollector client tls.keyFile', function() {
    context('When metricsCollector client tls.keyFile is null', function() {
      it('Should return false', function() {
        settings.metricsCollector.tls.keyFile = null;
        expect(settings.validate().valid).to.equal(false)
        expect(settings.validate().message).to.equal("metricsCollector.tls.keyFile is required");
      });
    });
    context('When metricsCollector client tls.keyFile is undefined', function() {
      it('Should return false', function() {
        delete settings.metricsCollector.tls.keyFile;
        expect(settings.validate().valid).to.equal(false)
        expect(settings.validate().message).to.equal("metricsCollector.tls.keyFile is required");
      });
    });
    context('When metricsCollector client tls.keyFile is not a string', function() {
      it('Should return false', function() {
        settings.metricsCollector.tls.keyFile = 1234;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("metricsCollector.tls.keyFile must be a string");
      });
    });
  });

  context('Validate metricsCollector client tls.certFile', function() {
    context('When metricsCollector client tls.certFile is null', function() {
      it('Should return false', function() {
        settings.metricsCollector.tls.certFile = null;
        expect(settings.validate().valid).to.equal(false)
        expect(settings.validate().message).to.equal("metricsCollector.tls.certFile is required");
      });
    });
    context('When metricsCollector client tls.certFile is undefined', function() {
      it('Should return false', function() {
        delete settings.metricsCollector.tls.certFile;
        expect(settings.validate().valid).to.equal(false)
        expect(settings.validate().message).to.equal("metricsCollector.tls.certFile is required");
      });
    });
    context('When metricsCollector client tls.certFile is not a string', function() {
      it('Should return false', function() {
        settings.metricsCollector.tls.certFile = 1234;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("metricsCollector.tls.certFile must be a string");
      });
    });
  });

  context('Validate metricsCollector client tls.caCertFile', function() {
    context('When metricsCollector client tls.caCertFile is null', function() {
      it('Should return false', function() {
        settings.metricsCollector.tls.caCertFile = null;
        expect(settings.validate().valid).to.equal(false)
        expect(settings.validate().message).to.equal("metricsCollector.tls.caCertFile is required");
      });
    });
    context('When metricsCollector client tls.caCertFile is undefined', function() {
      it('Should return false', function() {
        delete settings.metricsCollector.tls.caCertFile;
        expect(settings.validate().valid).to.equal(false)
        expect(settings.validate().message).to.equal("metricsCollector.tls.caCertFile is required");
      });
    });
    context('When metricsCollector client tls.caCertFile is not a string', function() {
      it('Should return false', function() {
        settings.metricsCollector.tls.caCertFile = 1234;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("metricsCollector.tls.caCertFile must be a string");
      });
    });
  });

  context('Validate metricsCollector client tls', function() {
    context('When metricsCollector client tls is null', function() {
      it('Should return true', function() {
        settings.metricsCollector.tls = null;
        expect(settings.validate().valid).to.equal(true);
      });
    });
    context('When metricsCollector client tls  is undefined', function() {
      it('Should return true', function() {
        delete settings.metricsCollector.tls;
        expect(settings.validate().valid).to.equal(true);
      });
    });
    context('When metricsCollector client tls is not an object', function() {
      it('Should return false', function() {
        settings.metricsCollector.tls = "notobject";
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("metricsCollector.tls must be an object");
      });
    });
  });

  context('Validate eventGenerator client tls.keyFile', function() {
    context('When eventGenerator client tls.keyFile is null', function() {
      it('Should return false', function() {
        settings.eventGenerator.tls.keyFile = null;
        expect(settings.validate().valid).to.equal(false)
        expect(settings.validate().message).to.equal("eventGenerator.tls.keyFile is required");
      });
    });
    context('When eventGenerator client tls.keyFile is undefined', function() {
      it('Should return false', function() {
        delete settings.eventGenerator.tls.keyFile;
        expect(settings.validate().valid).to.equal(false)
        expect(settings.validate().message).to.equal("eventGenerator.tls.keyFile is required");
      });
    });
    context('When eventGenerator client tls.keyFile is not a string', function() {
      it('Should return false', function() {
        settings.eventGenerator.tls.keyFile = 1234;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("eventGenerator.tls.keyFile must be a string");
      });
    });
  });

  context('Validate eventGenerator client tls.certFile', function() {
    context('When eventGenerator client tls.certFile is null', function() {
      it('Should return false', function() {
        settings.eventGenerator.tls.certFile = null;
        expect(settings.validate().valid).to.equal(false)
        expect(settings.validate().message).to.equal("eventGenerator.tls.certFile is required");
      });
    });
    context('When eventGenerator client tls.certFile is undefined', function() {
      it('Should return false', function() {
        delete settings.eventGenerator.tls.certFile;
        expect(settings.validate().valid).to.equal(false)
        expect(settings.validate().message).to.equal("eventGenerator.tls.certFile is required");
      });
    });
    context('When eventGenerator client tls.certFile is not a string', function() {
      it('Should return false', function() {
        settings.eventGenerator.tls.certFile = 1234;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("eventGenerator.tls.certFile must be a string");
      });
    });
  });

  context('Validate eventGenerator client tls.caCertFile', function() {
    context('When eventGenerator client tls.caCertFile is null', function() {
      it('Should return false', function() {
        settings.eventGenerator.tls.caCertFile = null;
        expect(settings.validate().valid).to.equal(false)
        expect(settings.validate().message).to.equal("eventGenerator.tls.caCertFile is required");
      });
    });
    context('When eventGenerator client tls.caCertFile is undefined', function() {
      it('Should return false', function() {
        delete settings.eventGenerator.tls.caCertFile;
        expect(settings.validate().valid).to.equal(false)
        expect(settings.validate().message).to.equal("eventGenerator.tls.caCertFile is required");
      });
    });
    context('When eventGenerator client tls.caCertFile is not a string', function() {
      it('Should return false', function() {
        settings.eventGenerator.tls.caCertFile = 1234;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("eventGenerator.tls.caCertFile must be a string");
      });
    });
  });

  context('Validate eventGenerator client tls', function() {
    context('When eventGenerator client tls is null', function() {
      it('Should return true', function() {
        settings.eventGenerator.tls = null;
        expect(settings.validate().valid).to.equal(true);
      });
    });
    context('When eventGenerator client tls  is undefined', function() {
      it('Should return true', function() {
        delete settings.eventGenerator.tls;
        expect(settings.validate().valid).to.equal(true);
      });
    });
    context('When eventGenerator client tls is not an object', function() {
      it('Should return false', function() {
        settings.eventGenerator.tls = "notobject";
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("eventGenerator.tls must be an object");
      });
    });
  });

  context('Validate serviceBroker client tls.keyFile', function() {
    context('When serviceBroker client tls.keyFile is null', function() {
      it('Should return false', function() {
        settings.serviceOffering.serviceBroker.tls.keyFile = null;
        expect(settings.validate().valid).to.equal(false)
        expect(settings.validate().message).to.equal("serviceOffering.serviceBroker.tls.keyFile is required");
      });
    });
    context('When serviceBroker client tls.keyFile is undefined', function() {
      it('Should return false', function() {
        delete settings.serviceOffering.serviceBroker.tls.keyFile;
        expect(settings.validate().valid).to.equal(false)
        expect(settings.validate().message).to.equal("serviceOffering.serviceBroker.tls.keyFile is required");
      });
    });
    context('When serviceBroker client tls.keyFile is not a string', function() {
      it('Should return false', function() {
        settings.serviceOffering.serviceBroker.tls.keyFile = 1234;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("serviceOffering.serviceBroker.tls.keyFile must be a string");
      });
    });
  });

  context('Validate serviceBroker client tls.certFile', function() {
    context('When serviceBroker client tls.certFile is null', function() {
      it('Should return false', function() {
        settings.serviceOffering.serviceBroker.tls.certFile = null;
        expect(settings.validate().valid).to.equal(false)
        expect(settings.validate().message).to.equal("serviceOffering.serviceBroker.tls.certFile is required");
      });
    });
    context('When serviceBroker client tls.certFile is undefined', function() {
      it('Should return false', function() {
        delete settings.serviceOffering.serviceBroker.tls.certFile;
        expect(settings.validate().valid).to.equal(false)
        expect(settings.validate().message).to.equal("serviceOffering.serviceBroker.tls.certFile is required");
      });
    });
    context('When serviceBroker client tls.certFile is not a string', function() {
      it('Should return false', function() {
        settings.serviceOffering.serviceBroker.tls.certFile = 1234;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("serviceOffering.serviceBroker.tls.certFile must be a string");
      });
    });
  });

  context('Validate serviceBroker client tls.caCertFile', function() {
    context('When serviceBroker client tls.caCertFile is null', function() {
      it('Should return false', function() {
        settings.serviceOffering.serviceBroker.tls.caCertFile = null;
        expect(settings.validate().valid).to.equal(false)
        expect(settings.validate().message).to.equal("serviceOffering.serviceBroker.tls.caCertFile is required");
      });
    });
    context('When serviceBroker client tls.caCertFile is undefined', function() {
      it('Should return false', function() {
        delete settings.serviceOffering.serviceBroker.tls.caCertFile;
        expect(settings.validate().valid).to.equal(false)
        expect(settings.validate().message).to.equal("serviceOffering.serviceBroker.tls.caCertFile is required");
      });
    });
    context('When serviceBroker client tls.caCertFile is not a string', function() {
      it('Should return false', function() {
        settings.serviceOffering.serviceBroker.tls.caCertFile = 1234;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("serviceOffering.serviceBroker.tls.caCertFile must be a string");
      });
    });
  });

  context('Validate serviceBroker client tls', function() {
    context('When serviceBroker client tls is null', function() {
      it('Should return true', function() {
        settings.serviceOffering.serviceBroker.tls = null;
        expect(settings.validate().valid).to.equal(true);
      });
    });
    context('When serviceBroker client tls  is undefined', function() {
      it('Should return true', function() {
        delete settings.serviceOffering.serviceBroker.tls;
        expect(settings.validate().valid).to.equal(true);
      });
    });
    context('When serviceBroker client tls is not an object', function() {
      it('Should return false', function() {
        settings.serviceOffering.serviceBroker.tls = "notobject";
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("serviceOffering.serviceBroker.tls must be an object");
      });
    });
  });

});