package org.cloudfoundry.autoscaler.data.couchdb.dao;

import java.lang.reflect.Field;
import java.util.List;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.data.couchdb.CouchdbStorageService;
import org.cloudfoundry.autoscaler.data.couchdb.dao.impl.MetricDBSegmentDAOImpl;
import org.cloudfoundry.autoscaler.data.couchdb.document.MetricDBSegment;
import org.cloudfoundry.autoscaler.test.testcase.base.JerseyTestBase;

import static org.cloudfoundry.autoscaler.test.constant.Constants.*;
import static org.junit.Assert.*;

import org.junit.Test;


public class MetricDBSegmentDAOTest extends JerseyTestBase{
	
	private static final Logger logger = Logger.getLogger(MetricDBSegmentDAOTest.class);
	private static MetricDBSegmentDAO dao = null;
	public MetricDBSegmentDAOTest()throws Exception{
		super();
		initConnection();
	}
	@Test
	public void appAutoScaleStateDAOTest(){
		
		dao.get(METRICDBSEGMENTDOCTESTID);
		assertTrue(!logContains(DOCUMENTNOTFOUNDERRORMSG));
		dao.tryGet(METRICDBSEGMENTDOCTESTID + "TMP");
		assertTrue(!logContains(DOCUMENTNOTFOUNDERRORMSG));
		dao.get(METRICDBSEGMENTDOCTESTID + "TMP");
		assertTrue(logContains(DOCUMENTNOTFOUNDERRORMSG));
		
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
