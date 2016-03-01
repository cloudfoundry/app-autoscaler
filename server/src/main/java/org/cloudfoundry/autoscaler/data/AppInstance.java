package org.cloudfoundry.autoscaler.data;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;


@JsonIgnoreProperties(ignoreUnknown = true)
public class AppInstance
{

  public String        instanceId;
  public String        ipAddress;
  public int           numCores;

  
  public AppInstance()
  {
	  instanceId = "";
	  ipAddress  = "";
	  numCores   = 0;
  }
  
  public AppInstance(String instId, String addr, int cores)
  {
	  instanceId = instId;
	  ipAddress  = addr;
	  numCores   = cores;
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

	public int getNumCores() {
		return numCores;
	}

	public void setNumCores(int numCores) {
		this.numCores = numCores;
	}

}
