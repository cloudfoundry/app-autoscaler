'use strict';
var path = require('path');
var url = require('url');
var fs = require('fs');
module.exports = function(settingsObj) {
  var db = function(dbUri) {
   if (dbUri != null) {
     let uri = dbUri.replace(/\/$/g, "");
     var dbUriObj = url.parse(dbUri, true);
     var dbSSL = null;
     if (dbUriObj.query) {
       var sslmode = dbUriObj.query['sslmode'];
       if (sslmode && (sslmode === 'prefer' || sslmode === 'require' || sslmode === 'verify-ca' || sslmode === 'verify-full')) {
         if (sslmode === 'verify-ca' || sslmode === 'verify-full') {
           if (dbUriObj.query['sslrootcert']) {
             dbSSL = {
               rejectUnauthorized: true,
               ca: fs.readFileSync(dbUriObj.query['sslrootcert'])
             };
           } else {
             dbSSL = {
               rejectUnauthorized: true,
             };
           }

         } else {
           dbSSL = {
             rejectUnauthorized: false,
           };
         }
       }
     }

     return {
       uri: uri,
       ssl: dbSSL
      };
    }
  };
  var cleanUpUri = function (uri) {
    if (uri) {
      uri = uri.replace(/\/$/g, "").toLowerCase();
    }
    return uri;
  };
  var addProtocol = function (uri) {
    var pattern = new RegExp("^http[s]{0,1}://");
    if (uri && (!pattern.test(uri))) {
      uri = "https://" + uri;
    }
    return uri;
  }

  var isMissing = function (value) {
    return typeof (value) === "undefined" || value === null;
  }
  var isNumber = function (value) {
    return typeof (value) === "number";
  }
  var isString = function (value) {
    return typeof (value) === "string";
  }
  var isObject = function (value) {
    return typeof (value) === "object";
  }
  var isBoolean = function (value) {
    return typeof (value) === "boolean";
  }

  var settings = {
    port: settingsObj.port,
    cfApi: addProtocol(cleanUpUri(settingsObj.cfApi)),
    cfClientId: settingsObj.cfClientId,
    cfClientSecret: settingsObj.cfClientSecret,
    skipSSLValidation: settingsObj.skipSSLValidation,
    cacheTTL: settingsObj.cacheTTL,
    publicPort: settingsObj.publicPort,
    scheduler: settingsObj.scheduler,
    scalingEngine: settingsObj.scalingEngine,
    metricsCollector: settingsObj.metricsCollector,
    eventGenerator: settingsObj.eventGenerator,
    tls: settingsObj.tls,
    publicTls: settingsObj.publicTls,
    infoFilePath: settingsObj.infoFilePath,
    serviceOffering: settingsObj.serviceOffering,
    httpClientTimeout: settingsObj.httpClientTimeout,
    minBreachDurationSecs: settingsObj.minBreachDurationSecs,
    minCoolDownSecs: settingsObj.minCoolDownSecs,
    healthPort: settingsObj.healthPort
    
  };
  if (settingsObj.db) {
    var dbObj = db(settingsObj.db.uri);
    settings.db = {
      maxConnections: settingsObj.db.maxConnections,
      minConnections: settingsObj.db.minConnections,
      idleTimeout: settingsObj.db.idleTimeout,
      uri: dbObj.uri,
      ssl: dbObj.ssl
    }
  }
  if (!isMissing(settings.scheduler)) {
    settings.scheduler.uri = addProtocol(cleanUpUri(settings.scheduler.uri));
  }
  if (!isMissing(settings.scalingEngine)) {
    settings.scalingEngine.uri = addProtocol(cleanUpUri(settings.scalingEngine.uri));
  }
  if (!isMissing(settings.metricsCollector)) {
    settings.metricsCollector.uri = addProtocol(cleanUpUri(settings.metricsCollector.uri));
  }
  if (!isMissing(settings.eventGenerator)) {
    settings.eventGenerator.uri = addProtocol(cleanUpUri(settings.eventGenerator.uri));
  }
  if (!isMissing(settings.serviceOffering.serviceBroker)) {
    settings.serviceOffering.serviceBroker.uri = addProtocol(cleanUpUri(settings.serviceOffering.serviceBroker.uri));
  }

  settings.validate = function () {
    if (isMissing(settings.port)) {
      return { valid: false, message: "port is required" }
    }
    if (!isNumber(settings.port)) {
      return { valid: false, message: "port must be a number" };
    }
    if (settings.port < 1 || settings.port > 65535) {
      return { valid: false, message: "value of port must be between 1 and 65535" };
    }

    if (isMissing(settings.publicPort)) {
      return { valid: false, message: "publicPort is required" }
    }
    if (!isNumber(settings.publicPort)) {
      return { valid: false, message: "publicPort must be a number" };
    }
    if (settings.publicPort < 1 || settings.publicPort > 65535) {
      return { valid: false, message: "value of publicPort must be between 1 and 65535" };
    }

    if (isMissing(settings.healthPort)) {
      return { valid: false, message: "healthPort is required" }
    }
    if (!isNumber(settings.healthPort)) {
      return { valid: false, message: "healthPort must be a number" };
    }
    if (settings.healthPort < 0 || settings.healthPort > 65535) {
      return { valid: false, message: "value of healthPort must be between 0 and 65535" };
    }

    if (settings.port == settings.publicPort || settings.port == settings.healthPort || settings.healthPort == settings.publicPort) {
      return { valid: false, message: "internal api port, public api port and health port should be different" }
    }
    if (isMissing(settings.infoFilePath)) {
      return { valid: false, message: "infoFilePath is required" }
    }
    if (!isString(settings.infoFilePath)) {
      return { valid: false, message: "infoFilePath must be a string" };
    }
    if (isMissing(settings.cfApi)) {
      return { valid: false, message: "cfApi is required" }
    }
    if (!isString(settings.cfApi)) {
      return { valid: false, message: "cfApi must be a string" };
    }
    if (isMissing(settings.cfClientId)) {
      return { valid: false, message: "cfClientId is required" }
    }
    if (!isString(settings.cfClientId)) {
      return { valid: false, message: "cfClientId must be a string" };
    }
    if (isMissing(settings.cfClientSecret)) {
      return { valid: false, message: "cfClientSecret is required" }
    }
    if (!isString(settings.cfClientSecret)) {
      return { valid: false, message: "cfClientSecret must be a string" };
    }
    if (isMissing(settings.skipSSLValidation)) {
      return { valid: false, message: 'skipSSLValidation is required' };
    }
    if (!isBoolean(settings.skipSSLValidation)) {
      return { valid: false, message: 'skipSSLValidation must be a boolean' };
    }
    if (isMissing(settings.cacheTTL)) {
      return { valid: false, message: "cacheTTL is required" }
    }
    if (!isNumber(settings.cacheTTL)) {
      return { valid: false, message: "cacheTTL must be a number" };
    }
    if (isMissing(settings.db.maxConnections)) {
      return { valid: false, message: "db.maxConnections is required" };
    }
    if (!isNumber(settings.db.maxConnections)) {
      return { valid: false, message: "db.maxConnections must be a number" };
    }
    if (settings.db.maxConnections <= 0) {
      return { valid: false, message: "db.maxConnections must be greater than 0" };
    }
    if (isMissing(settings.db.minConnections)) {
      return { valid: false, message: "db.minConnections is required" };
    }
    if (!isNumber(settings.db.minConnections)) {
      return { valid: false, message: "db.minConnections must be a number" };
    }
    if (settings.db.minConnections < 0) {
      return { valid: false, message: "db.minConnections must be greater than or equal to 0" };
    }
    if (isMissing(settings.db.idleTimeout)) {
      return { valid: false, message: "db.idleTimeout is required" };
    }
    if (!isNumber(settings.db.idleTimeout)) {
      return { valid: false, message: "db.idleTimeout must be a number" };
    }
    if (settings.db.idleTimeout <= 0) {
      return { valid: false, message: "db.idleTimeout must be greater than 0" };
    }
    if (isMissing(settings.db.uri)) {
      return { valid: false, message: "db.uri is required" };
    }
    if (!isString(settings.db.uri)) {
      return { valid: false, message: "db.uri must be a string" };
    }
    if (!isMissing(settings.tls)) {
      if (!isObject(settings.tls)) {
        return { valid: false, message: "tls must be an object" };
      }
      if (isMissing(settings.tls.keyFile)) {
        return { valid: false, message: "tls.keyFile is required" };
      }
      if (!isString(settings.tls.keyFile)) {
        return { valid: false, message: "tls.keyFile must be a string" };
      }
      if (isMissing(settings.tls.certFile)) {
        return { valid: false, message: "tls.certFile is required" };
      }
      if (!isString(settings.tls.certFile)) {
        return { valid: false, message: "tls.certFile must be a string" };
      }
      if (isMissing(settings.tls.caCertFile)) {
        return { valid: false, message: "tls.caCertFile is required" };
      }
      if (!isString(settings.tls.caCertFile)) {
        return { valid: false, message: "tls.caCertFile must be a string" };
      }
    }
    if (!isMissing(settings.publicTls)) {
      if (!isObject(settings.publicTls)) {
        return { valid: false, message: "publicTls must be an object" };
      }
      if (isMissing(settings.publicTls.keyFile)) {
        return { valid: false, message: "publicTls.keyFile is required" };
      }
      if (!isString(settings.publicTls.keyFile)) {
        return { valid: false, message: "publicTls.keyFile must be a string" };
      }
      if (isMissing(settings.publicTls.certFile)) {
        return { valid: false, message: "publicTls.certFile is required" };
      }
      if (!isString(settings.publicTls.certFile)) {
        return { valid: false, message: "publicTls.certFile must be a string" };
      }
      if (isMissing(settings.publicTls.caCertFile)) {
        return { valid: false, message: "publicTls.caCertFile is required" };
      }
      if (!isString(settings.publicTls.caCertFile)) {
        return { valid: false, message: "publicTls.caCertFile must be a string" };
      }
    }
    if (isMissing(settings.scheduler)) {
      return { valid: false, message: "scheduler is required" };
    }
    if (isMissing(settings.scheduler.uri)) {
      return { valid: false, message: "scheduler.uri is required" };
    }
    if (!isString(settings.scheduler.uri)) {
      return { valid: false, message: "scheduler.uri must be a string" };
    }
    if (!isMissing(settings.scheduler.tls)) {
      if (!isObject(settings.scheduler.tls)) {
        return { valid: false, message: "scheduler.tls must be an object" };
      }
      if (isMissing(settings.scheduler.tls.keyFile)) {
        return { valid: false, message: "scheduler.tls.keyFile is required" };
      }
      if (!isString(settings.scheduler.tls.keyFile)) {
        return { valid: false, message: "scheduler.tls.keyFile must be a string" };
      }
      if (isMissing(settings.scheduler.tls.caCertFile)) {
        return { valid: false, message: "scheduler.tls.caCertFile is required" };
      }
      if (!isString(settings.scheduler.tls.caCertFile)) {
        return { valid: false, message: "scheduler.tls.caCertFile must be a string" };
      }
      if (isMissing(settings.scheduler.tls.certFile)) {
        return { valid: false, message: "scheduler.tls.certFile is required" };
      }
      if (!isString(settings.scheduler.tls.certFile)) {
        return { valid: false, message: "scheduler.tls.certFile must be a string" };
      }
    }

    if (isMissing(settings.scalingEngine)) {
      return { valid: false, message: "scalingEngine is required" };
    }
    if (isMissing(settings.scalingEngine.uri)) {
      return { valid: false, message: "scalingEngine.uri is required" };
    }
    if (!isString(settings.scalingEngine.uri)) {
      return { valid: false, message: "scalingEngine.uri must be a string" };
    }
    if (!isMissing(settings.scalingEngine.tls)) {
      if (!isObject(settings.scalingEngine.tls)) {
        return { valid: false, message: "scalingEngine.tls must be an object" };
      }
      if (isMissing(settings.scalingEngine.tls.keyFile)) {
        return { valid: false, message: "scalingEngine.tls.keyFile is required" };
      }
      if (!isString(settings.scalingEngine.tls.keyFile)) {
        return { valid: false, message: "scalingEngine.tls.keyFile must be a string" };
      }
      if (isMissing(settings.scalingEngine.tls.caCertFile)) {
        return { valid: false, message: "scalingEngine.tls.caCertFile is required" };
      }
      if (!isString(settings.scalingEngine.tls.caCertFile)) {
        return { valid: false, message: "scalingEngine.tls.caCertFile must be a string" };
      }
      if (isMissing(settings.scalingEngine.tls.certFile)) {
        return { valid: false, message: "scalingEngine.tls.certFile is required" };
      }
      if (!isString(settings.scalingEngine.tls.certFile)) {
        return { valid: false, message: "scalingEngine.tls.certFile must be a string" };
      }
    }

    if (isMissing(settings.metricsCollector)) {
      return { valid: false, message: "metricsCollector is required" };
    }
    if (isMissing(settings.metricsCollector.uri)) {
      return { valid: false, message: "metricsCollector.uri is required" };
    }
    if (!isString(settings.metricsCollector.uri)) {
      return { valid: false, message: "metricsCollector.uri must be a string" };
    }
    if (!isMissing(settings.metricsCollector.tls)) {
      if (!isObject(settings.metricsCollector.tls)) {
        return { valid: false, message: "metricsCollector.tls must be an object" };
      }
      if (isMissing(settings.metricsCollector.tls.keyFile)) {
        return { valid: false, message: "metricsCollector.tls.keyFile is required" };
      }
      if (!isString(settings.metricsCollector.tls.keyFile)) {
        return { valid: false, message: "metricsCollector.tls.keyFile must be a string" };
      }
      if (isMissing(settings.metricsCollector.tls.caCertFile)) {
        return { valid: false, message: "metricsCollector.tls.caCertFile is required" };
      }
      if (!isString(settings.metricsCollector.tls.caCertFile)) {
        return { valid: false, message: "metricsCollector.tls.caCertFile must be a string" };
      }
      if (isMissing(settings.metricsCollector.tls.certFile)) {
        return { valid: false, message: "metricsCollector.tls.certFile is required" };
      }
      if (!isString(settings.metricsCollector.tls.certFile)) {
        return { valid: false, message: "metricsCollector.tls.certFile must be a string" };
      }
    }

    if (isMissing(settings.eventGenerator)) {
      return { valid: false, message: "eventGenerator is required" };
    }
    if (isMissing(settings.eventGenerator.uri)) {
      return { valid: false, message: "eventGenerator.uri is required" };
    }
    if (!isString(settings.eventGenerator.uri)) {
      return { valid: false, message: "eventGenerator.uri must be a string" };
    }
    if (!isMissing(settings.eventGenerator.tls)) {
      if (!isObject(settings.eventGenerator.tls)) {
        return { valid: false, message: "eventGenerator.tls must be an object" };
      }
      if (isMissing(settings.eventGenerator.tls.keyFile)) {
        return { valid: false, message: "eventGenerator.tls.keyFile is required" };
      }
      if (!isString(settings.eventGenerator.tls.keyFile)) {
        return { valid: false, message: "eventGenerator.tls.keyFile must be a string" };
      }
      if (isMissing(settings.eventGenerator.tls.caCertFile)) {
        return { valid: false, message: "eventGenerator.tls.caCertFile is required" };
      }
      if (!isString(settings.eventGenerator.tls.caCertFile)) {
        return { valid: false, message: "eventGenerator.tls.caCertFile must be a string" };
      }
      if (isMissing(settings.eventGenerator.tls.certFile)) {
        return { valid: false, message: "eventGenerator.tls.certFile is required" };
      }
      if (!isString(settings.eventGenerator.tls.certFile)) {
        return { valid: false, message: "eventGenerator.tls.certFile must be a string" };
      }
    }

    if (isMissing(settings.serviceOffering)) {
      return { valid: false, message: 'serviceOffering is required' };
    }
    if (!isObject(settings.serviceOffering)) {
      return { valid: false, message: 'serviceOffering must be an object' };
    }
    if (isMissing(settings.serviceOffering.enabled)) {
      return { valid: false, message: 'serviceOffering.enabled is required' };
    }
    if (!isBoolean(settings.serviceOffering.enabled)) {
      return { valid: false, message: 'serviceOffering.enabled must be a boolean' };
    }
    if (settings.serviceOffering.enabled) {
      if (isMissing(settings.serviceOffering.serviceBroker)) {
        return { valid: false, message: "serviceOffering.serviceBroker is required" };
      }
      if (isMissing(settings.serviceOffering.serviceBroker.uri)) {
        return { valid: false, message: "serviceOffering.serviceBroker.uri is required" };
      }
      if (!isString(settings.serviceOffering.serviceBroker.uri)) {
        return { valid: false, message: "serviceOffering.serviceBroker.uri must be a string" };
      }
      if (!isMissing(settings.serviceOffering.serviceBroker.tls)) {
        if (!isObject(settings.serviceOffering.serviceBroker.tls)) {
          return { valid: false, message: "serviceOffering.serviceBroker.tls must be an object" };
        }
        if (isMissing(settings.serviceOffering.serviceBroker.tls.keyFile)) {
          return { valid: false, message: "serviceOffering.serviceBroker.tls.keyFile is required" };
        }
        if (!isString(settings.serviceOffering.serviceBroker.tls.keyFile)) {
          return { valid: false, message: "serviceOffering.serviceBroker.tls.keyFile must be a string" };
        }
        if (isMissing(settings.serviceOffering.serviceBroker.tls.caCertFile)) {
          return { valid: false, message: "serviceOffering.serviceBroker.tls.caCertFile is required" };
        }
        if (!isString(settings.serviceOffering.serviceBroker.tls.caCertFile)) {
          return { valid: false, message: "serviceOffering.serviceBroker.tls.caCertFile must be a string" };
        }
        if (isMissing(settings.serviceOffering.serviceBroker.tls.certFile)) {
          return { valid: false, message: "serviceOffering.serviceBroker.tls.certFile is required" };
        }
        if (!isString(settings.serviceOffering.serviceBroker.tls.certFile)) {
          return { valid: false, message: "serviceOffering.serviceBroker.tls.certFile must be a string" };
        }
      }
    }
    if (isMissing(settings.httpClientTimeout)) {
      return { valid: false, message: "httpClientTimeout is required" };
    }
    if (!isNumber(settings.httpClientTimeout)) {
      return { valid: false, message: "httpClientTimeout must be a number" };
    }
    if (settings.httpClientTimeout <= 0) {
      return { valid: false, message: "value of httpClientTimeout must be greater than 0" };
    }
    if (isMissing(settings.minBreachDurationSecs)) {
      return { valid: false, message: "minBreachDurationSecs is required" };
    }
    if (!isNumber(settings.minBreachDurationSecs)) {
      return { valid: false, message: "minBreachDurationSecs must be a number" };
    }
    if (settings.minBreachDurationSecs <= 0) {
      return { valid: false, message: "value of minBreachDurationSecs must be greater than 0" };
    }
    if (isMissing(settings.minCoolDownSecs)) {
      return { valid: false, message: "minCoolDownSecs is required" };
    }
    if (!isNumber(settings.minCoolDownSecs)) {
      return { valid: false, message: "minCoolDownSecs must be a number" };
    }
    if (settings.minCoolDownSecs <= 0) {
      return { valid: false, message: "value of minCoolDownSecs must be greater than 0" };
    }

    return { valid: true }
  }

  return settings;
};