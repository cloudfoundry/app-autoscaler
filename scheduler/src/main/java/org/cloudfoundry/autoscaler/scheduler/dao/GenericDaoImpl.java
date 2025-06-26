package org.cloudfoundry.autoscaler.scheduler.dao;

import jakarta.persistence.EntityManager;
import jakarta.persistence.PersistenceContext;
import org.cloudfoundry.autoscaler.scheduler.util.error.DatabaseValidationException;

class GenericDaoImpl<T> implements GenericDao<T> {

  @PersistenceContext EntityManager entityManager;
  private Class<T> entityClass;

  GenericDaoImpl() {}

  GenericDaoImpl(Class<T> entityClass) {

    this.entityClass = entityClass;
  }

  @Override
  public T create(T entity) {
    try {
      entityManager.persist(entity);
      entityManager.flush();
      return entity;
    } catch (Exception exception) {
      throw new DatabaseValidationException("Create failed", exception);
    }
  }

  @Override
  public T update(T entity) {
    try {
      return entityManager.merge(entity);
    } catch (Exception exception) {
      throw new DatabaseValidationException("Update failed", exception);
    }
  }

  @Override
  public void delete(T entity) {
    try {
      entityManager.remove(entity);
      entityManager.flush();
    } catch (Exception exception) {
      throw new DatabaseValidationException("Delete failed", exception);
    }
  }

  @Override
  public T find(Long id) {
    try {
      return entityManager.find(entityClass, id);

    } catch (Exception exception) {
      throw new DatabaseValidationException("Find failed", exception);
    }
  }
}
