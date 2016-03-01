package org.cloudfoundry.autoscaler.metric.exceptions;

public class DataStoreException extends Exception {
	/**
	 * 
	 */
	private static final long serialVersionUID = 7055233976174285115L;

	public DataStoreException (String message){
		super(message);		
	}
	
	public DataStoreException (String message, Throwable cause){
		super(message,cause);		
	}
	
	public DataStoreException (Throwable cause){
		super(cause);		
	}
}
