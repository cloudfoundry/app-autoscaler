package org.cloudfoundry.autoscaler.data.couchdb.dao.impl;

import java.util.List;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.data.couchdb.dao.CommonDAO;
import org.cloudfoundry.autoscaler.data.couchdb.dao.base.TypedCouchDbRepositorySupport;
import org.ektorp.CouchDbConnector;

public abstract class CommonDAOImpl implements CommonDAO {
	private static final Logger logger = Logger.getLogger(CommonDAOImpl.class);

	public CommonDAOImpl() {

	}

	public CommonDAOImpl(CouchDbConnector db, boolean initDesignDocument) {
		if (initDesignDocument)
			try {
				initAllRepos();
			} catch (Exception e) {
				logger.error(e.getMessage(), e);
			}

	}

	public abstract <T> TypedCouchDbRepositorySupport<T> getDefaultRepo();

	public abstract <T> List<TypedCouchDbRepositorySupport<T>> getAllRepos();

	public <T> void initAllRepos() throws Exception {
		List<TypedCouchDbRepositorySupport<T>> repoList = getAllRepos();
		for (TypedCouchDbRepositorySupport<T> repo : repoList)
			repo.initView();
	}

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
