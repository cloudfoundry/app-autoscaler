package org.cloudfoundry.autoscaler.data.couchdb.document;

import java.io.Serializable;
import java.util.LinkedList;
import java.util.List;

import org.cloudfoundry.autoscaler.bean.InstanceMetrics;
import org.ektorp.support.TypeDiscriminator;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.databind.annotation.JsonSerialize;

@JsonIgnoreProperties(ignoreUnknown = true)
@JsonSerialize(include = JsonSerialize.Inclusion.NON_NULL)
@TypeDiscriminator ("doc.type=='AppInstanceMetrics'")
public class AppInstanceMetrics extends TypedCouchDbDocument implements Serializable, Cloneable{
    /**
	 * 
	 */
	private static final long serialVersionUID = 1L;
	private String appId;
    private String appName;
    private String appType;
    private String serviceId;
    private long timestamp;
    private double memQuota;
    private List<InstanceMetrics> instanceMetrics = new LinkedList<InstanceMetrics>();

    public AppInstanceMetrics() {
    	super();
    }

    
    public AppInstanceMetrics(String appId, String appName, String appType,
			String serviceId, long timestamp,
			List<InstanceMetrics> instanceMetrics) {
		super();
		this.appId = appId;
		this.appName = appName;
		this.appType = appType;
		this.serviceId = serviceId;
		this.timestamp = timestamp;
		this.instanceMetrics = instanceMetrics;

	}


	public String getAppId() {
        return appId;
    }

    public String getAppName() {
        return appName;
    }

    public String getAppType() {
        return appType;
    }

    public List<InstanceMetrics> getInstanceMetrics() {
        return instanceMetrics;
    }

    public String getServiceId() {
        return serviceId;
    }

    public long getTimestamp() {
        return timestamp;
    }

    public void setAppId(String appId) {
        this.appId = appId;
    }

    public void setAppName(String appName) {
        this.appName = appName;
    }

    public void setAppType(String appType) {
        this.appType = appType;
    }

    public void setInstanceMetrics(List<InstanceMetrics> instanceMetrics) {
        this.instanceMetrics = instanceMetrics;
    }

    public void setServiceId(String serviceId) {
        this.serviceId = serviceId;
    }

    public void setTimestamp(long timestamp) {
        this.timestamp = timestamp;
    }

    
    public double getMemQuota() {
		return memQuota;
	}

	public void setMemQuota(double memQuota) {
		this.memQuota = memQuota;
	}


	public void addInstanceMetrics(InstanceMetrics instMetrics) {
        this.instanceMetrics.add(instMetrics);
    }

    public AppInstanceMetrics shadowClone() {
        try {
        	AppInstanceMetrics cloned = (AppInstanceMetrics) super.clone();
            return cloned;
        } catch (CloneNotSupportedException e) {
            throw new RuntimeException(e);
        }
    }    
    
    @SuppressWarnings({ "rawtypes", "unchecked" })
	public AppInstanceMetrics deepClone() {
        try {
        	AppInstanceMetrics cloned = (AppInstanceMetrics) super.clone();
        	List clonedInstanceMetrics = new LinkedList<InstanceMetrics>();
            for ( InstanceMetrics instanceMetric : this.instanceMetrics ){
            	clonedInstanceMetrics.add(instanceMetric.clone());
            }
            cloned.setInstanceMetrics(clonedInstanceMetrics);
            return cloned;
        } catch (CloneNotSupportedException e) {
            throw new RuntimeException(e);
        }
    }   

}
