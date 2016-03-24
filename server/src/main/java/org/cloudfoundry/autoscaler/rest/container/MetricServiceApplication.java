package org.cloudfoundry.autoscaler.rest.container;

import java.util.HashSet;
import java.util.Set;

import javax.ws.rs.core.Application;

import org.cloudfoundry.autoscaler.metric.rest.DashboardREST;
import org.cloudfoundry.autoscaler.metric.rest.OperationREST;
import org.cloudfoundry.autoscaler.metric.rest.SubscriberREST;
import org.cloudfoundry.autoscaler.metric.rest.TestModeREST;

public class MetricServiceApplication extends Application {

    @Override
    public Set<Class<?>> getClasses() {
        Set<Class<?>> classes = new HashSet<Class<?>>();
        classes.add(SubscriberREST.class);
        classes.add(TestModeREST.class);
        classes.add(DashboardREST.class);
        classes.add(OperationREST.class);
        return classes;
    }
}
