'use strict';
var expect = require('chai').expect;

var fake_serviceinstance = { 'serviceId':'test_service', 'orgId':'test_org', 'spaceId':'test_space' };

describe('Service_instance Model Definition Test Suite', function() {
  before(function() {
    this.serviceInstance = require('../../lib/model')().service_instance;
    this.serviceInstance.sequelize.sync();
  });
  
  beforeEach(function() {
    this.serviceInstance.truncate({ cascade: true });
  });

  it('Should create a service instance successfully', function() {
    return this.serviceInstance.create(fake_serviceinstance )
      .then(function(instance) {
        expect(instance.serviceId).to.equal(fake_serviceinstance.serviceId);
      });
  });  
  
  it('Should failt to create a service instance for duplicate record', function() {
    this.serviceInstance.create(fake_serviceinstance);
    return this.serviceInstance.create(fake_serviceinstance)
    .then(function(instance) {})
    .catch(function(error) {
      expect(error).to.not.be.null;
      expect(error).to.have.deep.property('name').equal('SequelizeUniqueConstraintError');
      expect(error).to.have.deep.property('message').equal('Validation error');    
    });
  });
      
 
});
