package org.cloudfoundry.autoscaler.data.couchdb.document;

import org.cloudfoundry.autoscaler.bean.Trigger;
import org.ektorp.support.TypeDiscriminator;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;


@JsonIgnoreProperties(ignoreUnknown = true)
@TypeDiscriminator ("doc.type=='TriggerRecord'")
public class TriggerRecord extends TypedCouchDbDocument {

    /**
	 * 
	 */
	private static final long serialVersionUID = 1L;

	@JsonProperty("_id")
    private String id;

    private String appName;
    private String appId;
    private Trigger trigger;
    private String serverName; //The server that evaluate this trigger

    public TriggerRecord() {
    	super();
    }

    public TriggerRecord(String appName, Trigger trigger) {
    	super();
    	this.appName = appName;
        this.appId = trigger.getAppId();
        this.trigger = trigger;
        this.id = this.getAppId() + "_" + trigger.getMetric() + "_" + trigger.getTriggerId() + "_"
                + trigger.getMetricThreshold();
    }

	@Override
    public String toString() {
        StringBuilder sb = new StringBuilder("TriggerRecord[");
        sb.append("id:").append(id);
        sb.append(", appName:").append(appName);
        sb.append(", appId:").append(appId);
        sb.append(", trigger:").append(trigger);
        sb.append("]");

        return sb.toString();
    }

    public String getAppName() {
        return appName;
    }

    public void setAppName(String appName) {
        this.appName = appName;
    }

    public Trigger getTrigger() {
        return trigger;
    }

    public void setTrigger(Trigger trigger) {
        this.trigger = trigger;
    }

    public String getId() {
        return id;
    }

    public void setId(String id) {
        this.id = id;
    }

    public String getAppId() {
        return appId;
    }

    public void setAppId(String appId) {
        this.appId = appId;
    }

	public String getServerName() {
		return serverName;
	}

	public void setServerName(String serverName) {
		this.serverName = serverName;
	}

    
}
