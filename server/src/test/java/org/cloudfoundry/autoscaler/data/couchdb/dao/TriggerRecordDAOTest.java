package org.cloudfoundry.autoscaler.data.couchdb.dao;

import java.lang.reflect.Field;
import java.util.List;
import java.util.Map;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.data.couchdb.CouchdbStorageService;
import org.cloudfoundry.autoscaler.data.couchdb.dao.impl.AppAutoScaleStateDAOImpl;
import org.cloudfoundry.autoscaler.data.couchdb.dao.impl.TriggerRecordDAOImpl;
import org.cloudfoundry.autoscaler.data.couchdb.document.AppAutoScaleState;
import org.cloudfoundry.autoscaler.data.couchdb.document.TriggerRecord;
import org.cloudfoundry.autoscaler.rest.mock.couchdb.CouchDBDocumentManager;

import static org.cloudfoundry.autoscaler.test.constant.Constants.*;
import static org.junit.Assert.*;

import org.junit.Test;

import com.sun.jersey.test.framework.JerseyTest;

public class TriggerRecordDAOTest extends JerseyTest{
	
	private static final Logger logger = Logger.getLogger(TriggerRecordDAOTest.class);
	private static TriggerRecordDAO dao = null;
	public TriggerRecordDAOTest()throws Exception{
		super("org.cloudfoundry.autoscaler.rest");
		initConnection();
	}
	@Override
	public void tearDown() throws Exception{
		super.tearDown();
		CouchDBDocumentManager.getInstance().initDocuments();
	}
	@Test
	public void triggerRecordDAOTest() throws Exception{
		List<TriggerRecord> list = dao.findAll();
		assertTrue(list.size() > 0);
		list = dao.findByAppId(TESTAPPID);
		assertTrue(list.size() > 0);
		list = dao.findByServerName(SERVERNAME);
		assertTrue(list.size() > 0);
		Map<String,List<TriggerRecord>> map = dao.getAllTriggers(SERVERNAME);
		assertTrue(map.size() > 0);
		dao.removeByAppId(TESTAPPID);
		list = dao.findByAppId(TESTAPPID);
		assertTrue(list.size() == 0);
	}
	private static void initConnection() throws NoSuchFieldException, SecurityException, IllegalArgumentException, IllegalAccessException{
		CouchdbStorageService service = CouchdbStorageService.getInstance();
		Field field = null;
		field = CouchdbStorageService.class.getDeclaredField("triggerRecordDao");
		field.setAccessible(true);
		dao = (TriggerRecordDAOImpl) field.get(service);
        assertNotNull(dao);

	}

}
