package org.cloudfoundry.autoscaler.data.couchdb.dao.base;

import java.util.HashSet;
import java.util.Set;


public class CouchDBConnectionProfile {

	private static CouchDBConnectionProfile instance;
	private static int connectionCount = 0;
	private static Set<String> uuidSet = new HashSet<String>();
	
	
	public static synchronized CouchDBConnectionProfile getInstance() {
        if (instance == null) {
            instance = new CouchDBConnectionProfile();
        }
        return instance;
    }
	
    public int getConnectionCount(){
    	return connectionCount;
    }
    
    public Set<String> getConntionUUID(){
    	return uuidSet;
    }
    
    public void increaseConnectionCount(String uuid){
    	connectionCount++;
    	uuidSet.add(uuid);
    }
	
    public void decreaseConnectionCount(String uuid){
    	connectionCount--;
    	uuidSet.remove(uuid);
    }

    

	
	
}
