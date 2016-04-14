package org.cloudfoundry.autoscaler.data.couchdb.dao;

import java.lang.reflect.Field;
import java.util.List;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.data.couchdb.CouchdbStorageService;
import org.cloudfoundry.autoscaler.data.couchdb.dao.impl.AppAutoScaleStateDAOImpl;
import org.cloudfoundry.autoscaler.data.couchdb.dao.impl.ScalingHistoryDAOImpl;
import org.cloudfoundry.autoscaler.data.couchdb.document.AppAutoScaleState;
import org.cloudfoundry.autoscaler.data.couchdb.document.ScalingHistory;
import org.cloudfoundry.autoscaler.rest.mock.couchdb.CouchDBDocumentManager;

import static org.cloudfoundry.autoscaler.test.constant.Constants.*;
import static org.junit.Assert.*;

import org.junit.Test;

import com.sun.jersey.test.framework.JerseyTest;

public class ScalingHistoryDAOTest extends JerseyTest{
	
	private static final Logger logger = Logger.getLogger(ScalingHistoryDAOTest.class);
	private static ScalingHistoryDAO dao = null;
	public ScalingHistoryDAOTest()throws Exception{
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
		assertNull(history.getId());
		
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
