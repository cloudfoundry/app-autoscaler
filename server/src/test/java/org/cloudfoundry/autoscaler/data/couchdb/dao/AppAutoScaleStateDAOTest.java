package org.cloudfoundry.autoscaler.data.couchdb.dao;

import java.lang.reflect.Field;
import java.util.List;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.data.couchdb.CouchdbStorageService;
import org.cloudfoundry.autoscaler.data.couchdb.dao.impl.AppAutoScaleStateDAOImpl;
import org.cloudfoundry.autoscaler.data.couchdb.document.AppAutoScaleState;
import org.cloudfoundry.autoscaler.test.testcase.base.JerseyTestBase;

import static org.cloudfoundry.autoscaler.test.constant.Constants.*;
import static org.junit.Assert.*;

import org.junit.Test;


public class AppAutoScaleStateDAOTest extends JerseyTestBase{
	
	private static final Logger logger = Logger.getLogger(AppAutoScaleStateDAOTest.class);
	private static AppAutoScaleStateDAO dao = null;
	public AppAutoScaleStateDAOTest()throws Exception{
		super();
		initConnection();
	}
	@Test
	public void appAutoScaleStateDAOTest(){
		dao.get(APPAUTOSCALESTATEDOCTESTID);
		assertTrue(!logContains(DOCUMENTNOTFOUNDERRORMSG));
		dao.tryGet(APPAUTOSCALESTATEDOCTESTID + "TMP");
		assertTrue(!logContains(DOCUMENTNOTFOUNDERRORMSG));
		dao.get(APPAUTOSCALESTATEDOCTESTID + "TMP");
		assertTrue(logContains(DOCUMENTNOTFOUNDERRORMSG));
		
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
