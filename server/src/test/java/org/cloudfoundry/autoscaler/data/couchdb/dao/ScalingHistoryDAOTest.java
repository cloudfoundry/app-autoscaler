package org.cloudfoundry.autoscaler.data.couchdb.dao;

import java.lang.reflect.Field;
import java.util.List;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.data.couchdb.CouchdbStorageService;
import org.cloudfoundry.autoscaler.data.couchdb.dao.impl.ScalingHistoryDAOImpl;
import org.cloudfoundry.autoscaler.data.couchdb.document.ScalingHistory;
import org.cloudfoundry.autoscaler.test.testcase.base.JerseyTestBase;

import static org.cloudfoundry.autoscaler.test.constant.Constants.*;
import static org.junit.Assert.*;

import org.junit.Test;

public class ScalingHistoryDAOTest extends JerseyTestBase{
	
	private static final Logger logger = Logger.getLogger(ScalingHistoryDAOTest.class);
	private static ScalingHistoryDAO dao = null;
	public ScalingHistoryDAOTest()throws Exception{
		super();
		initConnection();
	}
	
	@Test
	public void appAutoScaleStateDAOTest(){
		
		dao.get(SCALINGHISTORYDOCTESTID);
		assertTrue(!logContains(DOCUMENTNOTFOUNDERRORMSG));
		dao.tryGet(SCALINGHISTORYDOCTESTID + "TMP");
		assertTrue(!logContains(DOCUMENTNOTFOUNDERRORMSG));
		dao.get(SCALINGHISTORYDOCTESTID + "TMP");
		assertTrue(logContains(DOCUMENTNOTFOUNDERRORMSG));
		
		List<ScalingHistory> list = dao.findAll();
		assertTrue(list.size() > 0);
		list = dao.findByScalingTime(TESTAPPID, 0, System.currentTimeMillis());
		assertTrue(list.size() > 0);
		ScalingHistory history = list.get(0);
		history.setAdjustment(-1);
		dao.update(history);
		history = (ScalingHistory)dao.get(history.getId());
		assertTrue(history.getRevision().startsWith("2-"));
		dao.remove(history);
		history = (ScalingHistory)dao.get(history.getId());
		assertNull(history);
		
	}
	private static void initConnection() throws NoSuchFieldException, SecurityException, IllegalArgumentException, IllegalAccessException{
		CouchdbStorageService service = CouchdbStorageService.getInstance();
		Field field = null;
		field = CouchdbStorageService.class.getDeclaredField("scalingHistoryDao");
		field.setAccessible(true);
		dao = (ScalingHistoryDAOImpl) field.get(service);
        assertNotNull(dao);

	}

}
