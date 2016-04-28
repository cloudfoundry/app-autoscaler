package org.cloudfoundry.autoscaler.servicebroker.data.entity.dao.couchdb;

import static org.cloudfoundry.autoscaler.servicebroker.test.constant.Constants.TESTAPPID;
import static org.cloudfoundry.autoscaler.servicebroker.test.constant.Constants.TESTBINDINGID;
import static org.cloudfoundry.autoscaler.servicebroker.test.constant.Constants.TESTORGID;
import static org.cloudfoundry.autoscaler.servicebroker.test.constant.Constants.TESTSERVERURL;
import static org.cloudfoundry.autoscaler.servicebroker.test.constant.Constants.TESTSERVICEID;
import static org.cloudfoundry.autoscaler.servicebroker.test.constant.Constants.TESTSPACEID;
import static org.junit.Assert.assertNotNull;
import static org.junit.Assert.assertNull;
import static org.junit.Assert.assertTrue;

import java.lang.reflect.Field;
import java.util.List;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.servicebroker.data.entity.ApplicationInstance;
import org.cloudfoundry.autoscaler.servicebroker.data.entity.ServiceInstance;
import org.cloudfoundry.autoscaler.servicebroker.data.storeservice.couchdb.CouchdbStoreService;
import org.cloudfoundry.autoscaler.servicebroker.test.testcase.base.JerseyTestBase;
import org.junit.Test;


/**
 *
 */
public class couchdbDAOTest extends JerseyTestBase{

    private static final Logger logger = Logger.getLogger(couchdbDAOTest.class);

    private static ApplicationInstanceDAOImpl applicationStore;
    private static ServiceInstanceDAOImpl serviceStore;
    public  couchdbDAOTest() throws Exception{
		super();
		 initConnection();
	}
    @Test
    public void applicationInstanceTest() throws InterruptedException {
        final String m = "applicationInstanceTest";
        logger.info(m + " started");
		
        ApplicationInstance application = new ApplicationInstance();
        application.setAppId(TESTAPPID);
        application.setBindingId(TESTBINDINGID);
        application.setServiceId(TESTSERVICEID);

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
        service.setServiceId(TESTSERVICEID);
        service.setOrgId(TESTORGID);
        service.setSpaceId(TESTSPACEID);
        service.setServerUrl(TESTSERVERURL);

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
