'use strict';

var path = require('path');
var expect = require('chai').expect;
var configSetting = require(path.join(__dirname, '../../../lib/config/settings.js'));
var defaultConfig = {
  "port": 80,
  "dbUri": "postgres://postgres@server:80/dbname",
  "dbMaxConnectionCount": 10,
  "dbMinConnectionCount": 0,
  "dbMaxIdleTime": 1000,
  "schedulerUri": "http://scheduleruri"
}

var settingTmp = {};
var settings;

describe('config setting Test Suite', function() {
  beforeEach(function() {
    settings = configSetting(defaultConfig)
  });

  it('Should contain the default configuration', function() {
      expect(settings.port).to.equal(defaultConfig.port);
      expect(settings.dbMaxConnectionCount).to.equal(defaultConfig.dbMaxConnectionCount);
      expect(settings.dbMinConnectionCount).to.equal(defaultConfig.dbMinConnectionCount);
      expect(settings.dbMaxIdleTime).to.equal(defaultConfig.dbMaxIdleTime);
      expect(settings.apiServerUri).to.equal(defaultConfig.apiServerUri);
      expect(settings.db.uri).to.equal(defaultConfig.dbUri);
      expect(settings.db.server).to.equal('postgres://postgres@server:80');
      expect(settings.db.name).to.equal('dbname');
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

    context('Validate dbMaxConnectionCount', function() {
      context('When dbMaxConnectionCount is null', function() {
        it('Should return false', function() {
          settings.dbMaxConnectionCount = null;
          expect(settings.validate().valid).to.equal(false);
        })
      });
      context('When dbMaxConnectionCount is undefined', function() {
        it('Should return false', function() {
          delete settings.dbMaxConnectionCount
          expect(settings.validate().valid).to.equal(false);
        })
      });
      context('When dbMaxConnectionCount is not an integer', function() {
        it('Should return false', function() {
          settings.dbMaxConnectionCount = "10";
          expect(settings.validate().valid).to.equal(false);
        })
      });
      context('When the dbMaxConnectionCount is out of range', function() {
        it('Should return false', function() {
          settings.dbMaxConnectionCount = -10;
          expect(settings.validate().valid).to.equal(false);
        })
      });
    });

    context('Validate dbMinConnectionCount', function() {
      context('When dbMinConnectionCount is null', function() {
        it('Should return false', function() {
          settings.dbMinConnectionCount = null;
          expect(settings.validate().valid).to.equal(false);
        })
      });
      context('When dbMinConnectionCount is undefined', function() {
        it('Should return false', function() {
          delete settings.dbMinConnectionCount
          expect(settings.validate().valid).to.equal(false);
        })
      });
      context('When dbMinConnectionCount is not an integer', function() {
        it('Should return false', function() {
          settings.dbMinConnectionCount = "10";
          expect(settings.validate().valid).to.equal(false);
        })
      });
      context('When the dbMinConnectionCount is out of range', function() {
        it('Should return false', function() {
          settings.dbMinConnectionCount = -10;
          expect(settings.validate().valid).to.equal(false);
        })
      });
    });

    context('Validate dbMaxIdleTime', function() {
      context('When dbMaxIdleTime is null', function() {
        it('Should return false', function() {
          settings.dbMaxIdleTime = null;
          expect(settings.validate().valid).to.equal(false);
        })
      });
      context('When dbMaxIdleTime is undefined', function() {
        it('Should return false', function() {
          delete settings.dbMaxIdleTime
          expect(settings.validate().valid).to.equal(false);
        })
      });
      context('When dbMaxIdleTime is not an integer', function() {
        it('Should return false', function() {
          settings.dbMaxIdleTime = "1000";
          expect(settings.validate().valid).to.equal(false);
        })
      });
      context('When the dbMaxIdleTime is out of range', function() {
        it('Should return false', function() {
          settings.dbMaxIdleTime = -1000;
          expect(settings.validate().valid).to.equal(false);
        })
      });
    });


    context('Validate dbUri', function() {
      context('When dbUri is null', function() {
        it('Should return false', function() {
          settings.db.uri = null
          expect(settings.validate().valid).to.equal(false);
        })
      });
      context('When dbUri is undefined', function() {
        it('Should return false', function() {
          delete settings.db.uri
          expect(settings.validate().valid).to.equal(false);
        })
      });
    });

    context('Validate schedulerUri', function() {
      context('When schedulerUri is null', function() {
        it('Should return false', function() {
          settings.schedulerUri = null
          expect(settings.validate().valid).to.equal(false);
        })
      });
      context('When schedulerUri is undefined', function() {
        it('Should return false', function() {
          delete settings.schedulerUri
          expect(settings.validate().valid).to.equal(false);
        })
      });
    });
  });

  context('dbUri', function() {
    it('Should filter the last slash', function() {
      var dbSetting = configSetting({dbUri: defaultConfig.dbUri + '/'}).db;
      expect(dbSetting.uri).to.equal(defaultConfig.dbUri);
      expect(dbSetting.server).to.equal("postgres://postgres@server:80");
      expect(dbSetting.name).to.equal("dbname");
    });

    context('When the dbUri is mixed case', function() {
      it('Should be lowercased', function() {
        var dbSetting = configSetting({dbUri: defaultConfig.dbUri.toUpperCase()}).db;
        expect(dbSetting.uri).to.equal(defaultConfig.dbUri);
      });
    });
  });

  context('schedulerUri', function() {
    it('Should filter the last slash', function() {
      var apiSetting = configSetting({schedulerUri: defaultConfig.schedulerUri + '/'}).schedulerUri;
      expect(apiSetting).to.equal(defaultConfig.schedulerUri);
    });

    context('When the schedulerUri is upper case', function() {
      it('Should be lowercased', function() {
        var apiSetting = configSetting({schedulerUri: defaultConfig.schedulerUri.toUpperCase()}).schedulerUri;
        expect(apiSetting).to.equal(defaultConfig.schedulerUri);
      });
    });
  });
});
