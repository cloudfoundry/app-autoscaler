var supertest = require("supertest");
var should = require("should");
var fs = require('fs');
var path = require('path');
var uuid = require('uuid');
var async = require('asyncawait/async');
var await = require('asyncawait/await');
var ServiceInstanceDao = require(path.join(__dirname, '../../../lib/dao/sequelize/serviceInstanceDao.js'));
var dbConnectionInfo = require(path.join(__dirname, '../../../lib/util/dbConnectionInfo.js'));

// This agent refers to PORT where program is runninng.
var dbUri = dbConnectionInfo.dbConnectionUri;

var serviceInstanceDao = new ServiceInstanceDao(dbUri);

describe('Unit Test for serviceInstanceDAO', function() {
  beforeEach(function(done) {
    serviceInstanceDao.removeAll();
    done();
  });

  it('should return 0 record with getAll() when the database is empty', async(function(done) {
    var list = await (serviceInstanceDao.getAll());
    list.length.should.equal(0);
    done();
  }));

  it('should return 1 element with getByServiceId() after add() a new record', async(function(done) {

    var serviceId = uuid.v4();
    var orgId = uuid.v4();
    var spaceId = uuid.v4();
    var result  = await (serviceInstanceDao.add(serviceId, orgId, spaceId));
    
    var instance = await (serviceInstanceDao.getByServiceId(serviceId));
    instance.get('serviceId').should.equal(serviceId);
    instance.get('orgId').should.equal(orgId);
    instance.get('spaceId').should.equal(spaceId);
    done();
  }));

  it('should return TRUE or FALSE to indicate whether the record is duplicate with previous after add() a record', async(function(done) {
    var serviceId = uuid.v4();
    var orgId = uuid.v4();
    var spaceId = uuid.v4();

    var resultCreate = await (serviceInstanceDao.add(serviceId, orgId, spaceId));
    resultCreate[1].should.equal(true);
    var resultCreateAgain = await (serviceInstanceDao.add(serviceId, orgId, spaceId));
    resultCreateAgain[1].should.equal(false);
    done();
 }));


  it('should get ERROR when unique service id constraint is violate when add() a new record', async(function(done) {
    var serviceId = uuid.v4();
    var orgId = uuid.v4();
    var spaceId = uuid.v4();

    var resultCreate = await (serviceInstanceDao.add(serviceId, orgId, spaceId));
    var anotherOrgId = uuid.v4();
    var anotherSpaceId = uuid.v4();
   
    var resultCreateAgain = await (serviceInstanceDao.add(serviceId, anotherOrgId, anotherSpaceId));
    resultCreateAgain.err.name.should.equal('SequelizeUniqueConstraintError');
    done();

  }));


  it('should return 0 element when add() a record and removeByServiceId()', async(function(done) {
    var serviceId = uuid.v4();
    var orgId = uuid.v4();
    var spaceId = uuid.v4();
    await (serviceInstanceDao.add(serviceId, orgId, spaceId));
    await (serviceInstanceDao.removeByServiceId(serviceId));
    var count = await (serviceInstanceDao.countByServiceId(serviceId));
    count.should.equal(0);
    done();
  }));
});