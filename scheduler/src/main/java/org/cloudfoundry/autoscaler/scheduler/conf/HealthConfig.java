package org.cloudfoundry.autoscaler.scheduler.conf;

import org.cloudfoundry.autoscaler.scheduler.health.DBStatusCollector;
import org.cloudfoundry.autoscaler.scheduler.health.HealthExporter;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import javax.sql.DataSource;

@Configuration
public class HealthConfig {

    @Bean
    public DBStatusCollector dbStatusCollector(@Qualifier("primary") DataSource primaryDS, @Qualifier("policy") DataSource policyDS) {
        DBStatusCollector dbStatusCollector = new DBStatusCollector();
        dbStatusCollector.setDataSource(primaryDS);
        dbStatusCollector.setPolicyDBDataSource(policyDS);
        return dbStatusCollector;
    }

    @Bean
    public HealthExporter healthExporter(@Autowired DBStatusCollector dbStatusCollector) {
        HealthExporter healthExporter = new HealthExporter();
        healthExporter.setDbStatusCollector(dbStatusCollector);
        healthExporter.init();
        return healthExporter;
    }
}
