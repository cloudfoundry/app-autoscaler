'use strict';

var path = require('path');
var expect = require('chai').expect;
var configSetting = require(path.join(__dirname, '../../lib/config/setting.js'));
var defaultConfig = {
  "port": 80,
  "username": "username",
  "password": "password",
  "dbUri": "postgres://postgres@server:80/dbname",
  "apiServerUri": "http://apiserveruri"
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

  context('apiServerUri', function() {
    it('Should filter the last slash', function() {
      var apiSetting = configSetting({apiServerUri: defaultConfig.apiServerUri + '/'}).apiServerUri;
      expect(apiSetting).to.equal(defaultConfig.apiServerUri);
    });

    context('When the apiServerUri is upper case', function() {
      it('Should be lowercased', function() {
        var apiSetting = configSetting({apiServerUri: defaultConfig.apiServerUri.toUpperCase()}).apiServerUri;
        expect(apiSetting).to.equal(defaultConfig.apiServerUri);
      });
    });
  });
});
