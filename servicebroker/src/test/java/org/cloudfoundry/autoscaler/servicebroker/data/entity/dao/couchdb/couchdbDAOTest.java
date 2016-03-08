package org.cloudfoundry.autoscaler.servicebroker.data.entity.dao.couchdb;

import static org.junit.Assert.*;

import java.lang.reflect.Field;
import java.util.List;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.servicebroker.data.entity.ApplicationInstance;
import org.cloudfoundry.autoscaler.servicebroker.data.entity.ServiceInstance;
import org.cloudfoundry.autoscaler.servicebroker.data.storeservice.couchdb.CouchdbStoreService;
import org.cloudfoundry.autoscaler.servicebroker.test.util.TestConstants;
import org.junit.After;
import org.junit.Before;
import org.junit.BeforeClass;
import org.junit.Test;


/**
 *
 */
public class couchdbDAOTest {

    private static final Logger logger = Logger.getLogger(couchdbDAOTest.class);

    private static ApplicationInstanceDAOImpl applicationStore;
    private static ServiceInstanceDAOImpl serviceStore;

    @BeforeClass
    public static void setUpBeforeClass() throws Exception {
        initConnection();
    }

 
    @Test
    public void applicationInstanceTest() throws InterruptedException {
        final String m = "applicationInstanceTest";
        logger.info(m + " started");

        ApplicationInstance application = new ApplicationInstance();
        application.setAppId(TestConstants.TEST_APPLICATION_ID);
        application.setBindingId(TestConstants.TEST_BINDING_ID);
        application.setServiceId(TestConstants.TEST_SERVICE_ID);

        //add & get
        applicationStore.add(application);
        ApplicationInstance entity = (ApplicationInstance) applicationStore.get(application.getId());
        assertNotNull(entity);
        assertTrue(entity.getRevision().startsWith("1-"));

        //update & query
        application.setAppId(application.getAppId() + "-2");
        applicationStore.update(application);
        entity = (ApplicationInstance) applicationStore.get(application.getId());
        assertNotNull(entity);
        assertTrue(entity.getRevision().startsWith("2-"));

        List<ApplicationInstance> queryEntities = applicationStore.findByAppId(application.getAppId());
        assertNotNull(queryEntities);
        Boolean found = false;
        for (ApplicationInstance queryEntity : queryEntities) {
            if (queryEntity.getAppId().equals(application.getAppId())) {
                found = true;
            }
        }
        assertTrue(found);

        queryEntities = applicationStore.findByServiceId(application.getServiceId());
        assertNotNull(queryEntities);
        found = false;
        for (ApplicationInstance queryEntity : queryEntities) {
            if (queryEntity.getServiceId().equals(application.getServiceId())) {
                found = true;
            }
        }
        assertTrue(found);

        queryEntities = applicationStore.findByBindingId(application.getBindingId());
        assertNotNull(queryEntities);
        found = false;
        for (ApplicationInstance queryEntity : queryEntities) {
            if (queryEntity.getBindingId().equals(application.getBindingId())) {
                found = true;
            }
        }
        assertTrue(found);

        //remove 
        applicationStore.remove(application);
        entity = (ApplicationInstance) applicationStore.get(application.getId());
        assertNull(entity);

        logger.info(m + " completed");
    }

    @Test
    public void serviceInstanceTest() throws InterruptedException {
        final String m = "serviceInstanceTest";
        logger.info(m + " started");

        ServiceInstance service = new ServiceInstance();
        service.setServiceId(TestConstants.TEST_SERVICE_ID);
        service.setOrgId(TestConstants.TEST_ORG_ID);
        service.setSpaceId(TestConstants.TEST_SPACE_ID);
        service.setServerUrl(TestConstants.TEST_SERVER_URL);

        //add & get
        serviceStore.add(service);
        ServiceInstance entity = (ServiceInstance) serviceStore.get(service.getId());
        assertNotNull(entity);
        assertTrue(entity.getRevision().startsWith("1-"));

        //update & query
        service.setServiceId(service.getServiceId() + "-2");
        serviceStore.update(service);
        entity = (ServiceInstance) serviceStore.get(service.getId());
        assertNotNull(entity);
        assertTrue(entity.getRevision().startsWith("2-"));

        List<ServiceInstance> queryEntities = serviceStore.findByServiceId(service.getServiceId());
        assertNotNull(queryEntities);
        Boolean found = false;
        for (ServiceInstance queryEntity : queryEntities) {
            if (queryEntity.getServiceId().equals(service.getServiceId())) {
                found = true;
            }
        }
        assertTrue(found);

        queryEntities = serviceStore.findByServerURL(service.getServerUrl());
        assertNotNull(queryEntities);
        found = false;
        for (ServiceInstance queryEntity : queryEntities) {
            if (queryEntity.getServerUrl().equals(service.getServerUrl())) {
                found = true;
            }
        }
        assertTrue(found);

        //remove 
        serviceStore.remove(service);
        entity = (ServiceInstance) serviceStore.get(service.getId());
        assertNull(entity);

        logger.info(m + " completed");
    }

    private static void initConnection() throws NoSuchFieldException, SecurityException, IllegalArgumentException, IllegalAccessException {

        Field field = null;
        CouchdbStoreService cloudantStorage = CouchdbStoreService.getInstance();

        field = CouchdbStoreService.class.getDeclaredField("applicationStore");
        field.setAccessible(true);
        applicationStore = (ApplicationInstanceDAOImpl) field.get(cloudantStorage);
        assertNotNull(applicationStore);

        field = CouchdbStoreService.class.getDeclaredField("serviceStore");
        field.setAccessible(true);
        serviceStore = (ServiceInstanceDAOImpl) field.get(cloudantStorage);
        assertNotNull(serviceStore);
    }
}
