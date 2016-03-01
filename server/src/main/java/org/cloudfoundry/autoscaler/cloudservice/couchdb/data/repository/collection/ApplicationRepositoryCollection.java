package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.collection;

import java.util.ArrayList;
import java.util.List;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.Application;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews.ApplicationRepository_All;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews.ApplicationRepository_ByAppId;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews.ApplicationRepository_ByBindingId;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews.ApplicationRepository_ByPolicyId;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews.ApplicationRepository_ByServiceId;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews.ApplicationRepository_ByServiceId_State;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.template.TypedCouchDbRepositorySupport;
import org.ektorp.CouchDbConnector;


public class ApplicationRepositoryCollection extends TypedRepoCollection<Application>{

	private static final Logger logger = Logger.getLogger(ApplicationRepositoryCollection.class);
	
    private ApplicationRepository_All ApplicationRepo_all;
    private ApplicationRepository_ByAppId ApplicationRepo_byAppId;
    private ApplicationRepository_ByBindingId ApplicationRepo_byBindingId;
    private ApplicationRepository_ByPolicyId ApplicationRepo_byPolicyId;
    private ApplicationRepository_ByServiceId_State ApplicationRepo_byServiceId_State;
    private ApplicationRepository_ByServiceId ApplicationRepo_byServiceId;

    public ApplicationRepositoryCollection(CouchDbConnector db) {
    	ApplicationRepo_all = new ApplicationRepository_All(db);
    	ApplicationRepo_byAppId =new ApplicationRepository_ByAppId(db);
    	ApplicationRepo_byBindingId = new ApplicationRepository_ByBindingId(db);
    	ApplicationRepo_byPolicyId = new ApplicationRepository_ByPolicyId(db);
    	ApplicationRepo_byServiceId_State = new ApplicationRepository_ByServiceId_State(db);
    	ApplicationRepo_byServiceId = new ApplicationRepository_ByServiceId(db);
    }
    
    public ApplicationRepositoryCollection(CouchDbConnector db, boolean initDesignDocument) {
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
    	repoList.add(ApplicationRepo_all);
    	repoList.add(ApplicationRepo_byAppId);
    	repoList.add(ApplicationRepo_byBindingId);
    	repoList.add(ApplicationRepo_byPolicyId);
    	repoList.add(ApplicationRepo_byServiceId_State);
    	repoList.add(ApplicationRepo_byServiceId);
    	return repoList;
    	
    }
    
	@Override
	public TypedCouchDbRepositorySupport getDefaultRepo() {
		return ApplicationRepo_all;
	}

	public List<Application> findByServiceId(String serviceId){
		return ApplicationRepo_byServiceId.findByServiceId(serviceId);
	}

	public List<Application> findByServiceIdAndState(String serviceId){
		return ApplicationRepo_byServiceId_State.findByServiceIdAndState(serviceId);
	}

	public List<Application> findByPolicyId(String policyId){
		return ApplicationRepo_byPolicyId.findByPolicyId(policyId);
	}

	public Application findByAppId(String appId){
		return ApplicationRepo_byAppId.findByAppId(appId);
		
	}

	public Application findByBindingId(String bindingId){
		return ApplicationRepo_byBindingId.findByBindingId(bindingId);
	}

	
	public List<Application> findDupliateByAppId(String appId){
		return ApplicationRepo_byAppId.findDupliateByAppId(appId);
	}
	

	public List<Application> getAllRecords() {
		try {
			return this.ApplicationRepo_all.getAllRecords();
		} catch (Exception e) {
			logger.error(e.getMessage(), e);
		}
		return null;
	}
	
	public void removeDuplicateByAppId(String appId){
		List<Application> apps = findDupliateByAppId(appId);
		if (apps == null || apps.size() <= 1){
			return;
		}
		for (int i = 1 ; i< apps.size(); i++)
			ApplicationRepo_all.remove(apps.get(i));
	}
	
	public void updateStateByAppId (String appId, String state){
        Application app = findByAppId(appId);
        app.setState(state);
        ApplicationRepo_all.update(app);
	}
	
	
	public void updatePolicyStateByAppId (String appId, String state){
        Application app = findByAppId(appId);
        app.setPolicyState(state);
        ApplicationRepo_all.update(app);
	}    



    
}

