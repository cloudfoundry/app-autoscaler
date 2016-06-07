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
			return entity;
		} catch(Exception exception){
			throw new DatabaseValidationException("Create Failed", exception);
		}
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
	public void delete(T entity) {
		try {
			entityManager.remove(entity);

		} catch(Exception exception){
			throw new DatabaseValidationException("Delete failed", exception);
		}
	}

	@Override
	public T find(Long id) {
		try {
			return entityManager.find(entityClass, id);
		} catch(Exception exception){
			throw new DatabaseValidationException("Find failed", exception);
		}
	}

}
