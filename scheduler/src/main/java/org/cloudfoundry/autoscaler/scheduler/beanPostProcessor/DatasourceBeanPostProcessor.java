package org.cloudfoundry.autoscaler.scheduler.beanPostProcessor;

import com.zaxxer.hikari.HikariDataSource;
import java.sql.Connection;
import java.sql.SQLException;
import javax.sql.DataSource;
import org.cloudfoundry.autoscaler.scheduler.util.error.DataSourceConfigurationException;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.BeansException;
import org.springframework.beans.factory.config.BeanPostProcessor;

public class DatasourceBeanPostProcessor implements BeanPostProcessor {
  private Logger logger = LoggerFactory.getLogger(this.getClass());

  @Override
  public Object postProcessBeforeInitialization(Object bean, String beanName)
      throws BeansException {
    // TODO Auto-generated method stub
    return bean;
  }

  @Override
  public Object postProcessAfterInitialization(Object bean, String beanName) throws BeansException {
    if (bean instanceof DataSource) {
      DataSource ds = (DataSource) bean;
      Connection con = null;
      try {
        // Log datasource details for debugging
        if (ds instanceof HikariDataSource) {
          HikariDataSource hikariDs = (HikariDataSource) ds;
          logger.info(
              "Attempting to connect to datasource '{}' with URL: {}",
              beanName,
              hikariDs.getJdbcUrl());
          logger.info("Datasource '{}' username: {}", beanName, hikariDs.getUsername());
        } else {
          logger.info(
              "Attempting to connect to datasource '{}' (type: {})",
              beanName,
              ds.getClass().getSimpleName());
        }
        con = ds.getConnection();
        logger.info("Successfully connected to datasource: {}", beanName);
      } catch (SQLException e) {
        // TODO Auto-generated catch block
        logger.error("Failed to connect to datasource: " + beanName, e);
        throw new DataSourceConfigurationException(
            beanName, "Failed to connect to datasource:" + beanName, e);
      } finally {
        try {
          if (con != null && !con.isClosed()) {
            con.close();
          }
        } catch (SQLException e) {
          logger.error("Failed to close connection from " + beanName, e);
        }
      }
    }
    return bean;
  }
}
