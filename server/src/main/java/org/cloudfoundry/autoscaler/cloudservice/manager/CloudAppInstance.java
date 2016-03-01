package org.cloudfoundry.autoscaler.cloudservice.manager;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;


@JsonIgnoreProperties(ignoreUnknown = true)
public class CloudAppInstance
{

  private String instanceId;
  private String ipAddress;
  private double numCores;
  private double cpuPerc;
  private int    memMB;

  
  public CloudAppInstance()
  {
	  instanceId = "";
	  ipAddress  = "";
  }
  
  public CloudAppInstance(String instId, String addr, double cores, double cpu, int mem)
  {
	  instanceId = instId;
	  ipAddress  = addr;
	  numCores   = cores;
	  cpuPerc    = cpu;
	  memMB      = mem;
  }
  
	public String getInstanceId() {
		return instanceId;
	}

	public void setInstanceId(String instanceId) {
		this.instanceId = instanceId;
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

	public int getMemMB() {
		return memMB;
	}

	public void setMemMB(int memMB) {
		this.memMB = memMB;
	}

}
