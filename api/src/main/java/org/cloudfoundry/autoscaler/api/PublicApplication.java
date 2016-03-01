package org.cloudfoundry.autoscaler.api;

import java.util.HashSet;
import java.util.Set;

import javax.ws.rs.core.Application;

import  org.cloudfoundry.autoscaler.api.rest.PublicRestApi;

public class PublicApplication extends Application {

    @Override
    public Set<Class<?>> getClasses() {
        Set<Class<?>> classes = new HashSet<Class<?>>();
        classes.add(PublicRestApi.class);
        return classes;
    }
} 