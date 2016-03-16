package org.cloudfoundry.autoscaler.metric.bean;

import java.io.Serializable;
import java.util.LinkedList;
import java.util.List;

import org.cloudfoundry.autoscaler.bean.InstanceMetrics;
import org.cloudfoundry.autoscaler.data.couchdb.document.AppInstanceMetrics;

import com.fasterxml.jackson.annotation.JsonIgnore;
import com.fasterxml.jackson.annotation.JsonIgnoreProperties;

@JsonIgnoreProperties(ignoreUnknown = true)
//@JsonIgnoreProperties(value = { "appId", "appName", "appType", "serviceId" })  
public class AppMetricPerInstance extends InstanceMetrics  implements Serializable{
    
    /**
	 * 
	 */
	private static final long serialVersionUID = 1L;

	@JsonIgnore
    
    private String appId;

    @JsonIgnore
    
    private String appName;

    @JsonIgnore
    
    private String appType;

    @JsonIgnore
    
    private String serviceId;

    public AppMetricPerInstance() {

    }

    public String getAppId() {
        return appId;
    }

    public void setAppId(String appId) {
        this.appId = appId;
    }

    public String getAppName() {
        return appName;
    }

    public void setAppName(String appName) {
        this.appName = appName;
    }

    public String getAppType() {
        return appType;
    }

    public void setAppType(String appType) {
        this.appType = appType;
    }

    public String getServiceId() {
        return serviceId;
    }

    public void setServiceId(String serviceId) {
        this.serviceId = serviceId;
    }
    
  

	public AppInstanceMetrics toAppInstanceMetrics(long timestamp){

    	
    	List<InstanceMetrics> instanceMetricsList = new LinkedList<InstanceMetrics>();
    	instanceMetricsList.add((InstanceMetrics)this);
    	
    	AppInstanceMetrics appInstanceMetric = new AppInstanceMetrics();
    	appInstanceMetric.setAppId(this.getAppId());
    	appInstanceMetric.setAppName(this.getAppName());
    	appInstanceMetric.setAppType(this.getAppType());
    	appInstanceMetric.setServiceId(this.getServiceId());
    	appInstanceMetric.setTimestamp(timestamp);
    	appInstanceMetric.setInstanceMetrics(instanceMetricsList);
    	
    	return appInstanceMetric;
    }
}
