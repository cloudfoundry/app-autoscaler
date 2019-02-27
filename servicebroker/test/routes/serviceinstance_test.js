'use strict';
var supertest = require("supertest");
var uuid = require('uuid');

var fs = require('fs');
var path = require('path');
var BrokerServer = require(path.join(__dirname, '../../lib/server.js'));
var configFilePath = path.join(__dirname, '../../config/settings.json');
var settings = require(path.join(__dirname, '../../lib/config/setting.js'))((JSON.parse(
  fs.readFileSync(configFilePath, 'utf8'))));
var catalog = JSON.parse(fs.readFileSync(path.join(__dirname, '../../config/catalog.json'), 'utf8'));

var models = require('../../lib/models')(settings.db);
var serviceInstance = models.service_instance;

var auth = new Buffer(settings.username + ":" + settings.password).toString('base64');

describe('service instance RESTful API', function() {
  var servers, publicServer, internalServer, healthServer, serviceInstanceId, orgId, spaceId, orgIdAgain, spaceIdAgain;
  serviceInstanceId = uuid.v4();
  orgId = uuid.v4();
  spaceId = uuid.v4();
  orgIdAgain = uuid.v4();
  spaceIdAgain = uuid.v4();


  beforeEach(function() {
    settings = require(path.join(__dirname, '../../lib/config/setting.js'))((JSON.parse(
      fs.readFileSync(configFilePath, 'utf8'))));
    catalog = JSON.parse(fs.readFileSync(path.join(__dirname, '../../config/catalog.json'), 'utf8'));
    serviceInstance.truncate({ cascade: true });

    servers = BrokerServer(settings, catalog, function(){});
    publicServer = servers.publicServer;
    internalServer = servers.internalServer;
    healthServer = servers.healthServer;
  });

  afterEach(function(done) {
    publicServer.close(function() {
      internalServer.close(function(){
        healthServer.close(done);
      });
    })
  });

  context('Provision service', function() {
    context('when there is no record', function() {
      context('when settings.dashboardRedirectUri and catalog.services[0].dashboard_client.redirect_uri both are present',function(){

        it("creates a new instance with 201 with catalog.services[0].dashboard_client.redirect_uri as dashboard base uri", function(done) {
          supertest(publicServer)
            .put("/v2/service_instances/" + serviceInstanceId)
            .set("Authorization", "Basic " + auth)
            .send({ "organization_guid": orgId, "space_guid": spaceId })
            .expect(201)
            .expect('Content-Type', /json/)
            .expect({
              dashboard_url: 'https://dashboard-redirect-uri-catalog.example.com/manage/'+ serviceInstanceId
            }, done);
        });
      });

      context('when only catalog.services[0].dashboard_client.redirect_uri is present',function(){
        beforeEach(function(){
          delete settings.dashboardRedirectUri;
        });
        it("creates a new instance with 201 with catalog.services[0].dashboard_client.redirect_uri as dashboard base uri", function(done) {
          supertest(publicServer)
            .put("/v2/service_instances/" + serviceInstanceId)
            .set("Authorization", "Basic " + auth)
            .send({ "organization_guid": orgId, "space_guid": spaceId })
            .expect(201)
            .expect('Content-Type', /json/)
            .expect({
              dashboard_url: 'https://dashboard-redirect-uri-catalog.example.com/manage/'+ serviceInstanceId
            }, done);
        });
      });
      
      context('when only settings.dashboardRedirectUri is present',function(){
        beforeEach(function(){
          delete catalog.services[0].dashboard_client;
        });
        it("creates a new instance with 201 with settings.dashboardRedirectUri as dashboard base uri", function(done) {
          supertest(publicServer)
            .put("/v2/service_instances/" + serviceInstanceId)
            .set("Authorization", "Basic " + auth)
            .send({ "organization_guid": orgId, "space_guid": spaceId })
            .expect(201)
            .expect('Content-Type', /json/)
            .expect({
              dashboard_url: 'https://dashboard-redirect-uri-settings.example.com/manage/'+ serviceInstanceId
            }, done);
        });
      });
      
      context('when settings.dashboardRedirectUri and catalog.services[0].dashboard_client.redirect_uri both are absent',function(){
        beforeEach(function(){
          delete catalog.services[0].dashboard_client;
          delete settings.dashboardRedirectUri;
        });
        it("creates a new instance with 201 with empty string as dashboard_url", function(done) {
          supertest(publicServer)
            .put("/v2/service_instances/" + serviceInstanceId)
            .set("Authorization", "Basic " + auth)
            .send({ "organization_guid": orgId, "space_guid": spaceId })
            .expect(201)
            .expect('Content-Type', /json/)
            .expect({
              dashboard_url: ''
            }, done);
        });
      });
    });

    context('when an instance already exists', function() {

      beforeEach(function(done) {
        supertest(publicServer)
          .put("/v2/service_instances/" + serviceInstanceId)
          .set("Authorization", "Basic " + auth)
          .send({ "organization_guid": orgId, "space_guid": spaceId })
          .expect(201)
          .expect('Content-Type', /json/)
          .expect({
            dashboard_url: 'https://dashboard-redirect-uri-catalog.example.com/manage/'+ serviceInstanceId
          }, done);
      });

      context('when orgId and spaceId are identical', function() {
        it('returns a 200', function(done) {
          supertest(publicServer)
            .put("/v2/service_instances/" + serviceInstanceId)
            .set("Authorization", "Basic " + auth)
            .set('Accept', 'application/json')
            .send({ "organization_guid": orgId, "space_guid": spaceId })
            .expect(200)
            .expect('Content-Type', /json/)
            .expect({
              dashboard_url: 'https://dashboard-redirect-uri-catalog.example.com/manage/'+ serviceInstanceId
            }, done);
        });
      });

      context('when orgId and spaceId are conflict to previous record', function() {
        it('returns a 409', function(done) {
          supertest(publicServer)
            .put("/v2/service_instances/" + serviceInstanceId)
            .set("Authorization", "Basic " + auth)
            .send({ "organization_guid": orgIdAgain, "space_guid": spaceIdAgain })
            .expect(409, done);
        });
      });
    });

  });


  context('Deprovision service ', function() {
    context('when there is no record', function() {
      it("delete an nonexist instance with 410", function(done) {
        supertest(publicServer)
          .delete("/v2/service_instances/" + "nonexistid")
          .set("Authorization", "Basic " + auth)
          .expect(410)
          .expect('Content-Type', /json/)
          .expect({}, done);
      });
    });

    context('when an instance already exists', function() {
      beforeEach(function(done) {
        supertest(publicServer)
            .put("/v2/service_instances/" + serviceInstanceId)
            .set("Authorization", "Basic " + auth)
            .send({ "organization_guid": orgId, "space_guid": spaceId })
            .expect(201)
            .expect('Content-Type', /json/)
            .expect({
              dashboard_url: 'https://dashboard-redirect-uri-catalog.example.com/manage/'+ serviceInstanceId
            }, done);
      });

      it("delete an instance with 200", function(done) {
        supertest(publicServer)
          .delete("/v2/service_instances/" + serviceInstanceId)
          .set("Authorization", "Basic " + auth)
          .expect(200)
          .expect('Content-Type', /json/)
          .expect({}, done);
      });
    });


  });

});
