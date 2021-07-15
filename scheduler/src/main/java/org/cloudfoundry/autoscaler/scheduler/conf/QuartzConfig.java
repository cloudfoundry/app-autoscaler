package org.cloudfoundry.autoscaler.scheduler.conf;

import java.util.Properties;
import javax.sql.DataSource;
import org.cloudfoundry.autoscaler.scheduler.quartz.QuartzJobFactory;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.boot.autoconfigure.quartz.QuartzProperties;
import org.springframework.context.ApplicationContext;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.scheduling.quartz.SchedulerFactoryBean;
import org.springframework.transaction.PlatformTransactionManager;

@Configuration
public class QuartzConfig {

  private final PlatformTransactionManager primaryTransactionManager;
  private final QuartzProperties quartzProperties;
  private final ApplicationContext applicationContext;

  public QuartzConfig(
      PlatformTransactionManager transactionManager,
      ApplicationContext applicationContext,
      QuartzProperties quartzProperties) {
    this.primaryTransactionManager = transactionManager;
    this.applicationContext = applicationContext;
    this.quartzProperties = quartzProperties;
  }

  @Bean
  public SchedulerFactoryBean quartzScheduler(@Qualifier("primary") DataSource primaryDataSource)
      throws Exception {

    SchedulerFactoryBean schedulerFactoryBean = new SchedulerFactoryBean();
    schedulerFactoryBean.setWaitForJobsToCompleteOnShutdown(true);
    schedulerFactoryBean.setOverwriteExistingJobs(true);
    schedulerFactoryBean.setDataSource(primaryDataSource);
    schedulerFactoryBean.setTransactionManager(primaryTransactionManager);
    schedulerFactoryBean.setJobFactory(quartzJobFactory());
    schedulerFactoryBean.setQuartzProperties(getQuartzProperties());
    schedulerFactoryBean.afterPropertiesSet();
    return schedulerFactoryBean;
  }

  @Bean
  public QuartzJobFactory quartzJobFactory() {
    QuartzJobFactory jobFactory = new QuartzJobFactory();
    jobFactory.setApplicationContext(applicationContext);
    return jobFactory;
  }

  public Properties getQuartzProperties() {
    Properties properties = new Properties();
    properties.putAll(quartzProperties.getProperties());
    return properties;
  }
}
