package org.cloudfoundry.autoscaler.data.couchdb.connection.manager;

import org.apache.log4j.Logger;
import org.ektorp.CouchDbConnector;
import org.ektorp.CouchDbInstance;
import org.ektorp.http.StdHttpClient;
import org.ektorp.http.StdHttpClient.Builder;
import org.ektorp.impl.StdCouchDbInstance;


public class CouchDbConnectionManager {


    private static final Logger logger = Logger.getLogger(CouchDbConnectionManager.class);
    private CouchDbConnector db;
    private CouchDbInstance dbInstance; 
    
    public CouchDbConnectionManager(String dbName, String userName, String password, String host, int port, boolean enableSSL,  int timeout)
    {
    	db = null;
        try
        {
       		db = initConnection(dbName, userName, password, host, port, enableSSL, timeout);
        }
        catch(Exception e)
        {
            logger.error(e.getMessage(),e);
        }
    }

    public CouchDbConnector getDb()
    {
        return db;
    }

    public void setDb(CouchDbConnector db)
    {
        this.db = db;
    }

    private CouchDbConnector initConnection(String dbName, String userName, String password, String host, int port, boolean enableSSL, int timeout)
    {
		Builder builder = new StdHttpClient.Builder()
		.host(host)
		.port(port)
		.connectionTimeout(timeout)
		.socketTimeout(timeout)
		.enableSSL(enableSSL); 

        
        if(userName != null && !userName.isEmpty() && password != null && !password.isEmpty())
            builder.username(userName).password(password);
        
        dbInstance = new StdCouchDbInstance(builder.build());
        if (!dbInstance.checkIfDbExists(dbName))
        	dbInstance.createDatabase(dbName);
        CouchDbConnector couchDB = dbInstance.createConnector(dbName, true);   
        return couchDB;


    }
	
    
    public boolean deleteDB(String dbName)
    {
    	if (dbInstance != null && dbInstance.checkIfDbExists(dbName)){
    			dbInstance.deleteDatabase(dbName);
    			return true;
    	}
    	else 
    		return false;
    }
    
}
