
package org.cloudfoundry.autoscaler.rest.container;

import java.util.HashSet;
import java.util.Set;

import javax.ws.rs.core.Application;

import org.cloudfoundry.autoscaler.rest.ApplicationRestApi;
import org.cloudfoundry.autoscaler.rest.BrokerCallBackRest;
import org.cloudfoundry.autoscaler.rest.EventRestApi;
import org.cloudfoundry.autoscaler.rest.PolicyRestApi;
import org.cloudfoundry.autoscaler.rest.ScalingHistoryRestApi;
import org.cloudfoundry.autoscaler.rest.ServerStatsRestApi;

public class ServiceApplication extends Application {

    @Override
    public Set<Class<?>> getClasses() {
        Set<Class<?>> classes = new HashSet<Class<?>>();
        classes.add(ApplicationRestApi.class);
        classes.add(PolicyRestApi.class);
        classes.add(EventRestApi.class);
        classes.add(ScalingHistoryRestApi.class);
        classes.add(BrokerCallBackRest.class);
        classes.add(ServerStatsRestApi.class);
        return classes;
    }
}
