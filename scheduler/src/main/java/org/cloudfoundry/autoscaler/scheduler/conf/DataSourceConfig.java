package org.cloudfoundry.autoscaler.scheduler.conf;

import com.zaxxer.hikari.HikariDataSource;
import java.util.Properties;
import javax.sql.DataSource;
import org.cloudfoundry.autoscaler.scheduler.beanPostProcessor.DatasourceBeanPostProcessor;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.boot.autoconfigure.jdbc.DataSourceProperties;
import org.springframework.boot.context.properties.ConfigurationProperties;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.context.annotation.Primary;
import org.springframework.orm.jpa.JpaTransactionManager;
import org.springframework.orm.jpa.LocalContainerEntityManagerFactoryBean;
import org.springframework.orm.jpa.vendor.HibernateJpaVendorAdapter;
import org.springframework.transaction.annotation.EnableTransactionManagement;

@Configuration
@EnableTransactionManagement
public class DataSourceConfig {

  @Bean
  @Primary
  @Qualifier("primary")
  @ConfigurationProperties(prefix = "spring.datasource")
  public DataSource dataSource(@Qualifier("primary") DataSourceProperties properties) {
    return properties.initializeDataSourceBuilder().type(HikariDataSource.class).build();
  }

  @Bean
  @Qualifier("primary")
  @ConfigurationProperties(prefix = "spring.datasource")
  public DataSourceProperties dataSourceProperties() {
    return new DataSourceProperties();
  }

  @Bean
  @Qualifier("policy")
  @ConfigurationProperties("spring.policy-db-datasource")
  public DataSourceProperties policyDbDataSourceProperties() {
    return new DataSourceProperties();
  }

  @Bean
  @Qualifier("policy")
  @ConfigurationProperties("spring.policy-db-datasource")
  public DataSource policyDbDataSource(@Qualifier("policy") DataSourceProperties properties) {
    return properties.initializeDataSourceBuilder().type(HikariDataSource.class).build();
  }

  @Bean
  @Qualifier("primary")
  public LocalContainerEntityManagerFactoryBean entityManagerFactory(
      @Qualifier("primary") DataSource primary) {

    LocalContainerEntityManagerFactoryBean localContainerEntityManagerFactoryBean =
        new LocalContainerEntityManagerFactoryBean();
    localContainerEntityManagerFactoryBean.setDataSource(primary);
    localContainerEntityManagerFactoryBean.setPackagesToScan(
        "org.cloudfoundry.autoscaler.scheduler");
    localContainerEntityManagerFactoryBean.setPersistenceUnitName("default-persistence-unit");
    HibernateJpaVendorAdapter jpaVendorAdapter = new HibernateJpaVendorAdapter();

    jpaVendorAdapter.setGenerateDdl(false);
    localContainerEntityManagerFactoryBean.setJpaVendorAdapter(jpaVendorAdapter);
    localContainerEntityManagerFactoryBean.setJpaProperties(additionalProperties());
    return localContainerEntityManagerFactoryBean;
  }

  @Bean
  @Primary
  @Qualifier("primary")
  public JpaTransactionManager transactionManager(
      @Qualifier("primary")
          LocalContainerEntityManagerFactoryBean localContainerEntityManagerFactoryBean) {
    JpaTransactionManager transactionManager = new JpaTransactionManager();
    transactionManager.setEntityManagerFactory(localContainerEntityManagerFactoryBean.getObject());
    return transactionManager;
  }

  @Bean
  @Qualifier("policy")
  public JpaTransactionManager policyDbTransactionManager(
      @Qualifier("policy") DataSource policyDataSource) {
    JpaTransactionManager policyTransactionManager = new JpaTransactionManager();
    policyTransactionManager.setDataSource(policyDataSource);
    return policyTransactionManager;
  }

  private Properties additionalProperties() {
    Properties properties = new Properties();
    properties.setProperty("hibernate.hbm2ddl.auto", "none");
    properties.setProperty("hibernate.show_sql", "false");
    return properties;
  }

  @Bean
  public DatasourceBeanPostProcessor datasourceBeanPostProcessor() {
    return new DatasourceBeanPostProcessor();
  }
}
