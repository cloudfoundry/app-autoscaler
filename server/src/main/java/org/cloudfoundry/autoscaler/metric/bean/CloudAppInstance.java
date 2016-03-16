package org.cloudfoundry.autoscaler.metric.bean;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;

@JsonIgnoreProperties(ignoreUnknown = true)
public class CloudAppInstance {

    private String instanceIndex;
    private String ipAddress;
    private double numCores;
    private double cpuPerc;
    private double memMB;
    private double memQuotaMB;
    private long timestamp;

    public CloudAppInstance() {
        instanceIndex = "";
        ipAddress = "";
    }

    public CloudAppInstance(String instId, String addr, double cores, double cpu, double mem, double memQuotaMB) {
        this(instId, addr, cores, cpu, mem, memQuotaMB, System.currentTimeMillis());
    }

    public CloudAppInstance(String instId, String addr, double cores, double cpu, double mem, double memQuotaMB,
            long timestamp) {
        instanceIndex = instId;
        ipAddress = addr;
        numCores = cores;
        cpuPerc = cpu;
        memMB = mem;
        this.memQuotaMB = memQuotaMB;
        this.timestamp = timestamp;
    }

    public String getInstanceIndex() {
        return instanceIndex;
    }

    public void setInstanceIndex(String instanceId) {
        this.instanceIndex = instanceId;
    }

    public String getIpAddress() {
        return ipAddress;
    }

    public void setIpAddress(String ipAddress) {
        this.ipAddress = ipAddress;
    }

    public double getNumCores() {
        return numCores;
    }

    public void setNumCores(double numCores) {
        this.numCores = numCores;
    }

    public double getCpuPerc() {
        return cpuPerc;
    }

    public void setCpuPerc(double cpuPerc) {
        this.cpuPerc = cpuPerc;
    }

    public double getMemMB() {
        return memMB;
    }

    public void setMemMB(double memMB) {
        this.memMB = memMB;
    }

    public double getMemQuotaMB() {
        return memQuotaMB;
    }

    public void setMemQuotaMB(double memQuotaMB) {
        this.memQuotaMB = memQuotaMB;
    }

    public long getTimestamp() {
        return timestamp;
    }

    public void setTimestamp(long timestamp) {
        this.timestamp = timestamp;
    }

}
