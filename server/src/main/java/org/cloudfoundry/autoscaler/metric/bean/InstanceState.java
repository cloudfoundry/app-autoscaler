package org.cloudfoundry.autoscaler.metric.bean;

public enum InstanceState {
    DOWN, STARTING, RUNNING, CRASHED, FLAPPING, UNKNOWN;

    public static InstanceState valueOfWithDefault(String s) {
        try {
            return valueOf(s);
        } catch (IllegalArgumentException e) {
        }
        return UNKNOWN;
    }
}