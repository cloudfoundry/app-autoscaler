package org.cloudfoundry.autoscaler.cloudservice.statestore;


public interface IStateStore
{
	
	public boolean  exists(String key);
	
	public Object   get(String key);
	
	public Object   put(String key, Object value);
	
	public Object   remove(String key);
	
}
