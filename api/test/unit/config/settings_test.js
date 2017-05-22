'use strict';

var path = require('path');
var expect = require('chai').expect;
var configSetting = require(path.join(__dirname, '../../../lib/config/setting.js'));
var defaultConfig;
var settingTmp = {};
var settings;

describe('config setting Test Suite', function() {
  beforeEach(function() {
    defaultConfig = {
      "port": 8080,
      "db": {
        "maxConnections": 10,
        "minConnections": 0,
        "idleTimeout": 1000,
        "uri": "postgres://postgres@server:80/dbname",
      },
      "scheduler": {
        "uri": "http://scheduleruri",
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
      }
    }
    settings = configSetting(defaultConfig);
  });

  it('Should contain the default configuration', function() {
    expect(settings.port).to.equal(defaultConfig.port);
    expect(settings.db.maxConnections).to.equal(defaultConfig.db.maxConnections);
    expect(settings.db.minConnections).to.equal(defaultConfig.db.minConnections);
    expect(settings.db.idleTimeout).to.equal(defaultConfig.db.idleTimeout);
    expect(settings.db.uri).to.equal(defaultConfig.db.uri);
    expect(settings.db.server).to.equal('postgres://postgres@server:80');
    expect(settings.db.name).to.equal('dbname');
    expect(settings.tls.keyFile).to.equal(defaultConfig.tls.keyFile);
    expect(settings.tls.certFile).to.equal(defaultConfig.tls.certFile);
    expect(settings.tls.caCertFile).to.equal(defaultConfig.tls.caCertFile);
    expect(settings.scheduler.uri).to.equal(defaultConfig.scheduler.uri);
    expect(settings.scheduler.tls.keyFile).to.equal(defaultConfig.scheduler.tls.keyFile);
    expect(settings.scheduler.tls.caCertFile).to.equal(defaultConfig.scheduler.tls.caCertFile);
    expect(settings.scheduler.tls.certFile).to.equal(defaultConfig.scheduler.tls.certFile);

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
      context('When db.minConnections is null', function() {
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


    context('Validate dbUri', function() {
      context('When dbUri is null', function() {
        it('Should return false', function() {
          settings.db.uri = null
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("dbUri is required");
        })
      });
      context('When dbUri is undefined', function() {
        it('Should return false', function() {
          delete settings.db.uri
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("dbUri is required");
        })
      });
    });

    
  context('dbUri', function() {
    it('Should filter the last slash', function() {
      var dbSetting = configSetting({ db: { uri: defaultConfig.db.uri + '/' }, scheduler: { uri: defaultConfig.scheduler.uri, tls: defaultConfig.scheduler.tls } }).db;
      expect(dbSetting.uri).to.equal(defaultConfig.db.uri);
      expect(dbSetting.server).to.equal("postgres://postgres@server:80");
      expect(dbSetting.name).to.equal("dbname");
    });

    context('When the dbUri is mixed case', function() {
      it('Should be lowercased', function() {
        var dbSetting = configSetting({ db: { uri:defaultConfig.db.uri.toUpperCase() }, scheduler: { uri: defaultConfig.scheduler.uri, tls: defaultConfig.scheduler.tls } }).db;
        expect(dbSetting.uri).to.equal(defaultConfig.db.uri);
      });
    });
  });

  context('scheduler uri', function() {
    it('Should filter the last slash', function() {
      var apiSetting = configSetting({ scheduler : { uri: defaultConfig.scheduler.uri + '/' }}).scheduler;
      expect(apiSetting.uri).to.equal(defaultConfig.scheduler.uri);
    });

    context('When the scheduler uri is upper case', function() {
      it('Should be lowercased', function() {
        var apiSetting = configSetting({ scheduler : { uri: defaultConfig.scheduler.uri.toUpperCase() }}).scheduler;
        expect(apiSetting.uri).to.equal(defaultConfig.scheduler.uri);
      });
    });
  });

 context('Validate scheduler uri', function() {
      context('When scheduler uri is null', function() {
        it('Should return false', function() {
          settings.scheduler.uri = null
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("scheduler uri is required");
        })
      });
      context('When scheduler uri is undefined', function() {
        it('Should return false', function() {
          delete settings.scheduler.uri
          expect(settings.validate().valid).to.equal(false);
          expect(settings.validate().message).to.equal("scheduler uri is required");
        })
      });
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
  
 context('Validate scheduler client tls.keyFile', function(){
    context('When scheduler client tls.keyFile is null', function(){
      it('Should return false',function(){
        settings.scheduler.tls.keyFile = null;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("scheduler.tls.keyFile is required");
      });
    });
    context('When scheduler client tls.keyFile is undefined', function(){
      it('Should return false',function(){
        delete settings.scheduler.tls.keyFile;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("scheduler.tls.keyFile is required");
      });
    });
  });

  context('Validate scheduler client tls.certFile', function(){
    context('When scheduler client tls.certFile is null', function(){
      it('Should return false',function(){
        settings.scheduler.tls.certFile = null;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("scheduler.tls.certFile is required");
      });
    });
    context('When scheduler client tls.certFile is undefined', function(){
      it('Should return false',function(){
        delete settings.scheduler.tls.certFile;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("scheduler.tls.certFile is required");
      });
    });
  });

  context('Validate scheduler client tls.caCertFile', function(){
    context('When scheduler client tls.caCertFile is null', function(){
      it('Should return false',function(){
        settings.scheduler.tls.caCertFile = null;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("scheduler.tls.caCertFile is required");
      });
    });
    context('When scheduler client tls.caCertFile is undefined', function(){
      it('Should return false',function(){
        delete settings.scheduler.tls.caCertFile;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("scheduler.tls.caCertFile is required");
      });
    });
  });

  context('Validate scheduler client tls', function(){
    context('When scheduler client tls is null', function(){
      it('Should return false',function(){
        settings.scheduler.tls = null;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("scheduler tls is required");
      });
    });
    context('When scheduler client tls  is undefined', function(){
      it('Should return false',function(){
        delete settings.scheduler.tls;
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("scheduler.tls should be an object");
      });
    });
    context('When scheduler client tls is not an object', function(){
      it('Should return false',function(){
        settings.scheduler.tls = "notobject";
        expect(settings.validate().valid).to.equal(false);
        expect(settings.validate().message).to.equal("scheduler.tls should be an object");
      });
    });
  });

});
