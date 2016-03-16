package org.cloudfoundry.autoscaler.manager;

import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.CopyOnWriteArrayList;

import org.cloudfoundry.autoscaler.data.AutoScalingDataStore;
import org.cloudfoundry.autoscaler.data.couchdb.AutoScalingDataStoreFactory;
import org.cloudfoundry.autoscaler.data.couchdb.document.AutoScalerPolicy;
import org.cloudfoundry.autoscaler.exceptions.DataStoreException;
import org.cloudfoundry.autoscaler.exceptions.PolicyNotFoundException;

public class PolicyManagerImpl implements PolicyManager{
	
	private static PolicyManagerImpl instance= new PolicyManagerImpl();
	
	private ConcurrentHashMap<String, AutoScalerPolicy> policyCache = new ConcurrentHashMap<String, AutoScalerPolicy>();
	List<AutoScalerPolicy> monitoredCache = new CopyOnWriteArrayList<AutoScalerPolicy>();
	
	public static PolicyManagerImpl getInstance(){
		return instance;
	}
	private PolicyManagerImpl(){
		
	}
	
	@Override
	public AutoScalerPolicy getPolicyById(String policyId)  throws PolicyNotFoundException, DataStoreException{
		AutoScalerPolicy policy = null;
		if (policyCache.containsKey(policyId)){
			policy= policyCache.get(policyId); 
		} else {
			AutoScalingDataStore dataStore = AutoScalingDataStoreFactory.getAutoScalingDataStore();
			policy = dataStore.getPolicyById(policyId);
			policyCache.put(policyId, policy);
		}
		return policy;
	}

	@Override
	public String createPolicy(AutoScalerPolicy policy) throws DataStoreException {
		AutoScalingDataStore dataStore = AutoScalingDataStoreFactory.getAutoScalingDataStore();
		String newCreatedId = dataStore.savePolicy(policy);
		policyCache.put(policy.getPolicyId(), policy);
		if (policy.getScheduledPolicies() != null
				&& !policy.getScheduledPolicies().isEmpty()) {
			monitoredCache.add(policy);
		}
		return newCreatedId;
	}
	
	@Override
	public void updatePolicy(AutoScalerPolicy policy) throws DataStoreException {
		AutoScalingDataStore dataStore = AutoScalingDataStoreFactory.getAutoScalingDataStore();
		dataStore.savePolicy(policy);
		policyCache.put(policy.getPolicyId(), policy);
		
		removeCachedPolicy(policy.getPolicyId());
		if (policy.getScheduledPolicies() != null
				&& !policy.getScheduledPolicies().isEmpty()) {
			monitoredCache.add(policy);
		}
	}

	@Override
	public void deletePolicy(String policyId) throws DataStoreException, PolicyNotFoundException {
		AutoScalingDataStore dataStore = AutoScalingDataStoreFactory.getAutoScalingDataStore();
		dataStore.deletePolicy(policyId);
		policyCache.remove(policyId);
		removeCachedPolicy(policyId);
	}
	
	@Override
	public void recoverMonitoredCache() throws Exception {
    	AutoScalingDataStore dataStore = AutoScalingDataStoreFactory.getAutoScalingDataStore();
    	List<AutoScalerPolicy> scalingPolicies = dataStore.getAutoScalerPolicies();
    	List<AutoScalerPolicy> filteredScalingPolicies = new ArrayList<AutoScalerPolicy>();
		if (scalingPolicies != null) {
			for(AutoScalerPolicy policy : scalingPolicies) {
				//we can improve this based on policy states 
				//also if the scheduled policy is only specific day and all in the past
				if(!policy.getScheduledPolicies().isEmpty()) {
					filteredScalingPolicies.add(policy);
				}
			}
			monitoredCache.addAll(filteredScalingPolicies);
		}
    }

	private void removeCachedPolicy(String policyId) {
		AutoScalerPolicy toRemoved = null;
		for (AutoScalerPolicy policy : monitoredCache) {
			if (policy.getPolicyId().equals(policyId)) {
				toRemoved = policy;
			}
		}
		if (toRemoved != null) {
			monitoredCache.remove(toRemoved);
		}
	}
	
	@Override
	public List<AutoScalerPolicy> getMonitoredCache() {
		return monitoredCache;
	}

	
	@Override
	public void invalidateCache() {
		policyCache.clear();
	}
}
