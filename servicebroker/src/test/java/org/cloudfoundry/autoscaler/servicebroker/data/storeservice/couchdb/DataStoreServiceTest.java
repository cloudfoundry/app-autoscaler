package org.cloudfoundry.autoscaler.servicebroker.data.storeservice.couchdb;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertFalse;
import static org.junit.Assert.assertNotNull;
import static org.junit.Assert.assertNull;
import static org.junit.Assert.assertTrue;

import java.util.List;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.servicebroker.data.entity.ApplicationInstance;
import org.cloudfoundry.autoscaler.servicebroker.data.entity.ServiceInstance;
import org.cloudfoundry.autoscaler.servicebroker.data.storeservice.IDataStoreService;
import org.cloudfoundry.autoscaler.servicebroker.test.util.TestConstants;
import org.junit.After;
import org.junit.AfterClass;
import org.junit.Before;
import org.junit.BeforeClass;
import org.junit.Test;


/**
 *
 */
public class DataStoreServiceTest {

 
    private static final Logger logger = Logger.getLogger(DataStoreServiceTest.class);
     @Test
    public void clouandDataStoreServiceTest() throws InterruptedException {
        final String m = "clouandDataStoreServiceTest";
        logger.info(m + " started");

        IDataStoreService store = CouchdbStoreService.getInstance();
        this.createServiceInstanceTest(store);
        this.bindApplicationTest(store);
        this.unbindApplication(store);
        this.deleteServiceInstance(store);
        logger.info(m + " completed");
    }

      private void createServiceInstanceTest(IDataStoreService store) {
        store.createService(TestConstants.TEST_SERVICE_ID, TestConstants.TEST_SERVER_URL, TestConstants.TEST_ORG_ID, TestConstants.TEST_SPACE_ID);

        ServiceInstance serviceInstance = store.getServiceInstanceByServiceId(TestConstants.TEST_SERVICE_ID);
        assertNotNull(serviceInstance);
        assertEquals(serviceInstance.getServiceId(), TestConstants.TEST_SERVICE_ID);
        assertEquals(serviceInstance.getOrgId(), TestConstants.TEST_ORG_ID);
        assertEquals(serviceInstance.getSpaceId(), TestConstants.TEST_SPACE_ID);
        assertEquals(serviceInstance.getServerUrl(), TestConstants.TEST_SERVER_URL);

        List<String> serviceIdArray = store.getServiceInstanceIdByServerURL(TestConstants.TEST_SERVER_URL);
        assertNotNull(serviceIdArray);
        Boolean found = false;
        for (String serviceId : serviceIdArray) {
            if (serviceId.equals(TestConstants.TEST_SERVICE_ID)) {
                found = true;
            }
        }
        assertTrue(found);
    }

    private void bindApplicationTest(IDataStoreService store) {
        store.bindApplication(TestConstants.TEST_APPLICATION_ID, TestConstants.TEST_SERVICE_ID, TestConstants.TEST_BINDING_ID);

        ApplicationInstance applicationInstance = store.getBoundAppByBindingId(TestConstants.TEST_BINDING_ID);
        assertNotNull(applicationInstance);
        assertEquals(applicationInstance.getBindingId(), TestConstants.TEST_BINDING_ID);
        assertEquals(applicationInstance.getServiceId(), TestConstants.TEST_SERVICE_ID);
        assertEquals(applicationInstance.getAppId(), TestConstants.TEST_APPLICATION_ID);

        List<ApplicationInstance> applicationInstanceArray = store.getBoundAppByAppId(TestConstants.TEST_APPLICATION_ID);
        assertNotNull(applicationInstanceArray);
        Boolean found = false;
        for (ApplicationInstance entry : applicationInstanceArray) {
            if (entry.getBindingId().equalsIgnoreCase(TestConstants.TEST_BINDING_ID) &&
                entry.getServiceId().equalsIgnoreCase(TestConstants.TEST_SERVICE_ID)) {
                found = true;
            }
        }
        assertTrue(found);

        List<String> applicationInstanceIdArray = store.getBoundAppIdByServiceId(TestConstants.TEST_SERVICE_ID);
        assertNotNull(applicationInstanceIdArray);
        found = false;
        for (String appId : applicationInstanceIdArray) {
            if (appId.equals(TestConstants.TEST_APPLICATION_ID)) {
                found = true;
            }
        }
        assertTrue(found);

    }

    private void unbindApplication(IDataStoreService store) {
        store.unbindApplication(TestConstants.TEST_BINDING_ID);

        ApplicationInstance applicationInstance = store.getBoundAppByBindingId(TestConstants.TEST_BINDING_ID);
        assertNull(applicationInstance);

        Boolean found = false;
        List<ApplicationInstance> applicationInstanceArray = store.getBoundAppByAppId(TestConstants.TEST_APPLICATION_ID);
        if (applicationInstanceArray != null) {
            for (ApplicationInstance entry : applicationInstanceArray) {
                if (entry.getBindingId().equalsIgnoreCase(TestConstants.TEST_BINDING_ID) &&
                    entry.getServiceId().equalsIgnoreCase(TestConstants.TEST_SERVICE_ID)) {
                    found = true;
                }
            }
        }
        assertFalse(found);

        List<String> applicationInstanceIdArray = store.getBoundAppIdByServiceId(TestConstants.TEST_SERVICE_ID);
        found = false;
        if (applicationInstanceIdArray != null) {
            for (String appId : applicationInstanceIdArray) {
                if (appId.equals(TestConstants.TEST_APPLICATION_ID)) {
                    found = true;
                }
            }
        }
        assertFalse(found);

    }

    private void deleteServiceInstance(IDataStoreService store) {
        store.deleteService(TestConstants.TEST_SERVICE_ID);

        ServiceInstance serviceInstance = store.getServiceInstanceByServiceId(TestConstants.TEST_SERVICE_ID);
        assertNull(serviceInstance);

        List<String> serviceIdArray = store.getServiceInstanceIdByServerURL(TestConstants.TEST_SERVER_URL);
        Boolean found = false;
        if (serviceIdArray != null) {
            for (String serviceId : serviceIdArray) {
                if (serviceId.equals(TestConstants.TEST_SERVICE_ID)) {
                    found = true;
                }
            }
        }
        assertFalse(found);

    }
}
