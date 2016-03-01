package org.cloudfoundry.autoscaler.statestore;

public class AutoScalingDataStoreFactory {

	public static AutoScalingDataStore getAutoScalingDataStore() {
		return CouchdbStorageService.getInstance();
	}

}
