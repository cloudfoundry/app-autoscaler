package org.cloudfoundry.autoscaler.data.couchdb.dao;

import java.lang.reflect.Field;
import java.util.List;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.data.couchdb.CouchdbStorageService;
import org.cloudfoundry.autoscaler.data.couchdb.dao.impl.AppAutoScaleStateDAOImpl;
import org.cloudfoundry.autoscaler.data.couchdb.document.AppAutoScaleState;
import org.cloudfoundry.autoscaler.rest.mock.couchdb.CouchDBDocumentManager;

import static org.cloudfoundry.autoscaler.test.constant.Constants.*;
import static org.junit.Assert.*;

import org.junit.Test;

import com.sun.jersey.test.framework.JerseyTest;

public class AppAutoScaleStateDAOTest extends JerseyTest{
	
	private static final Logger logger = Logger.getLogger(AppAutoScaleStateDAOTest.class);
	private static AppAutoScaleStateDAO dao = null;
	public AppAutoScaleStateDAOTest()throws Exception{
		super("org.cloudfoundry.autoscaler.rest");
		initConnection();
	}
	@Override
	public void tearDown() throws Exception{
		super.tearDown();
		CouchDBDocumentManager.getInstance().initDocuments();
	}
	@Test
	public void appAutoScaleStateDAOTest(){
		AppAutoScaleState state = dao.findByAppId(TESTAPPID);
		assertNotNull(state);
		state.setLastActionEndTime(System.currentTimeMillis());
		dao.update(state);
		state = dao.findByAppId(TESTAPPID);
		assertTrue(state.getRevision().startsWith("2-"));
		List<AppAutoScaleState> list = dao.findAll();
		assertTrue(list.size() > 0);
		dao.remove(state);
		state = dao.findByAppId(TESTAPPID);
		assertNull(state);
	}
	private static void initConnection() throws NoSuchFieldException, SecurityException, IllegalArgumentException, IllegalAccessException{
		CouchdbStorageService service = CouchdbStorageService.getInstance();
		Field field = null;
		field = CouchdbStorageService.class.getDeclaredField("appAutoScaleStateDao");
		field.setAccessible(true);
		dao = (AppAutoScaleStateDAOImpl) field.get(service);
        assertNotNull(dao);

	}

}
