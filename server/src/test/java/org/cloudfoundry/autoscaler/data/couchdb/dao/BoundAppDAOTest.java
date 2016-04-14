package org.cloudfoundry.autoscaler.data.couchdb.dao;

import java.lang.reflect.Field;
import java.util.List;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.data.couchdb.CouchdbStorageService;
import org.cloudfoundry.autoscaler.data.couchdb.dao.impl.AppAutoScaleStateDAOImpl;
import org.cloudfoundry.autoscaler.data.couchdb.dao.impl.BoundAppDAOImpl;
import org.cloudfoundry.autoscaler.data.couchdb.document.AppAutoScaleState;
import org.cloudfoundry.autoscaler.data.couchdb.document.BoundApp;
import org.cloudfoundry.autoscaler.rest.mock.couchdb.CouchDBDocumentManager;

import static org.cloudfoundry.autoscaler.test.constant.Constants.*;
import static org.junit.Assert.*;

import org.junit.Test;

import com.sun.jersey.test.framework.JerseyTest;

public class BoundAppDAOTest extends JerseyTest{
	
	private static final Logger logger = Logger.getLogger(BoundAppDAOTest.class);
	private static BoundAppDAO dao = null;
	public BoundAppDAOTest()throws Exception{
		super("org.cloudfoundry.autoscaler.rest");
		initConnection();
	}
	@Override
	public void tearDown() throws Exception{
		super.tearDown();
		CouchDBDocumentManager.getInstance().initDocuments();
	}
	@Test
	public void BoundAppDAOTestTest() throws Exception{
		List<BoundApp> list = dao.findAll();
		assertTrue(list.size() > 0);
	    list = dao.findByServerName(SERVERNAME);
		assertTrue(list.size() > 0);
		list = dao.findByServiceId(TESTSERVICEID);
		assertTrue(list.size() > 0);list = dao.findByServiceIdAndAppId(TESTSERVICEID, TESTAPPID);
		assertTrue(list.size() > 0);
		list = dao.getAllBoundApps(SERVERNAME);
		assertTrue(list.size() > 0);
		BoundApp app = dao.findByAppId(TESTAPPID);
		assertNotNull(app);
		app.setAppType("java");
		dao.update(app);
		app = dao.findByAppId(TESTAPPID);
		assertTrue(app.getRevision().startsWith("2-"));
		dao.remove(app);
		app = dao.findByAppId(TESTAPPID);
		assertNull(app);
		
	}
	private static void initConnection() throws NoSuchFieldException, SecurityException, IllegalArgumentException, IllegalAccessException{
		CouchdbStorageService service = CouchdbStorageService.getInstance();
		Field field = null;
		field = CouchdbStorageService.class.getDeclaredField("boundAppDao");
		field.setAccessible(true);
		dao = (BoundAppDAOImpl) field.get(service);
        assertNotNull(dao);

	}

}
