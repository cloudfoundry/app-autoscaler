package org.cloudfoundry.autoscaler.scheduler.conf;

import org.cloudfoundry.autoscaler.scheduler.quartz.QuartzJobFactory;
import org.springframework.boot.autoconfigure.quartz.QuartzProperties;
import org.springframework.context.ApplicationContext;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.scheduling.quartz.SchedulerFactoryBean;
import org.springframework.transaction.PlatformTransactionManager;

import javax.sql.DataSource;
import java.util.Properties;

@Configuration
public class QuartzConfig {

    private final DataSource primaryDataSource;
    private final PlatformTransactionManager primaryTransactionManager;
    private final QuartzProperties quartzProperties;
    private final ApplicationContext applicationContext;

    public QuartzConfig(DataSource primaryDataSource, PlatformTransactionManager transactionManager, ApplicationContext applicationContext, QuartzProperties quartzProperties) {
        this.primaryDataSource = primaryDataSource;
        this.primaryTransactionManager = transactionManager;
        this.applicationContext = applicationContext;
        this.quartzProperties = quartzProperties;
    }

    @Bean
    public SchedulerFactoryBean quartzScheduler() {

        SchedulerFactoryBean schedulerFactoryBean = new SchedulerFactoryBean();
        schedulerFactoryBean.setQuartzProperties(getQuartzProperties());
        schedulerFactoryBean.setApplicationContext(applicationContext);
        schedulerFactoryBean.setWaitForJobsToCompleteOnShutdown(true);
        schedulerFactoryBean.setOverwriteExistingJobs(true);
        schedulerFactoryBean.setDataSource(primaryDataSource);
        schedulerFactoryBean.setTransactionManager(primaryTransactionManager);
        schedulerFactoryBean.setJobFactory(quartzJobFactory());

        return new SchedulerFactoryBean();
    }

    @Bean
    public QuartzJobFactory quartzJobFactory() {
        return new QuartzJobFactory();
    }

    private Properties getQuartzProperties() {
        Properties properties = new Properties();
        properties.putAll(quartzProperties.getProperties());
        return properties;
    }
}
