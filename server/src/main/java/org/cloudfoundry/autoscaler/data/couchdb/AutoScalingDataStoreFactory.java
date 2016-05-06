package org.cloudfoundry.autoscaler.data.couchdb;

import org.cloudfoundry.autoscaler.data.AutoScalingDataStore;

public class AutoScalingDataStoreFactory {

	public static void InitializeAutoScalingDataStore() throws Exception{
		CouchdbStorageService.Initialize();
	}
	
	public static AutoScalingDataStore getAutoScalingDataStore() {
		return CouchdbStorageService.getInstance();
	}

}
