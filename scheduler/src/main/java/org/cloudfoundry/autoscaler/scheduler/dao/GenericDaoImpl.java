package org.cloudfoundry.autoscaler.scheduler.dao;

import javax.persistence.EntityManager;
import javax.persistence.PersistenceContext;

import org.cloudfoundry.autoscaler.scheduler.util.error.DatabaseValidationException;

/**
 * 
 *
 * @param <T>
 */
public class GenericDaoImpl<T> implements GenericDao<T> {

	@PersistenceContext
	EntityManager entityManager;
	private Class<T> entityClass;

	public GenericDaoImpl() {

	}

	public GenericDaoImpl(Class<T> entityClass) {

		this.entityClass = entityClass;
	}

	@Override
	public T create(T entity) {
		try{
			entityManager.persist(entity);
			entityManager.flush();
		} catch(Exception exception){
			throw new DatabaseValidationException("Create Failed", exception);
		}
		return entity;
	}

	@Override
	public T update(T entity) {
		try {
			return entityManager.merge(entity);
		} catch(Exception exception){
			throw new DatabaseValidationException("Update Failed", exception);
		}
	}

	@Override
	public Boolean delete(T entity) {
		boolean deleted = false;
		try {
			entityManager.remove(entity);
			deleted = true;
		} catch(Exception exception){
			throw new DatabaseValidationException("Delete failed", exception);
		}
		return deleted;
	}

	@Override
	public T find(Long id) {
		T entity;
		try {
			entity = (T) entityManager.find(entityClass, id);
		} catch(Exception exception){
			throw new DatabaseValidationException("Find failed", exception);
		}
		return entity;
	}

}
