package org.cloudfoundry.autoscaler.data.couchdb.dao;

import java.lang.reflect.Field;
import java.util.List;

import org.cloudfoundry.autoscaler.data.couchdb.CouchdbStorageService;
import org.cloudfoundry.autoscaler.data.couchdb.dao.impl.ApplicationDAOImpl;
import org.cloudfoundry.autoscaler.data.couchdb.document.Application;
import org.cloudfoundry.autoscaler.test.testcase.base.JerseyTestBase;

import static org.cloudfoundry.autoscaler.test.constant.Constants.*;
import static org.junit.Assert.*;

import org.junit.Test;


public class ApplicationDAOTest extends JerseyTestBase {

	private static ApplicationDAO dao = null;

	public ApplicationDAOTest() throws Exception {
		super();
		initConnection();
	}

	@Test
	public void applicationDAOTest() {
		dao.get(APPLICATIONDOCTESTID);
		assertTrue(!logContains(DOCUMENTNOTFOUNDERRORMSG));
		dao.tryGet(APPLICATIONDOCTESTID + "TMP");
		assertTrue(!logContains(DOCUMENTNOTFOUNDERRORMSG));
		dao.get(APPLICATIONDOCTESTID + "TMP");
		assertTrue(logContains(DOCUMENTNOTFOUNDERRORMSG));
		List<Application> list = dao.findAll();		
		assertTrue(list.size() > 0);
		Application application = dao.findByBindId(TESTBINDINGID);
		assertNotNull(application);
		list = dao.findByPolicyId(TESTPOLICYID);
		assertTrue(list.size() > 0);
		list = dao.findByServiceId(TESTSERVICEID);
		assertTrue(list.size() > 0);
		list = dao.findByServiceIdAndState(TESTSERVICEID);
		assertTrue(list.size() > 0);
		application = dao.findByAppId(TESTAPPID);
		assertNotNull(application);
		application.setAppType("java");
		dao.update(application);
		application = dao.findByAppId(TESTAPPID);
		assertTrue(application.getRevision().startsWith("2-"));
		dao.remove(application);
		application = dao.findByAppId(TESTAPPID);
		assertNull(application);
	}

	private static void initConnection()
			throws NoSuchFieldException, SecurityException, IllegalArgumentException, IllegalAccessException {
		CouchdbStorageService service = CouchdbStorageService.getInstance();
		Field field = null;
		field = CouchdbStorageService.class.getDeclaredField("applicationDao");
		field.setAccessible(true);
		dao = (ApplicationDAOImpl) field.get(service);
		assertNotNull(dao);

	}

}
