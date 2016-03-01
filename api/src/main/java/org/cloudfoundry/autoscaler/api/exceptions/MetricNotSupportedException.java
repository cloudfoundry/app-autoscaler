package org.cloudfoundry.autoscaler.api.exceptions;

public class MetricNotSupportedException  extends Exception{
	/**
	 * 
	 */
	private static final long serialVersionUID = 4759052752491394238L;
	private String metric;
	
	public MetricNotSupportedException(String id) {
		super();
		metric = id;
	}
	  
	public MetricNotSupportedException(String id, String message) {
		super(message);
		metric = id;
	}
	
	public MetricNotSupportedException(String id, Throwable cause) {
		super(cause);
		metric = id;
	}
 
	public String getAppId() {
		return metric;
	}
}
