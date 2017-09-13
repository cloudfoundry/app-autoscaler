'use strict';

var path = require('path');
var expect = require('chai').expect;
var configSetting = require(path.join(__dirname, '../../../lib/config/setting.js'));
var defaultConfig,defaultConfigTemplate;
var settingTmp = {};
var settings;

describe('config setting Test Suite', function() {
  beforeEach(function() {
    defaultConfigTemplate = {
      "port": 8080,
      "publicPort": 8081,
      "cfApi": "api.bosh-lite.com",
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
      "tls": {
        "keyFile": "keyFilePath",
        "certFile": "certFilePath",
        "caCertFile": "caCertFilePath"
      },
      "publicTls": {
        "keyFile": "keyFilePath",
        "certFile": "certFilePath",
        "caCertFile": "caCertFilePath"
      }
    }
    //setting.js will modifty defaultConfigTemplate, here clone defaultConfigTemplate to defaultConfig first
    defaultConfig = JSON.parse(JSON.stringify(defaultConfigTemplate))
    settings = configSetting(defaultConfigTemplate);
  });

  it('Should contain the default configuration', function() {
    expect(settings.port).to.equal(defaultConfig.port);

    expect(settings.publicPort).to.equal(defaultConfig.publicPort);

    expect(settings.cfApi).to.equal("https://" + defaultConfig.cfApi.toLowerCase());

    expect(settings.db.maxConnections).to.equal(defaultConfig.db.maxConnections);
    expect(settings.db.minConnections).to.equal(defaultConfig.db.minConnections);
    expect(settings.db.idleTimeout).to.equal(defaultConfig.db.idleTimeout);
    expect(settings.db.uri).to.equal(defaultConfig.db.uri);
    expect(settings.db.server).to.equal('postgres://postgres@server:80');
    expect(settings.db.name).to.equal('dbname');

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

    context('Validate internal port and public port',function(){
      context('When publicPort is equal to internal port', function() {
        it('Should return false', function() {
          settings.publicPort = 3002;
          settings.port = 3002;
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("internal api port and public api port should be different");
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
      it("Should add http if no protocol",function(){
        var apiSetting = configSetting({ cfApi: defaultConfig.cfApi, db: { uri: defaultConfig.db.uri}, scheduler: { uri: defaultConfig.scheduler.uri, tls: defaultConfig.scheduler.tls }, scalingEngine: { uri: defaultConfig.scalingEngine.uri, tls: defaultConfig.scalingEngine.tls },metricsCollector: { uri: defaultConfig.metricsCollector.uri, tls: defaultConfig.metricsCollector.tls } });
        expect(apiSetting.cfApi).to.equal("https://" + defaultConfig.cfApi);
      });
      it("Should filter the last slash",function(){
        var apiSetting = configSetting({ cfApi: defaultConfig.cfApi + "/", db: { uri: defaultConfig.db.uri}, scheduler: { uri: defaultConfig.scheduler.uri, tls: defaultConfig.scheduler.tls }, scalingEngine: { uri: defaultConfig.scalingEngine.uri, tls: defaultConfig.scalingEngine.tls },metricsCollector: { uri: defaultConfig.metricsCollector.uri, tls: defaultConfig.metricsCollector.tls } });
        expect(apiSetting.cfApi).to.equal("https://" + defaultConfig.cfApi);
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


    context('db.uri', function() {
      it('Should filter the last slash', function() {
        var dbSetting = configSetting({ db: { uri: defaultConfig.db.uri + '/' }, scheduler: { uri: defaultConfig.scheduler.uri, tls: defaultConfig.scheduler.tls }, scalingEngine: { uri: defaultConfig.scalingEngine.uri, tls: defaultConfig.scalingEngine.tls },metricsCollector: { uri: defaultConfig.metricsCollector.uri, tls: defaultConfig.metricsCollector.tls } }).db;
        expect(dbSetting.uri).to.equal(defaultConfig.db.uri);
        expect(dbSetting.server).to.equal("postgres://postgres@server:80");
        expect(dbSetting.name).to.equal("dbname");
      });
    });

    context('scheduler.uri', function() {

      it('Should add http if no protocol', function() {
        var apiSetting = configSetting({ scheduler: { uri: defaultConfig.scheduler.uri }, scalingEngine: { uri: defaultConfig.scalingEngine.uri, tls: defaultConfig.scalingEngine.tls },metricsCollector: { uri: defaultConfig.metricsCollector.uri, tls: defaultConfig.metricsCollector.tls } }).scheduler;
        expect(apiSetting.uri).to.equal("https://" + defaultConfig.scheduler.uri.toLowerCase());
      });

      it('Should filter the last slash', function() {
        var apiSetting = configSetting({ scheduler: { uri: defaultConfig.scheduler.uri + '/' }, scalingEngine: { uri: defaultConfig.scalingEngine.uri, tls: defaultConfig.scalingEngine.tls },metricsCollector: { uri: defaultConfig.metricsCollector.uri, tls: defaultConfig.metricsCollector.tls } }).scheduler;
        expect(apiSetting.uri).to.equal("https://" + defaultConfig.scheduler.uri.toLowerCase());
      });

      context('When the scheduler.uri is upper case', function() {
        it('Should be lowercased', function() {
          var apiSetting = configSetting({ scheduler: { uri: defaultConfig.scheduler.uri.toUpperCase() }, scalingEngine: { uri: defaultConfig.scalingEngine.uri, tls: defaultConfig.scalingEngine.tls },metricsCollector: { uri: defaultConfig.metricsCollector.uri, tls: defaultConfig.metricsCollector.tls } }).scheduler;
          expect(apiSetting.uri).to.equal("https://" + defaultConfig.scheduler.uri.toLowerCase());
        });
      });
    });
    context('scalingEngine uri', function() {

      it('Should add http if no protocol', function() {
        var apiSetting = configSetting({ scalingEngine: { uri: defaultConfig.scalingEngine.uri }, scheduler: { uri: defaultConfig.scheduler.uri, tls: defaultConfig.scheduler.tls },metricsCollector: { uri: defaultConfig.metricsCollector.uri, tls: defaultConfig.metricsCollector.tls } }).scalingEngine;
        expect(apiSetting.uri).to.equal("https://" + defaultConfig.scalingEngine.uri.toLowerCase());
      });

      it('Should filter the last slash', function() {
        var apiSetting = configSetting({ scalingEngine: { uri: defaultConfig.scalingEngine.uri + '/' }, scheduler: { uri: defaultConfig.scheduler.uri, tls: defaultConfig.scheduler.tls },metricsCollector: { uri: defaultConfig.metricsCollector.uri, tls: defaultConfig.metricsCollector.tls } }).scalingEngine;
        expect(apiSetting.uri).to.equal("https://" + defaultConfig.scalingEngine.uri.toLowerCase());
      });

      context('When the scalingEngine uri is upper case', function() {
        it('Should be lowercased', function() {
          var apiSetting = configSetting({ scalingEngine: { uri: defaultConfig.scalingEngine.uri.toUpperCase() }, scheduler: { uri: defaultConfig.scheduler.uri, tls: defaultConfig.scheduler.tls },metricsCollector: { uri: defaultConfig.metricsCollector.uri, tls: defaultConfig.metricsCollector.tls } }).scalingEngine;
          expect(apiSetting.uri).to.equal("https://" + defaultConfig.scalingEngine.uri.toLowerCase());
        });
      });
    });
    context('metricsCollector uri', function() {

      it('Should add http if no protocol', function() {
        var apiSetting = configSetting({ metricsCollector: { uri: defaultConfig.metricsCollector.uri }, scheduler: { uri: defaultConfig.scheduler.uri, tls: defaultConfig.scheduler.tls }, scalingEngine: { uri: defaultConfig.scalingEngine.uri, tls: defaultConfig.scalingEngine.tls } }).metricsCollector;
        expect(apiSetting.uri).to.equal("https://" + defaultConfig.metricsCollector.uri.toLowerCase());
      });

      it('Should filter the last slash', function() {
        var apiSetting = configSetting({ metricsCollector: { uri: defaultConfig.metricsCollector.uri + '/' }, scheduler: { uri: defaultConfig.scheduler.uri, tls: defaultConfig.scheduler.tls }, scalingEngine: { uri: defaultConfig.scalingEngine.uri, tls: defaultConfig.scalingEngine.tls } }).metricsCollector;
        expect(apiSetting.uri).to.equal("https://" + defaultConfig.metricsCollector.uri.toLowerCase());
      });

      context('When the metricsCollector uri is upper case', function() {
        it('Should be lowercased', function() {
          var apiSetting = configSetting({ metricsCollector: { uri: defaultConfig.metricsCollector.uri.toUpperCase() }, scheduler: { uri: defaultConfig.scheduler.uri, tls: defaultConfig.scheduler.tls },scalingEngine: { uri: defaultConfig.scalingEngine.uri, tls: defaultConfig.scalingEngine.tls } }).metricsCollector;
          expect(apiSetting.uri).to.equal("https://" + defaultConfig.metricsCollector.uri.toLowerCase());
        });
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

});
