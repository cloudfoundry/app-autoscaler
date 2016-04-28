package org.cloudfoundry.autoscaler.data.couchdb.dao;

import java.lang.reflect.Field;
import java.util.List;
import java.util.Map;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.data.couchdb.CouchdbStorageService;
import org.cloudfoundry.autoscaler.data.couchdb.dao.impl.TriggerRecordDAOImpl;
import org.cloudfoundry.autoscaler.data.couchdb.document.TriggerRecord;
import org.cloudfoundry.autoscaler.test.testcase.base.JerseyTestBase;

import static org.cloudfoundry.autoscaler.test.constant.Constants.*;
import static org.junit.Assert.*;

import org.junit.Test;

public class TriggerRecordDAOTest extends JerseyTestBase{
	
	private static final Logger logger = Logger.getLogger(TriggerRecordDAOTest.class);
	private static TriggerRecordDAO dao = null;
	public TriggerRecordDAOTest()throws Exception{
		super();
		initConnection();
	}
	@Test
	public void triggerRecordDAOTest() throws Exception{
		
		dao.get(TRIGGERRECORDDOCTESTID);
		assertTrue(!logContains(DOCUMENTNOTFOUNDERRORMSG));
		dao.tryGet(TRIGGERRECORDDOCTESTID + "TMP");
		assertTrue(!logContains(DOCUMENTNOTFOUNDERRORMSG));
		dao.get(TRIGGERRECORDDOCTESTID + "TMP");
		assertTrue(logContains(DOCUMENTNOTFOUNDERRORMSG));
		
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
