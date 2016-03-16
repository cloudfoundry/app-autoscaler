package org.cloudfoundry.autoscaler.metric.bean;

import java.util.List;
import java.util.Map;
import java.util.Map.Entry;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.ConcurrentMap;

import org.cloudfoundry.autoscaler.bean.InstanceMetrics;
import org.cloudfoundry.autoscaler.data.couchdb.document.AppInstanceMetrics;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;

@JsonIgnoreProperties(ignoreUnknown = true)
public class ApplicationMetrics {
    private String appId;
    private String appName;
    private String appType;
    private String serviceId;
    private double memQuota;
    private long timestamp;

    //<instance index, instance metrics>
    private ConcurrentMap<Integer, InstanceMetrics> pollerMetricsMap = new ConcurrentHashMap<Integer, InstanceMetrics>();

    public ApplicationMetrics() {

    }
    
    public ApplicationMetrics shadowClone() {
        ApplicationMetrics appMetrics = new ApplicationMetrics();

        appMetrics.setAppId(getAppId());
        appMetrics.setAppName(getAppName());
        appMetrics.setAppType(getAppType());
        appMetrics.setServiceId(getServiceId());
        appMetrics.setTimestamp(getTimestamp());

        
        appMetrics.setPollerMetricsMap(getPollerMetricsMap());

        return appMetrics;
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
    
    public double getMemQuota() {
		return memQuota;
	}

	public void setMemQuota(double memQuota) {
		this.memQuota = memQuota;
	}

	

    public ConcurrentMap<Integer, InstanceMetrics> getPollerMetricsMap() {
        return pollerMetricsMap;
    }

    public void setPollerMetricsMap(ConcurrentMap<Integer, InstanceMetrics> pollerMetricsMap) {
        this.pollerMetricsMap = pollerMetricsMap;
    }

    public long getTimestamp() {
        return timestamp;
    }

    public void setTimestamp(long timestamp) {
        this.timestamp = timestamp;
    }
    
    //merge the metrics in appMetricMap for dashboard & db store
    //if the output will be stored to DB, we won't allow any staled data.
    //if the output will be display in dashboard only, we allow the staled data for some case. 
    public AppInstanceMetrics mergeToAppInstanceMetrics(boolean storeToDB, boolean staledOK) {
    	
    	//correct the invalid setting. storeToDB has higher priority than staledOK setting.
    	if (storeToDB && staledOK)
    		staledOK = false;
    	
        AppInstanceMetrics appInstMetrics = new AppInstanceMetrics();
        appInstMetrics.setAppId(this.getAppId());
        appInstMetrics.setAppName(this.getAppName());
        appInstMetrics.setAppType(this.getAppType());
        appInstMetrics.setServiceId(this.getServiceId());
        appInstMetrics.setTimestamp(this.getTimestamp());
        appInstMetrics.setMemQuota(this.getMemQuota());

        Map<Integer, InstanceMetrics> pollerMetricsMap = this.getPollerMetricsMap();
       
        
        
        List<InstanceMetrics> targetInstanceMetrics = appInstMetrics.getInstanceMetrics();
		

		// use poller metric as the criterion to merge data and store to DB when
		// necessary.
		for (Entry<Integer, InstanceMetrics> metricEntry : pollerMetricsMap.entrySet()) {
			InstanceMetrics pollerInstanceMetric = metricEntry.getValue();

			// discard this entry if pollerInstanceMetric is null;
			if (pollerInstanceMetric == null)
				continue;

			
			if (staledOK || !pollerInstanceMetric.isStored())
				targetInstanceMetrics.add(pollerInstanceMetric);
			if (storeToDB) {
				pollerInstanceMetric.setStored(true);
			}

		}
            
        

        return appInstMetrics;
    }

    

}
