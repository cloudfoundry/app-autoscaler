package org.cloudfoundry.autoscaler.data.couchdb;

import org.cloudfoundry.autoscaler.data.AutoScalingDataStore;

public class AutoScalingDataStoreFactory {

	public static AutoScalingDataStore getAutoScalingDataStore() {
		return CouchdbStorageService.getInstance();
	}

}
