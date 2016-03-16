package org.cloudfoundry.autoscaler.data.couchdb.document;

import org.ektorp.support.TypeDiscriminator;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;

@JsonIgnoreProperties(ignoreUnknown = true)
@TypeDiscriminator ("doc.type=='BoundApp'")
public class BoundApp  extends TypedCouchDbDocument  {
    /**
	 * 
	 */
	private static final long serialVersionUID = 1L;
	private String appId;
    private String serviceId;
    private String appType;
    private String appName;
    private String serverName;
    
    public BoundApp() {
    	super();
    }
    
    public BoundApp(String appId, String serviceId, String appType, String appName) {
    	super();
    	this.appId = appId;
        this.serviceId = serviceId;
        this.appType = appType;
        this.appName = appName;
        
    }

   
    public String getAppId() {
        return appId;
    }

    public void setAppId(String appId) {
        this.appId = appId;
    }

    public String getServiceId() {
        return serviceId;
    }

    public void setServiceId(String serviceId) {
        this.serviceId = serviceId;
    }

    public String getAppType() {
        return appType;
    }

    public void setAppType(String appType) {
        this.appType = appType;
    }

    @Override
    public boolean equals(Object obj) {
        if (this == obj) {
            return true;
        }
        if (obj == null || !(obj instanceof BoundApp)) {
            return false;
        }
        BoundApp that = (BoundApp) obj;
        return this.appId.equals(that.appId);
    }

    @Override
    public int hashCode() {
        return appId.hashCode();
    }

    public String getAppName() {
        return appName;
    }

    public void setAppName(String appName) {
        this.appName = appName;
    }

    public String toString() {
        StringBuilder sb = new StringBuilder("BoundApp[");
        sb.append("id=").append(getId()).append(",");
        sb.append("rev=").append(getRevision()).append(",");
        sb.append("appId=").append(appId).append(",");
        sb.append("serviceId=").append(serviceId).append(",");
        sb.append("appName=").append(appName).append(",");
        sb.append("appType=").append(appType);
        sb.append("]");

        return sb.toString();
    }


	public String getServerName() {
		return serverName;
	}

	public void setServerName(String serverName) {
		this.serverName = serverName;
	}

}

