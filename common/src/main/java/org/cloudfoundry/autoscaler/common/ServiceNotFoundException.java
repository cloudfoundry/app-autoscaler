package org.cloudfoundry.autoscaler.common;

public class ServiceNotFoundException extends Exception{
	private static final long serialVersionUID = 1L;

	private String serviceName;
	
	private String appId;
	
	public ServiceNotFoundException(String serviceName, String appId) {
		super();
		this.serviceName = serviceName;
		this.appId = appId;
	}
	  
	public ServiceNotFoundException(String serviceName, String appId, String message) {
		super(message);
		this.serviceName = serviceName;
		this.appId = appId;
	}
	
	public ServiceNotFoundException(String serviceName, String appId, Throwable cause) {
		super(cause);
		this.serviceName = serviceName;
		this.appId = appId;
	}
 
	public String getServiceName() {
		return this.serviceName;
	}

	public String getAppId() {
		return this.appId;
	}
}
