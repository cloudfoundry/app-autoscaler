package org.cloudfoundry.autoscaler.scheduler.dao;

public interface GenericDao<T> {

  public T create(T entity);

  public T update(T entity);

  public void delete(T entity);

  public T find(Long id);
}
