package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.collection;

import java.util.ArrayList;
import java.util.List;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.ServiceConfig;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews.ConfigRepository_All;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.template.TypedCouchDbRepositorySupport;
import org.ektorp.CouchDbConnector;


public class ConfigRepositoryCollection extends TypedRepoCollection<ServiceConfig>{

	private static final Logger logger = Logger.getLogger(ConfigRepositoryCollection.class);
	
    private ConfigRepository_All configRepo_all;
    

    public ConfigRepositoryCollection(CouchDbConnector db) {
    	configRepo_all = new ConfigRepository_All(db);

    }
    
    public ConfigRepositoryCollection(CouchDbConnector db, boolean initDesignDocument) {
    	this(db);
    	if (initDesignDocument)
			try {
				initAllRepos();
			} catch (Exception e) {
 				logger.error(e.getMessage(), e);
			}
    }
    
    @Override
    public List<TypedCouchDbRepositorySupport> getAllRepos(){
    	List<TypedCouchDbRepositorySupport> repoList = new ArrayList<TypedCouchDbRepositorySupport>();
 
    	repoList.add(configRepo_all);
    	return repoList;
    	
    }
    
	@Override
	public TypedCouchDbRepositorySupport getDefaultRepo() {
		return configRepo_all;
	}
    
	public List<ServiceConfig> getAllRecords() {
		try {
			return this.configRepo_all.getAllRecords();
		} catch (Exception e) {
			logger.error(e.getMessage(), e);
		}
		return null;
	}

    
}
