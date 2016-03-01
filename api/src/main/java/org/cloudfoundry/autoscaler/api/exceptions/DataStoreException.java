package org.cloudfoundry.autoscaler.api.exceptions;

public class DataStoreException extends Exception {
	/**
	 * 
	 */
	private static final long serialVersionUID = -3300617382722836097L;

	/**
	 * 
	 */

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
