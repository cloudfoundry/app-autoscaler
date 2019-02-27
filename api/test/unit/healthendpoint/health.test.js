
"use strict";

var request = require("supertest");
var expect = require("chai").expect;
var fs = require("fs");
var path = require("path");
var HttpStatus = require('http-status-codes');
var settings = require(path.join(__dirname, '../../../lib/config/setting.js'))((JSON.parse(
    fs.readFileSync(path.join(__dirname, '../../../config/settings.json'), 'utf8'))));
  var relativePath = path.relative(process.cwd(), path.join(__dirname, "../../../../test-certs"));
  var testSetting = require(path.join(__dirname, '../test.helper.js'))(relativePath, settings);
var API = require('../../../app.js');
var internalServer;
var publicServer;
var healthServer;
var servers;

describe("health", function () {
    before(function () {
        servers = API(testSetting, function () { });
        internalServer = servers.internalServer;
        publicServer = servers.publicServer;
        healthServer = servers.healthServer;
    });
    after(function (done) {
        internalServer.close(function () {
            publicServer.close(function () {
                healthServer.close(done);
            });
        });
    });
    it("return health metrics", function (done) {
        request(healthServer)
            .get('/')
            .end(function (error, result) {
                expect(result.statusCode).to.equal(200);
                expect(result.text).contains("autoscaler_apiserver_process_cpu_user_seconds_total");
                expect(result.text).contains("autoscaler_apiserver_process_cpu_system_seconds_total");
                expect(result.text).contains("autoscaler_apiserver_process_cpu_seconds_total");
                expect(result.text).contains("autoscaler_apiserver_process_start_time_seconds");
                expect(result.text).contains("autoscaler_apiserver_process_resident_memory_bytes");
                expect(result.text).contains("autoscaler_apiserver_nodejs_eventloop_lag_seconds");
                expect(result.text).contains("autoscaler_apiserver_nodejs_active_handles_total");
                expect(result.text).contains("autoscaler_apiserver_nodejs_active_requests_total");
                expect(result.text).contains("autoscaler_apiserver_nodejs_heap_size_total_bytes");
                expect(result.text).contains("autoscaler_apiserver_nodejs_heap_size_used_bytes");
                expect(result.text).contains("autoscaler_apiserver_nodejs_heap_space_size_total_bytes");
                expect(result.text).contains("autoscaler_apiserver_nodejs_heap_space_size_used_bytes");
                expect(result.text).contains("autoscaler_apiserver_nodejs_heap_space_size_available_bytes");
                expect(result.text).contains("autoscaler_apiserver_nodejs_version_info");
                done();
            });
    });
});

