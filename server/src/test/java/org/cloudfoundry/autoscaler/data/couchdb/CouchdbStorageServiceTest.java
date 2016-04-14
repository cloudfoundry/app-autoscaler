package org.cloudfoundry.autoscaler.data.couchdb;

import org.apache.log4j.Logger;

import com.sun.jersey.test.framework.JerseyTest;

public class CouchdbStorageServiceTest extends JerseyTest{
	private static final Logger logger = Logger.getLogger(CouchdbStorageServiceTest.class);
	
	public CouchdbStorageServiceTest(){
		super("org.cloudfoundry.autoscaler.rest");
	}

}
