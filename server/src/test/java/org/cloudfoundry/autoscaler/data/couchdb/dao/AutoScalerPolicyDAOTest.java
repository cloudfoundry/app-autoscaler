package org.cloudfoundry.autoscaler.data.couchdb.dao;

import java.lang.reflect.Field;
import java.util.List;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.data.couchdb.CouchdbStorageService;
import org.cloudfoundry.autoscaler.data.couchdb.dao.impl.AutoScalerPolicyDAOImpl;
import org.cloudfoundry.autoscaler.data.couchdb.document.AutoScalerPolicy;
import org.cloudfoundry.autoscaler.exceptions.PolicyNotFoundException;
import org.cloudfoundry.autoscaler.test.testcase.base.JerseyTestBase;

import static org.cloudfoundry.autoscaler.test.constant.Constants.*;
import static org.junit.Assert.*;

import org.junit.Test;


public class AutoScalerPolicyDAOTest extends JerseyTestBase{
	
	private static final Logger logger = Logger.getLogger(AutoScalerPolicyDAOTest.class);
	private static AutoScalerPolicyDAO dao = null;
	public AutoScalerPolicyDAOTest()throws Exception{
		super();
		initConnection();
	}
	@Test
	public void autoScalerPolicyDAOTest() throws PolicyNotFoundException{
		
		dao.get(AUTOSCALERPOLICYDOCTESTID);
		assertTrue(!logContains(DOCUMENTNOTFOUNDERRORMSG));
		dao.tryGet(AUTOSCALERPOLICYDOCTESTID + "TMP");
		assertTrue(!logContains(DOCUMENTNOTFOUNDERRORMSG));
		dao.get(AUTOSCALERPOLICYDOCTESTID + "TMP");
		assertTrue(logContains(DOCUMENTNOTFOUNDERRORMSG));
		
		List<AutoScalerPolicy> list = dao.findAll();
		assertTrue(list.size() > 0);
		AutoScalerPolicy policy= dao.findByPolicyId(TESTPOLICYID);
		assertNotNull(policy);
		policy.setInstanceMaxCount(4);
		dao.update(policy);
		policy= dao.findByPolicyId(TESTPOLICYID);
		assertTrue(policy.getRevision().startsWith("2-"));
//		dao.remove(policy);
//		policy= dao.findByPolicyId(TESTPOLICYID);
//		assertNull(policy);
		
		}
	private static void initConnection() throws NoSuchFieldException, SecurityException, IllegalArgumentException, IllegalAccessException{
		CouchdbStorageService service = CouchdbStorageService.getInstance();
		Field field = null;
		field = CouchdbStorageService.class.getDeclaredField("autoScalerPolicyDao");
		field.setAccessible(true);
		dao = (AutoScalerPolicyDAOImpl) field.get(service);
        assertNotNull(dao);

	}

}
