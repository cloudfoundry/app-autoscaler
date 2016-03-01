package org.cloudfoundry.autoscaler.servicebroker.data.entity.dao.couchdb;

import org.cloudfoundry.autoscaler.servicebroker.data.entity.dao.CommonDAO;

public abstract class CommonDAOImpl implements CommonDAO{
	
	public abstract <T> TypedCouchdbRepositorySupport<T> getDefaultRepo();

	@Override
	public <T> Object get(String id) {
		try {
			return this.getDefaultRepo().get(id);
		} catch (org.ektorp.DocumentNotFoundException e) {
			return null;
		}
		
	}

	@Override
	public <T> void add(T entity) {
		this.getDefaultRepo().add(entity);
	}

	@Override
	public <T> void remove(T entity) {
		this.getDefaultRepo().remove(entity);
	}


	@Override
	public <T> void update(T entity) {
		this.getDefaultRepo().update(entity);
		
	}

 


}	
