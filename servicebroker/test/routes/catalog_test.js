'use strict';
var supertest = require("supertest");
var expect = require('chai').expect;

var Ajv = require('ajv');
var fs = require('fs');
var path = require('path');
var BrokerServer = require(path.join(__dirname, '../../lib/server.js'));
var configFilePath = path.join(__dirname, '../../config/settings.json');
var settings = require(path.join(__dirname, '../../lib/config/setting.js'))((JSON.parse(
  fs.readFileSync(configFilePath, 'utf8'))));
var auth = new Buffer(settings.username + ":" + settings.password).toString('base64');
var fakeValidPolicy = JSON.parse(fs.readFileSync(path.join(__dirname, '../fakePolicy.json'), 'utf8'));

var catalog = JSON.parse(fs.readFileSync(path.join(__dirname, '../../config/catalog.json'), 'utf8'));
var schemaValidation = JSON.parse(fs.readFileSync(path.resolve(settings.schemaValidationPath), 'utf8'));
for (let i = 0; i < catalog.services[0].plans.length; i++) {
  catalog.services[0].plans[i].schemas = schemaValidation.schemas
}

var ajv = new Ajv();
ajv.addMetaSchema(require('ajv/lib/refs/json-schema-draft-06.json'));

describe('getCatalog RESTful API', function() {
  var servers, publicServer, internalServer, healthServer, validate;
  beforeEach(function() {
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

  it("should return catalog json", function(done) {
    supertest(publicServer)
      .get("/v2/catalog")
      .set("Authorization", "Basic " + auth)
      .expect(200)
      .expect("Content-type", /json/)
      .end(function(err, res) {
        expect(JSON.stringify(res.body)).to.be.equal(JSON.stringify(catalog));
        expect(res.body.services[0].plans[0]).to.have.property('schemas');
        validate = ajv.compile(res.body.services[0].plans[0].schemas.service_binding.create.parameters);
        expect(validate(fakeValidPolicy)).to.be.true;
        done();
      });
  });

});
describe("Validate schema definition",function(){

  var servers, publicServer, internalServer, healthServer, validate, catalogSchema, fakePolicy;
  before(function(done){
    servers = BrokerServer(settings, catalog, function(){});
    publicServer = servers.publicServer;
    internalServer = servers.internalServer;
    healthServer = servers.healthServer;
    supertest(publicServer)
      .get("/v2/catalog")
      .set("Authorization", "Basic " + auth)
      .expect(200)
      .expect("Content-type", /json/)
      .end(function(err, res) {
        expect(JSON.stringify(res.body)).to.be.equal(JSON.stringify(catalog));
        expect(res.body.services[0].plans[0]).to.have.property('schemas');
        catalogSchema = res.body.services[0].plans[0].schemas.service_binding.create.parameters;
        validate = ajv.compile(res.body.services[0].plans[0].schemas.service_binding.create.parameters);
        expect(validate(fakeValidPolicy)).to.be.true;
        done();
      });
  });

  beforeEach(function(){
    fakePolicy = JSON.parse(fs.readFileSync(path.join(__dirname, '../fakePolicy.json'), 'utf8'));
  });

  after(function(done) {
    publicServer.close(function() {
      internalServer.close(function(){
        healthServer.close(done);
      });
    })
  });

  it("validate instance_max_count", function() {
    fakePolicy.instance_max_count = -5;
    expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
  });

  it("validate instance_min_count", function() {
    fakePolicy.instance_min_count = -5;
    expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
  });

  it("validate existance of scaling_rules or schedules",function(){
    expect(ajv.validate(catalogSchema,fakePolicy)).to.be.true;
    delete fakePolicy.scaling_rules;
    delete fakePolicy.schedules;
    expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
    fakePolicy = JSON.parse(fs.readFileSync(path.join(__dirname, '../fakePolicy.json'), 'utf8'));
    delete fakePolicy.scaling_rules;
    expect(ajv.validate(catalogSchema,fakePolicy)).to.be.true;
    fakePolicy = JSON.parse(fs.readFileSync(path.join(__dirname, '../fakePolicy.json'), 'utf8'));
    delete fakePolicy.schedules;
    expect(ajv.validate(catalogSchema,fakePolicy)).to.be.true;
  });

  context("validate scaling rules",function(){

    it("validate scaling rule structure",function(){
      expect(fakePolicy.scaling_rules[0]).to.have.property("metric_type");
      expect(fakePolicy.scaling_rules[0]).to.have.property("threshold");
      expect(fakePolicy.scaling_rules[0]).to.have.property("adjustment");
      expect(fakePolicy.scaling_rules[0]).to.have.property("operator");
      delete fakePolicy.scaling_rules[0].threshold;
      expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
    });

    it("validate metric_type with non alphanumeric characters",function(){
      fakePolicy.scaling_rules[0].metric_type = "ab cd$"
      expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
    });

    it("validate metric_type with underscore",function(){
      fakePolicy.scaling_rules[0].metric_type = "my_metric2"
      expect(ajv.validate(catalogSchema,fakePolicy)).to.be.true;
    });

    it("validate operator",function(){
      fakePolicy.scaling_rules[0].operator = "$"
      expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
    });

    it("validate threshold",function(){
      fakePolicy.scaling_rules[0].threshold = "not a number";
      expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
    });

    it("validate adjustment",function(){
      fakePolicy.scaling_rules[0].adjustment = "*2"
      expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
    });

    it("validate breach_duration_secs if exists",function(){
      fakePolicy.scaling_rules[0].breach_duration_secs = 3601;
      expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
      fakePolicy.scaling_rules[0].breach_duration_secs = 59;
      expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
      delete fakePolicy.scaling_rules[0].breach_duration_secs;
      expect(ajv.validate(catalogSchema,fakePolicy)).to.be.true;
    });

    it("validate cool_down_secs if exists",function(){
      fakePolicy.scaling_rules[0].cool_down_secs = 3601;
      expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
      fakePolicy.scaling_rules[0].cool_down_secs = 59;
      expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
      delete fakePolicy.scaling_rules[0].cool_down_secs;
      expect(ajv.validate(catalogSchema,fakePolicy)).to.be.true;
    });

  });

  context("validate schedules",function(){

    it("validate timezone",function(){
      delete fakePolicy.schedules.timezone;
      expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
    });

    it("validate existance of recurring_schedule or specific_date",function(){
      expect(ajv.validate(catalogSchema,fakePolicy)).to.be.true;
      delete fakePolicy.schedules.recurring_schedule;
      expect(ajv.validate(catalogSchema,fakePolicy)).to.be.true;
      fakePolicy = JSON.parse(fs.readFileSync(path.join(__dirname, '../fakePolicy.json'), 'utf8'));
      delete fakePolicy.schedules.specific_date;
      expect(ajv.validate(catalogSchema,fakePolicy)).to.be.true;
      delete fakePolicy.schedules.recurring_schedule;
      expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
    });
    context("validate recurring_schedule",function(){

      it("validate start_time",function(){
        fakePolicy.schedules.recurring_schedule[0].start_time = "24:10"
        expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
        fakePolicy.schedules.recurring_schedule[0].start_time = "21:70"
        expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
        delete fakePolicy.schedules.recurring_schedule[0].start_time;
        expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
      });

      it("validate end_time",function(){
        fakePolicy.schedules.recurring_schedule[0].end_time = "24:10"
        expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
        fakePolicy.schedules.recurring_schedule[0].end_time = "21:70"
        expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
        delete fakePolicy.schedules.recurring_schedule[0].end_time;
        expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
      });

      it("validate start_date",function(){
        fakePolicy.schedules.recurring_schedule[0].start_date = "2018-23-12"
        expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
        fakePolicy.schedules.recurring_schedule[0].start_date = "2022-11-33"
        expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
        fakePolicy.schedules.recurring_schedule[0].start_date = "20220-11-33"
        expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
        delete fakePolicy.schedules.recurring_schedule[0].start_date;
        expect(ajv.validate(catalogSchema,fakePolicy)).to.be.true;
      });

      it("validate end_date",function(){
        fakePolicy.schedules.recurring_schedule[0].end_date = "2018-03-35"
        expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
        fakePolicy.schedules.recurring_schedule[0].end_date = "2022-11-33"
        expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
        fakePolicy.schedules.recurring_schedule[0].end_date = "20220-11-33"
        expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
        delete fakePolicy.schedules.recurring_schedule[0].end_date;
        expect(ajv.validate(catalogSchema,fakePolicy)).to.be.true;
      });

      it("validate instance_max_count", function() {
        fakePolicy.schedules.recurring_schedule[0].instance_max_count = -5;
        expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
      });
    
      it("validate instance_min_count", function() {
        fakePolicy.schedules.recurring_schedule[0].instance_min_count = -5;
        expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
      });

      it("validate initial_min_instance_count", function() {
        fakePolicy.schedules.recurring_schedule[0].initial_min_instance_count = -5;
        expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
      });

      it("validate existance of either days_of_week or days_of_month",function(){
        fakePolicy.schedules.recurring_schedule[0].days_of_month = [1,2,3];
        expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
        delete fakePolicy.schedules.recurring_schedule[0].days_of_month;
        expect(ajv.validate(catalogSchema,fakePolicy)).to.be.true;
        fakePolicy = JSON.parse(fs.readFileSync(path.join(__dirname, '../fakePolicy.json'), 'utf8'));
        fakePolicy.schedules.recurring_schedule[1].days_of_week = [1,2,3];
        expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
        delete fakePolicy.schedules.recurring_schedule[1].days_of_week;
        expect(ajv.validate(catalogSchema,fakePolicy)).to.be.true;
       
      });

      it("validate days_of_month",function(){
        fakePolicy.schedules.recurring_schedule[1].days_of_month = [1,2,35]
        expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
        fakePolicy.schedules.recurring_schedule[1].days_of_month = [];
        expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
        fakePolicy.schedules.recurring_schedule[1].days_of_month = ["1","2","3"];
        expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
      });

      it("validate days_of_week",function(){
        fakePolicy.schedules.recurring_schedule[0].days_of_week = [1,2,8]
        expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
        fakePolicy.schedules.recurring_schedule[0].days_of_week = [];
        expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
        fakePolicy.schedules.recurring_schedule[0].days_of_week = ["1","2","3"];
        expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
      });
    });

    context("validate specific_date",function(){

      it("validate start_date_time",function(){
        fakePolicy.schedules.specific_date[0].start_date_time = "2018-03-21"
        expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
        fakePolicy.schedules.specific_date[0].start_date_time = "2018-03-21T12:65"
        expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
        fakePolicy.schedules.specific_date[0].start_date_time = "2018-03-21T25:40"
        expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
        fakePolicy.schedules.specific_date[0].start_date_time = "2018-03-32T13:40"
        expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
        fakePolicy.schedules.specific_date[0].start_date_time = "2018-13-12T13:40"
        expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
        fakePolicy.schedules.specific_date[0].start_date_time = "10018-13-12T13:40"
        expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
        delete fakePolicy.schedules.specific_date[0].start_date_time;
        expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
      });

      it("validate end_date_time",function(){
        fakePolicy.schedules.specific_date[0].end_date_time = "2018-03-21"
        expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
        fakePolicy.schedules.specific_date[0].end_date_time = "2018-03-21T12:65"
        expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
        fakePolicy.schedules.specific_date[0].end_date_time = "2018-03-21T25:40"
        expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
        fakePolicy.schedules.specific_date[0].end_date_time = "2018-03-32T13:40"
        expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
        fakePolicy.schedules.specific_date[0].end_date_time = "2018-13-12T13:40"
        expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
        fakePolicy.schedules.specific_date[0].end_date_time = "10018-13-12T13:40"
        expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
        delete fakePolicy.schedules.specific_date[0].end_date_time;
        expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
      });

      it("validate instance_max_count", function() {
        fakePolicy.schedules.specific_date[0].instance_max_count = -5;
        expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
      });
    
      it("validate instance_min_count", function() {
        fakePolicy.schedules.specific_date[0].instance_min_count = -5;
        expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
      });

      it("validate initial_min_instance_count", function() {
        fakePolicy.schedules.specific_date[0].initial_min_instance_count = -5;
        expect(ajv.validate(catalogSchema,fakePolicy)).to.be.false;
      });
    });

  });
});
