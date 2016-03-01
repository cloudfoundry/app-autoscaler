package org.cloudfoundry.autoscaler.exceptions;

public class TriggerNotSubscribedException  extends Exception{
	/**
	 * 
	 */
	private static final long serialVersionUID = -792053836079011529L;
	private String appId;

	public TriggerNotSubscribedException(){
		super();
	}

	public TriggerNotSubscribedException(String appId){
		super();
		this.appId = appId;
	}
	public TriggerNotSubscribedException(String appId, String message){
		super(message);
	}
	
	public String getAppId(){
		return appId;
	}
}
