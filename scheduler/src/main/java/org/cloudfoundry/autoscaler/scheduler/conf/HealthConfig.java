package org.cloudfoundry.autoscaler.scheduler.conf;

import javax.sql.DataSource;
import org.cloudfoundry.autoscaler.scheduler.health.DbStatusCollector;
import org.cloudfoundry.autoscaler.scheduler.health.HealthExporter;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Configuration
public class HealthConfig {

  @Bean
  public DbStatusCollector dbStatusCollector(
      @Qualifier("primary") DataSource primaryDs, @Qualifier("policy") DataSource policyDs) {
    DbStatusCollector dbStatusCollector = new DbStatusCollector();
    dbStatusCollector.setDataSource(primaryDs);
    dbStatusCollector.setPolicyDbDataSource(policyDs);
    return dbStatusCollector;
  }

  @Bean(initMethod = "init")
  public HealthExporter healthExporter(@Autowired DbStatusCollector dbStatusCollector) {
    HealthExporter healthExporter = new HealthExporter();
    healthExporter.setDbStatusCollector(dbStatusCollector);
    return healthExporter;
  }
}
