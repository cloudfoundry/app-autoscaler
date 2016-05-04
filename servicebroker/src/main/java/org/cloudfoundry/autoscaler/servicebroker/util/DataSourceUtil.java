package org.cloudfoundry.autoscaler.servicebroker.util;

import org.cloudfoundry.autoscaler.common.util.ConfigManager;
import org.cloudfoundry.autoscaler.servicebroker.Constants;
import org.cloudfoundry.autoscaler.servicebroker.data.storeservice.IDataStoreService;
import org.cloudfoundry.autoscaler.servicebroker.data.storeservice.couchdb.CouchdbStoreService;



public class DataSourceUtil {
	
    private static IDataStoreService storeProvider = null;    
    
    private DataSourceUtil() {
    }

    
    public static IDataStoreService getStoreProvider() throws Exception{
		if ( storeProvider == null)
			setStoreProvider(ConfigManager.get(Constants.CONFIG_ENTRY_DATASTORE_PROVIDER, Constants.CONFIG_ENTRY_DATASTORE_PROVIDER_COUCHDB));
    	return storeProvider;
	}

	public static void setStoreProvider(String provider) throws Exception{
		IDataStoreService storage = null;
		if (provider.equalsIgnoreCase(Constants.CONFIG_ENTRY_DATASTORE_PROVIDER_COUCHDB)) {
			storage = CouchdbStoreService.getInstance();
		} else {
			storage = CouchdbStoreService.getInstance();
		}    	
		storeProvider = storage;
	}
	
	
}
