package org.cloudfoundry.autoscaler.api.validation;

import java.util.List;

public class Policy {

	private int      instanceMinCount;

	private int      instanceMaxCount;

	private List<PolicyTrigger> policyTriggers;
	
	private Schedule schedules;


	public int getInstanceMinCount() {
		return instanceMinCount;
	}

	public void setinstanceMinCount(int instanceMinCount) {
		this.instanceMinCount = instanceMinCount;
	}


	public int getInstanceMaxCount() {
		return instanceMaxCount;
	}

	public void setInstanceMaxCount(int instanceMaxCount) {
		this.instanceMaxCount = instanceMaxCount;
	}

    public List<PolicyTrigger> getPolicyTriggers() {
    	return this.policyTriggers;
    }

    public void setPolicyTriggers(List<PolicyTrigger> policyTriggers) {
    	this.policyTriggers = policyTriggers;
    }

	public Schedule getSchedules() {
		return this.schedules;
	}

	public void setSchedules(Schedule schedules) {
		this.schedules = schedules;
	}

}