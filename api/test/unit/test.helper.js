'use strict';
process.env.NODE_TLS_REJECT_UNAUTHORIZED = '0';
var path = require('path');
module.exports = function (basedir, settings) {
    var isMissing = function (value) {
        return typeof (value) === "undefined" || value === null;
    }

    if (!isMissing(settings.tls)) {
        settings.tls = {
            keyFile: path.join(basedir, path.basename(settings.tls.keyFile)),
            certFile: path.join(basedir, path.basename(settings.tls.certFile)),
            caCertFile: path.join(basedir, path.basename(settings.tls.caCertFile))
        };
    };
    if (!isMissing(settings.publicTls)) {
        settings.publicTls = {
            keyFile: path.join(basedir, path.basename(settings.publicTls.keyFile)),
            certFile: path.join(basedir, path.basename(settings.publicTls.certFile)),
            caCertFile: path.join(basedir, path.basename(settings.publicTls.caCertFile))
        };
    };
    if (!isMissing(settings.metricsCollector.tls)) {
        settings.metricsCollector.tls = {
            keyFile: path.join(basedir, path.basename(settings.metricsCollector.tls.keyFile)),
            certFile: path.join(basedir, path.basename(settings.metricsCollector.tls.certFile)),
            caCertFile: path.join(basedir, path.basename(settings.metricsCollector.tls.caCertFile))
        };
    };
    if (!isMissing(settings.scalingEngine.tls)) {
        settings.scalingEngine.tls = {
            keyFile: path.join(basedir, path.basename(settings.scalingEngine.tls.keyFile)),
            certFile: path.join(basedir, path.basename(settings.scalingEngine.tls.certFile)),
            caCertFile: path.join(basedir, path.basename(settings.scalingEngine.tls.caCertFile))
        };
    };
    if (!isMissing(settings.scheduler.tls)) {
        settings.scheduler.tls = {
            keyFile: path.join(basedir, path.basename(settings.scheduler.tls.keyFile)),
            certFile: path.join(basedir, path.basename(settings.scheduler.tls.certFile)),
            caCertFile: path.join(basedir, path.basename(settings.scheduler.tls.caCertFile))
        };
    };
    if (!isMissing(settings.serviceOffering.serviceBroker.tls)) {
        settings.serviceOffering.serviceBroker.tls = {
            keyFile: path.join(basedir, path.basename(settings.serviceOffering.serviceBroker.tls.keyFile)),
            certFile: path.join(basedir, path.basename(settings.serviceOffering.serviceBroker.tls.certFile)),
            caCertFile: path.join(basedir, path.basename(settings.serviceOffering.serviceBroker.tls.caCertFile))
        };
    };
    if (!isMissing(settings.infoFilePath)) {
        settings.infoFilePath = path.join(basedir, settings.infoFilePath);
    };
    return settings;
};