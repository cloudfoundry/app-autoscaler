package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.collection;

import java.util.List;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.template.TypedCouchDbRepositorySupport;

public abstract class TypedRepoCollection<T> {

	private static final Logger logger = Logger
			.getLogger(TypedRepoCollection.class);

	public abstract List<TypedCouchDbRepositorySupport> getAllRepos();

	public abstract TypedCouchDbRepositorySupport getDefaultRepo();

	public void initAllRepos() throws Exception {
		List<TypedCouchDbRepositorySupport> repoList = getAllRepos();
		for (TypedCouchDbRepositorySupport repo : repoList)
			repo.initView();
	}

	public T get(String id) {
		try {
			return (T) this.getDefaultRepo().get(id);
		} catch (org.ektorp.DocumentNotFoundException e) {
			logger.info("org.ektorp.DocumentNotFoundException with " + id + " from " + this.getClass().getSimpleName() + " with message: " + e.getMessage());
		} catch (Exception e) {
			logger.error(e.getMessage(), e);
		}
		return null;
	}

	public void add(T entity) {
		try {
			this.getDefaultRepo().add(entity);
		} catch (org.ektorp.UpdateConflictException e) {
			logger.error("org.ektorp.UpdateConflictException for class " + entity.getClass().getSimpleName() + " with message: " + e.getMessage());
		} catch (Exception e) {
			logger.error(e.getMessage(), e);
		}
	}

	public void remove(T entity) {
		try {
			this.getDefaultRepo().remove(entity);
		} catch (Exception e) {
			logger.error(e.getMessage(), e);
		}

	}

	public void update(T entity) {
		try {
			this.getDefaultRepo().update(entity);
		} catch (org.ektorp.InvalidDocumentException e) {
			logger.error("org.ektorp.UpdateConflictException for class " + entity.getClass().getSimpleName() + " with message: " + e.getMessage());
		} catch (Exception e) {
			logger.error(e.getMessage(), e);
		}

	}

}
