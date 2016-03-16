package org.cloudfoundry.autoscaler.data.couchdb.document;

import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.UUID;

import org.cloudfoundry.autoscaler.bean.AutoScalerPolicyTrigger;
import org.ektorp.support.TypeDiscriminator;

import com.fasterxml.jackson.annotation.JsonIgnore;
import com.fasterxml.jackson.annotation.JsonIgnoreProperties;


@JsonIgnoreProperties(ignoreUnknown = true)
@TypeDiscriminator ("doc.type=='AutoScalerPolicy'")
public class AutoScalerPolicy  extends TypedCouchDbDocument {

	/**
	 * 
	 */
	private static final long serialVersionUID = 2279984704134465137L;
	
	public final static String STATE_ENABLED = "enabled";
	public final static String STATE_DISABLED = "disabled";

//	@JsonProperty("_id")
	private String            policyId;
	private String            policyName;
	private int      instanceMinCount      = 1;
	private int      instanceMaxCount      = 10;
	private String timezone;

	private List<AutoScalerPolicyTrigger> policyTriggers;
	private String orgId;
	private String spaceId;
	
	private String currentScheduledPolicyId;
	private Map<String, ScheduledPolicy> scheduledPolicies = new HashMap<String, ScheduledPolicy>();
	
	public AutoScalerPolicy(){
		super();
	}
	
	public AutoScalerPolicy(String name, List<AutoScalerPolicyTrigger> triggers)
	{
		super();
		policyName   = name;
		policyTriggers = triggers;
	}

	public String getPolicyName() {
		return policyName;
	}

	public void setPolicyName(String configId) {
		this.policyName = configId;
	}

	public List<AutoScalerPolicyTrigger> getPolicyTriggers() {
		return policyTriggers;
	}

	public void setPolicyTriggers(List<AutoScalerPolicyTrigger> triggers) {
		this.policyTriggers = triggers;
	}

	public String getPolicyId() {
		return policyId;
	}

	public void setPolicyId(String policyId) {
		this.policyId = policyId;
	}

	public int getInstanceMinCount() {
		return instanceMinCount;
	}

	public void setInstanceMinCount(int instanceMinCount) {
		this.instanceMinCount = instanceMinCount;
	}

	public int getInstanceMaxCount() {
		return instanceMaxCount;
	}

	public void setInstanceMaxCount(int instanceMaxCount) {
		this.instanceMaxCount = instanceMaxCount;
	}

	public String getOrgId() {
		return orgId;
	}

	public void setOrgId(String orgId) {
		this.orgId = orgId;
	}

	public String getSpaceId() {
		return spaceId;
	}

	public void setSpaceId(String spaceId) {
		this.spaceId = spaceId;
	}
	
	public String getCurrentScheduledPolicyId() {
		return currentScheduledPolicyId;
	}

	public void setCurrentScheduledPolicyId(String currentScheduledPolicyId) {
		this.currentScheduledPolicyId = currentScheduledPolicyId;
	}

	public Map<String, ScheduledPolicy> getScheduledPolicies() {
		return scheduledPolicies;
	}

	public void setScheduledPolicies(Map<String, ScheduledPolicy> scheduledPolicies) {
		this.scheduledPolicies = scheduledPolicies;
	}
	
	public void addScheduledPolicies(ScheduledPolicy scheduledPolicy) {
		String key  = UUID.randomUUID().toString();
		this.scheduledPolicies.put(key, scheduledPolicy);
	}
	@JsonIgnore
    
	public String getCurrentScheduleType() {
		if (currentScheduledPolicyId == null) {
			return null;
		}
		return scheduledPolicies.get(currentScheduledPolicyId).getType();
	}
	@JsonIgnore
    
	public String getCurrentScheduleStartTime() {
		if (currentScheduledPolicyId == null) {
			return null;
		}
		return scheduledPolicies.get(currentScheduledPolicyId).getStartTime();
	}
	@JsonIgnore
    
	public String getCurrentScheduleRepeatCycle() {
		if (currentScheduledPolicyId == null) {
			return null;
		}
		return scheduledPolicies.get(currentScheduledPolicyId).getRepeatCycle();
	}
    @JsonIgnore
    
	public int getCurrentInstanceMinCount() {
		if (currentScheduledPolicyId == null) {
			return instanceMinCount;
		}
		return scheduledPolicies.get(currentScheduledPolicyId).getInstanceMinCount();
	}
	
    @JsonIgnore
    
    public int getCurrentInstanceMaxCount() {
		if (currentScheduledPolicyId == null) {
			return instanceMaxCount;
		}
		return scheduledPolicies.get(currentScheduledPolicyId).getInstanceMaxCount();
	}

	public String getTimezone() {
		return timezone;
	}

	public void setTimezone(String timezone) {
		this.timezone = timezone;
	}
	
}
