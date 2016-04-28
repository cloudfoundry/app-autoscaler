package org.cloudfoundry.autoscaler.servicebroker.data.storeservice.couchdb;

import static org.cloudfoundry.autoscaler.servicebroker.test.constant.Constants.TESTAPPID;
import static org.cloudfoundry.autoscaler.servicebroker.test.constant.Constants.TESTBINDINGID;
import static org.cloudfoundry.autoscaler.servicebroker.test.constant.Constants.TESTORGID;
import static org.cloudfoundry.autoscaler.servicebroker.test.constant.Constants.TESTSERVERURL;
import static org.cloudfoundry.autoscaler.servicebroker.test.constant.Constants.TESTSERVICEID;
import static org.cloudfoundry.autoscaler.servicebroker.test.constant.Constants.TESTSPACEID;
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
import org.cloudfoundry.autoscaler.servicebroker.rest.mock.couchdb.CouchDBDocumentManager;
import org.junit.Test;

import com.sun.jersey.test.framework.JerseyTest;


/**
 *
 */
public class DataStoreServiceTest extends JerseyTest{

 
    private static final Logger logger = Logger.getLogger(DataStoreServiceTest.class);
    public  DataStoreServiceTest() throws Exception{
		super("org.cloudfoundry.autoscaler.servicebroker.rest.mock.couchdb");
	}
    @Override
	public void tearDown() throws Exception{
		super.tearDown();
		CouchDBDocumentManager.getInstance().initDocuments();
	}
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
        store.createService(TESTSERVICEID, TESTSERVERURL, TESTORGID, TESTSPACEID);

        ServiceInstance serviceInstance = store.getServiceInstanceByServiceId(TESTSERVICEID);
        assertNotNull(serviceInstance);
        logger.info("constant serviceId is " + TESTSERVICEID + ", and instance id is " + serviceInstance.getServiceId());
        assertEquals(serviceInstance.getServiceId(), TESTSERVICEID);
        assertEquals(serviceInstance.getOrgId(), TESTORGID);
        assertEquals(serviceInstance.getSpaceId(), TESTSPACEID);
        assertEquals(serviceInstance.getServerUrl(), TESTSERVERURL);

        List<String> serviceIdArray = store.getServiceInstanceIdByServerURL(TESTSERVERURL);
        assertNotNull(serviceIdArray);
        Boolean found = false;
        for (String serviceId : serviceIdArray) {
            if (serviceId.equals(TESTSERVICEID)) {
                found = true;
            }
        }
        assertTrue(found);
    }

    private void bindApplicationTest(IDataStoreService store) {
        store.bindApplication(TESTAPPID, TESTSERVICEID, TESTBINDINGID);

        ApplicationInstance applicationInstance = store.getBoundAppByBindingId(TESTBINDINGID);
        assertNotNull(applicationInstance);
        assertEquals(applicationInstance.getBindingId(), TESTBINDINGID);
        assertEquals(applicationInstance.getServiceId(), TESTSERVICEID);
        assertEquals(applicationInstance.getAppId(), TESTAPPID);

        List<ApplicationInstance> applicationInstanceArray = store.getBoundAppByAppId(TESTAPPID);
        assertNotNull(applicationInstanceArray);
        Boolean found = false;
        for (ApplicationInstance entry : applicationInstanceArray) {
            if (entry.getBindingId().equalsIgnoreCase(TESTBINDINGID) &&
                entry.getServiceId().equalsIgnoreCase(TESTSERVICEID)) {
                found = true;
            }
        }
        assertTrue(found);

        List<String> applicationInstanceIdArray = store.getBoundAppIdByServiceId(TESTSERVICEID);
        assertNotNull(applicationInstanceIdArray);
        found = false;
        for (String appId : applicationInstanceIdArray) {
            if (appId.equals(TESTAPPID)) {
                found = true;
            }
        }
        assertTrue(found);

    }

    private void unbindApplication(IDataStoreService store) {
        store.unbindApplication(TESTBINDINGID);

        ApplicationInstance applicationInstance = store.getBoundAppByBindingId(TESTBINDINGID);
        assertNull(applicationInstance);

        Boolean found = false;
        List<ApplicationInstance> applicationInstanceArray = store.getBoundAppByAppId(TESTAPPID);
        if (applicationInstanceArray != null) {
            for (ApplicationInstance entry : applicationInstanceArray) {
                if (entry.getBindingId().equalsIgnoreCase(TESTBINDINGID) &&
                    entry.getServiceId().equalsIgnoreCase(TESTSERVICEID)) {
                    found = true;
                }
            }
        }
        assertFalse(found);

        List<String> applicationInstanceIdArray = store.getBoundAppIdByServiceId(TESTSERVICEID);
        found = false;
        if (applicationInstanceIdArray != null) {
            for (String appId : applicationInstanceIdArray) {
                if (appId.equals(TESTAPPID)) {
                    found = true;
                }
            }
        }
        assertFalse(found);

    }

    private void deleteServiceInstance(IDataStoreService store) {
        store.deleteService(TESTSERVICEID);

        ServiceInstance serviceInstance = store.getServiceInstanceByServiceId(TESTSERVICEID);
        assertNull(serviceInstance);

        List<String> serviceIdArray = store.getServiceInstanceIdByServerURL(TESTSERVERURL);
        Boolean found = false;
        if (serviceIdArray != null) {
            for (String serviceId : serviceIdArray) {
                if (serviceId.equals(TESTSERVICEID)) {
                    found = true;
                }
            }
        }
        assertFalse(found);

    }
}
