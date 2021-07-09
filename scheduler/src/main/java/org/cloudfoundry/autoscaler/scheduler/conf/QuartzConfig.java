package org.cloudfoundry.autoscaler.scheduler.conf;

import org.cloudfoundry.autoscaler.scheduler.quartz.QuartzJobFactory;
import org.quartz.spi.TriggerFiredBundle;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.beans.factory.config.AutowireCapableBeanFactory;
import org.springframework.boot.autoconfigure.quartz.QuartzProperties;
import org.springframework.context.ApplicationContext;
import org.springframework.context.ApplicationContextAware;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.context.annotation.Lazy;
import org.springframework.scheduling.quartz.SchedulerFactoryBean;
import org.springframework.scheduling.quartz.SpringBeanJobFactory;
import org.springframework.transaction.PlatformTransactionManager;

import javax.sql.DataSource;
import java.util.Properties;

@Configuration
public class QuartzConfig {


   /* @Value("${spring.org.quartz.scheduler.instanceId}")
    private String schedulerInstanceId;*/

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
    public SchedulerFactoryBean quartzScheduler(@Qualifier("primary") DataSource primaryDataSource) throws Exception {

        SchedulerFactoryBean schedulerFactoryBean = new SchedulerFactoryBean();
        schedulerFactoryBean.setWaitForJobsToCompleteOnShutdown(true);
        schedulerFactoryBean.setOverwriteExistingJobs(true);
        schedulerFactoryBean.setDataSource(primaryDataSource);
        schedulerFactoryBean.setTransactionManager(primaryTransactionManager);
        schedulerFactoryBean.setJobFactory(quartzJobFactory());

        schedulerFactoryBean.setQuartzProperties(getQuartzProperties());
        //schedulerFactoryBean.setSchedulerName(schedulerName);
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
