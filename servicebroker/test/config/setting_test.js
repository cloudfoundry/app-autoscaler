'use strict';

var path = require('path');
var expect = require('chai').expect;
var configSetting = require(path.join(__dirname, '../../lib/config/setting.js'));
var setting = {
  "port": 80,
  "username": "username",
  "password": "password",
  "dbUri": "postgres://postgres@server:80/dbname",
  "apiServerUri": "http://apiserveruri"
}
var settingTmp = {};
var settingObj, settingContent, dbSetting;
var apiServerUriParsed;
var settingUndefined;
describe('config setting Test Suite', function() {
  beforeEach(function() {
    settingContent = configSetting(setting).getSetting();

  });
  context('Validate setting', function() {
    context('When setting is correctly', function() {
      beforeEach(function() {
        settingObj = configSetting(setting);
      });
      it('Should return true', function() {
        expect(settingObj.validate(setting).valid).to.equal(true);
      })
    });
    context('Validate port', function() {


      context('When port is null', function() {
        beforeEach(function() {
          settingTmp = {};
          settingTmp.port = null;
          settingTmp.username = setting.username;
          settingTmp.password = setting.password;
          settingTmp.dbUri = setting.dbUri;
          settingTmp.apiServerUri = setting.apiServerUri;
          settingObj = configSetting(settingTmp);

        });
        it('Should return false', function() {
          expect(settingObj.validate().valid).to.equal(false);
        })
      });
      context('When port is undefined', function() {
        beforeEach(function() {
          settingTmp = {};
          settingTmp.username = setting.username;
          settingTmp.password = setting.password;
          settingTmp.dbUri = setting.dbUri;
          settingTmp.apiServerUri = setting.apiServerUri;
          settingObj = configSetting(settingTmp);

        });
        it('Should return false', function() {
          expect(settingObj.validate().valid).to.equal(false);
        })
      });
      context('When port is not an integer', function() {
        beforeEach(function() {
          settingTmp = {};
          settingTmp.port = "80";
          settingTmp.username = setting.username;
          settingTmp.password = setting.password;
          settingTmp.dbUri = setting.dbUri;
          settingTmp.apiServerUri = setting.apiServerUri;
          settingObj = configSetting(settingTmp);

        });
        it('Should return false', function() {
          expect(settingObj.validate().valid).to.equal(false);
        })
      });
    });

    context('Validate username', function() {

      context('When username is null', function() {
        beforeEach(function() {
          settingTmp = {};
          settingTmp.port = setting.port;
          settingTmp.username = null;
          settingTmp.password = setting.password;
          settingTmp.dbUri = setting.dbUri;
          settingTmp.apiServerUri = setting.apiServerUri;
          settingObj = configSetting(settingTmp);

        });
        it('Should return false', function() {
          expect(settingObj.validate().valid).to.equal(false);
        })
      });
      context('When username is undefined', function() {
        beforeEach(function() {
          settingTmp = {};
          settingTmp.port = setting.port;
          settingTmp.password = setting.password;
          settingTmp.dbUri = setting.dbUri;
          settingTmp.apiServerUri = setting.apiServerUri;
          settingObj = configSetting(settingTmp);

        });
        it('Should return false', function() {
          expect(settingObj.validate().valid).to.equal(false);
        })
      });
    });

    context('Validate password', function() {

      context('When password is null', function() {
        beforeEach(function() {
          settingTmp = {};
          settingTmp.port = setting.port;
          settingTmp.password = null;
          settingTmp.dbUri = setting.dbUri;
          settingTmp.apiServerUri = setting.apiServerUri;
          settingObj = configSetting(settingTmp);

        });
        it('Should return false', function() {
          expect(settingObj.validate().valid).to.equal(false);
        })
      });
      context('When password is undefined', function() {
        beforeEach(function() {
          settingTmp = {};
          settingTmp.port = setting.port;
          settingTmp.username = setting.username;
          settingTmp.dbUri = setting.dbUri;
          settingTmp.apiServerUri = setting.apiServerUri;
          settingObj = configSetting(settingTmp);

        });
        it('Should return false', function() {
          expect(settingObj.validate().valid).to.equal(false);
        })
      });
    });

    context('Validate dbUri', function() {

      context('When dbUri is null', function() {
        beforeEach(function() {
          settingTmp = {};
          settingTmp.port = setting.port;
          settingTmp.password = setting.password;
          settingTmp.dbUri = null;
          settingTmp.apiServerUri = setting.apiServerUri;
          settingObj = configSetting(settingTmp);

        });
        it('Should return false', function() {
          expect(settingObj.validate().valid).to.equal(false);
        })
      });
      context('When dbUri is undefined', function() {
        beforeEach(function() {
          settingTmp = {};
          settingTmp.port = setting.port;
          settingTmp.username = setting.username;
          settingTmp.password = setting.password;
          settingTmp.apiServerUri = setting.apiServerUri;
          settingObj = configSetting(settingTmp);

        });
        it('Should return false', function() {
          expect(settingObj.validate().valid).to.equal(false);
        })
      });
      context("When dbUri's format is invalid", function() {
        context("When db user name is not defined", function() {
          beforeEach(function() {
            settingTmp = {};
            settingTmp.port = setting.port;
            settingTmp.username = setting.username;
            settingTmp.password = setting.password;
            settingTmp.apiServerUri = setting.apiServerUri;
            settingTmp.dbUri = "postgres://@host:5432/autoscaler";
            settingObj = configSetting(settingTmp);

          });
          it('Should return false', function() {
            expect(settingObj.validate().valid).to.equal(false);
          })
        });
        context("When db host is not defined", function() {
          beforeEach(function() {
            settingTmp = {};
            settingTmp.port = setting.port;
            settingTmp.username = setting.username;
            settingTmp.password = setting.password;
            settingTmp.apiServerUri = setting.apiServerUri;
            settingTmp.dbUri = "postgres://username@:5432/autoscaler";
            settingObj = configSetting(settingTmp);

          });
          it('Should return false', function() {
            expect(settingObj.validate().valid).to.equal(false);
          })
        });
        context("When db port is not defined", function() {
          beforeEach(function() {
            settingTmp = {};
            settingTmp.port = setting.port;
            settingTmp.username = setting.username;
            settingTmp.password = setting.password;
            settingTmp.apiServerUri = setting.apiServerUri;
            settingTmp.dbUri = "postgres://username@host:/autoscaler";
            settingObj = configSetting(settingTmp);

          });
          it('Should return false', function() {
            expect(settingObj.validate().valid).to.equal(false);
          })
        });
        context("When db name is not defined", function() {
          beforeEach(function() {
            settingTmp = {};
            settingTmp.port = setting.port;
            settingTmp.username = setting.username;
            settingTmp.password = setting.password;
            settingTmp.apiServerUri = setting.apiServerUri;
            settingTmp.dbUri = "postgres://username@host:5432/";
            settingObj = configSetting(settingTmp);

          });
          it('Should return false', function() {
            expect(settingObj.validate().valid).to.equal(false);
          })
        });
      });
    });

    context('Validate apiServerUri', function() {

      context('When apiServerUri is null', function() {
        beforeEach(function() {
          settingTmp = {};
          settingTmp.port = setting.port;
          settingTmp.password = setting.password;
          settingTmp.dbUri = setting.dbUri;
          settingTmp.apiServerUri = null;
          settingObj = configSetting(settingTmp);

        });
        it('Should return false', function() {
          expect(settingObj.validate().valid).to.equal(false);
        })
      });
      context('When apiServerUri is undefined', function() {
        beforeEach(function() {
          settingTmp = {};
          settingTmp.port = setting.port;
          settingTmp.username = setting.username;
          settingTmp.password = setting.password;
          settingTmp.dbUri = setting.dbUri;
          settingObj = configSetting(settingTmp);

        });
        it('Should return false', function() {
          expect(settingObj.validate().valid).to.equal(false);
        })
      });
    });
  });
  it('Should get port correctly', function() {
    expect(settingContent.port).to.equal(setting.port);
  });
  it('Should get username and password correctly', function() {
    expect(settingContent.username).to.equal(setting.username);
    expect(settingContent.password).to.equal(setting.password);
  });
  context('Parse dbUri', function() {
    it('Should parse dbUri correctly', function() {
      var dbSetting = settingContent.db;
      expect(dbSetting.uri).to.equal(setting.dbUri);
      expect(dbSetting.server).to.equal("postgres://postgres@server:80");
      expect(dbSetting.name).to.equal("dbname");
    });

    context('When parse dbUri', function() {

      beforeEach(function() {
        settingTmp = {};
        settingTmp.port = setting.port;
        settingTmp.username = setting.username;
        settingTmp.password = setting.password;
        settingTmp.dbUri = setting.dbUri + '/';
        settingTmp.apiServerUri = setting.apiServerUri;
        dbSetting = configSetting(settingTmp).getSetting().db;

      });
      it('Should filter the last slash', function() {

        expect(dbSetting.uri).to.equal(setting.dbUri);
        expect(dbSetting.server).to.equal("postgres://postgres@server:80");
        expect(dbSetting.name).to.equal("dbname");
      });

    });

    context('When parse dbUri is upper case', function() {

      beforeEach(function() {
        var settingTmp = {};
        settingTmp.port = setting.port;
        settingTmp.username = setting.username;
        settingTmp.password = setting.password;
        settingTmp.dbUri = setting.dbUri.toUpperCase();
        settingTmp.apiServerUri = setting.apiServerUri;
        dbSetting = configSetting(setting).getSetting().db;

      });
      it('Should handle correctly ', function() {
        expect(dbSetting.uri).to.equal(setting.dbUri);
        expect(dbSetting.server).to.equal("postgres://postgres@server:80");
        expect(dbSetting.name).to.equal("dbname");
      });

    });
  });
  context('Parse apiServerUri', function() {
    beforeEach(function(done) {
      apiServerUriParsed = configSetting(setting).getSetting().apiServerUri;
      done();
    });
    it('Should parse apiServerUri correctly', function() {
      expect(apiServerUriParsed).to.equal(setting.apiServerUri);
    });

    context('When parse apiServerUri', function() {
      beforeEach(function() {
        settingTmp = {};
        settingTmp.port = setting.port;
        settingTmp.username = setting.username;
        settingTmp.password = setting.password;
        settingTmp.dbUri = setting.dbUri;
        settingTmp.apiServerUri = setting.apiServerUri + '/';
        apiServerUriParsed = configSetting(settingTmp).getSetting().apiServerUri;

      });
      it('Should filter the last slash ', function() {
        expect(apiServerUriParsed).to.equal(setting.apiServerUri);
      });
    });

    context('When parse dbUri is upper case', function() {
      beforeEach(function() {
        settingTmp = {};
        settingTmp.port = setting.port;
        settingTmp.username = setting.username;
        settingTmp.password = setting.password;
        settingTmp.dbUri = setting.dbUri;
        settingTmp.apiServerUri = setting.apiServerUri.toUpperCase();
        apiServerUriParsed = configSetting(setting).getSetting().apiServerUri;

      });
      it('Should handle correctly', function() {
        expect(apiServerUriParsed).to.equal(setting.apiServerUri);
      });
    });
  });
});