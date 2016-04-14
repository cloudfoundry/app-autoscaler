package org.cloudfoundry.autoscaler.data.couchdb.dao;

import java.lang.reflect.Field;
import java.util.List;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.data.couchdb.CouchdbStorageService;
import org.cloudfoundry.autoscaler.data.couchdb.dao.impl.AppAutoScaleStateDAOImpl;
import org.cloudfoundry.autoscaler.data.couchdb.dao.impl.MetricDBSegmentDAOImpl;
import org.cloudfoundry.autoscaler.data.couchdb.dao.impl.ServiceConfigDAOImpl;
import org.cloudfoundry.autoscaler.data.couchdb.document.AppAutoScaleState;
import org.cloudfoundry.autoscaler.data.couchdb.document.MetricDBSegment;
import org.cloudfoundry.autoscaler.data.couchdb.document.ServiceConfig;
import org.cloudfoundry.autoscaler.rest.mock.couchdb.CouchDBDocumentManager;

import static org.cloudfoundry.autoscaler.test.constant.Constants.*;
import static org.junit.Assert.*;

import org.junit.Test;

import com.sun.jersey.test.framework.JerseyTest;

public class MetricDBSegmentDAOTest extends JerseyTest{
	
	private static final Logger logger = Logger.getLogger(MetricDBSegmentDAOTest.class);
	private static MetricDBSegmentDAO dao = null;
	public MetricDBSegmentDAOTest()throws Exception{
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
		List<MetricDBSegment> list = dao.findAll();
		assertTrue(list.size() > 0);
		MetricDBSegment segment = dao.findByMetricDBPostfix("continuously");
		assertNotNull(segment);
		segment = dao.findByPostfix("continuously");
		assertNotNull(segment);
		list = dao.findByServerName(SERVERNAME);
		assertTrue(list.size() > 0);
		segment = dao.findByServerNameSegmentSeq(SERVERNAME, 0);
		assertNotNull(segment);
		list = dao.findLastestMetricDBs(SERVERNAME);
		assertTrue(list.size() > 0);
		segment.setEndTimestamp(System.currentTimeMillis());
		dao.update(segment);
		segment = dao.findByPostfix("continuously");
		assertTrue(segment.getRevision().startsWith("2-"));
		dao.remove(segment);
		segment = dao.findByPostfix("continuously");
		assertNull(segment);
		
		
		
	}
	private static void initConnection() throws NoSuchFieldException, SecurityException, IllegalArgumentException, IllegalAccessException{
		CouchdbStorageService service = CouchdbStorageService.getInstance();
		Field field = null;
		field = CouchdbStorageService.class.getDeclaredField("metricDBSegmentDao");
		field.setAccessible(true);
		dao = (MetricDBSegmentDAOImpl) field.get(service);
        assertNotNull(dao);

	}

}
