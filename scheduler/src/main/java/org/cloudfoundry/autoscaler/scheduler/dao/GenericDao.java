package org.cloudfoundry.autoscaler.scheduler.dao;

/**
 * @author Fujitsu
 *
 * @param <T>
 */
public interface GenericDao<T> {

	public T create(T entity);

	public T update(T entity);

	public Boolean delete(T entity);

	public T find(Long id);

}