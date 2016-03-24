package org.cloudfoundry.autoscaler.manager;

import org.cloudfoundry.autoscaler.bean.MonitorTriggerEvent;
import org.cloudfoundry.autoscaler.data.couchdb.document.AutoScalerPolicy;

public interface ApplicationScaleManager {
	
	public void doScaleByTrigger(MonitorTriggerEvent event) throws Exception;
	public void doScaleBySchedule(String appId,AutoScalerPolicy policy) throws Exception;
	

}
