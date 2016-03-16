package org.cloudfoundry.autoscaler.data.couchdb.document;

import org.ektorp.support.TypeDiscriminator;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;


@JsonIgnoreProperties(ignoreUnknown = true)
@TypeDiscriminator ("doc.type=='AppAutoScaleState'")
public class AppAutoScaleState extends TypedCouchDbDocument
{
	/**
	 * 
	 */
	private static final long serialVersionUID = 8096435938474246095L;
	private String  appId;
	private int     instanceCountState;
	private String  lastActionTriggerId;
	private int     lastActionInstanceTarget;
	private long    instanceStepCoolDownEndTime;
	private long lastActionStartTime;
	private long lastActionEndTime;
	private String errorCode;// the error code of failed scaling
	private String historyId; //history id of last scaling action
	private ScalingHistory scaleEvent;

	public AppAutoScaleState(){
		super();
	}
	
	public AppAutoScaleState(String id, int state) {
		super();
		appId                       = id;
		instanceCountState          = state;
		lastActionTriggerId         = null;
		lastActionInstanceTarget    = 0;
		instanceStepCoolDownEndTime = 0;
		lastActionEndTime = 0;
	}
	
	public AppAutoScaleState(String apId){
		appId = apId;
	}
	

	
	public String getAppId() {
		return appId;
	}

	public void setAppId(String appId) {
		this.appId = appId;
	}

	public int getInstanceCountState() {
		return instanceCountState;
	}

	public void setInstanceCountState(int instanceCountState) {
		this.instanceCountState = instanceCountState;
	}

	public String getLastActionTriggerId() {
		return lastActionTriggerId;
	}

	public void setLastActionTriggerId(String lastActionTriggerId) {
		this.lastActionTriggerId = lastActionTriggerId;
	}

	public int getLastActionInstanceTarget() {
		return lastActionInstanceTarget;
	}

	public void setLastActionInstanceTarget(int lastActionInstanceTarget) {
		this.lastActionInstanceTarget = lastActionInstanceTarget;
	}

	public long getInstanceStepCoolDownEndTime() {
		return instanceStepCoolDownEndTime;
	}

	public void setInstanceStepCoolDownEndTime(long instanceStepCoolDownEndTime) {
		this.instanceStepCoolDownEndTime = instanceStepCoolDownEndTime;
	}

	public long getLastActionStartTime() {
		return lastActionStartTime;
	}

	public void setLastActionStartTime(long lastActionStartTime) {
		this.lastActionStartTime = lastActionStartTime;
	}

	public long getLastActionEndTime() {
		return lastActionEndTime;
	}

	public void setLastActionEndTime(long lastActionEndTime) {
		this.lastActionEndTime = lastActionEndTime;
	}

	public ScalingHistory getScaleEvent() {
		return scaleEvent;
	}

	public void setScaleEvent(ScalingHistory scaleEvent) {
		this.scaleEvent = scaleEvent;
	}

	public String getErrorCode() {
		return errorCode;
	}

	public void setErrorCode(String errorCode) {
		this.errorCode = errorCode;
	}

	public String getHistoryId() {
		return historyId;
	}

	public void setHistoryId(String historyId) {
		this.historyId = historyId;
	}
}

