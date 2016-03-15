package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.dao;

public interface CommonDAO {

	public <T> Object get(String id);

	public <T> void add(T entity);

	public <T> void remove(T entity);

	public <T> void update(T entity);

}
