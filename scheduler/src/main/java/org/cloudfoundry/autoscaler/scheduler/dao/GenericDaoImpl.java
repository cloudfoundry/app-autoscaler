package org.cloudfoundry.autoscaler.scheduler.dao;

import javax.persistence.EntityManager;
import javax.persistence.PersistenceContext;

/**
 * @author Fujitsu
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
		entityManager.persist(entity);
		entityManager.flush();
		return entity;
	}

	@Override
	public T update(T entity) {
		try {
			return entityManager.merge(entity);
		} catch (Exception ex) {
			return null;
		}
	}

	@Override
	public Boolean delete(T entity) {
		try {
			entityManager.remove(entity);
		} catch (Exception ex) {
			return false;
		}
		return true;
	}

	@Override
	public T find(Long id) {
		return (T) entityManager.find(entityClass, id);
	}

}
