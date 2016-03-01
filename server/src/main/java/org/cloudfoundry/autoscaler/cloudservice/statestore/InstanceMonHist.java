package org.cloudfoundry.autoscaler.cloudservice.statestore;

import java.util.ArrayList;

public class InstanceMonHist
{

	private static final long MAX_HISTORY_MSECS = 1000L * 60L * 60L; // 1 hour - this must be a configurable property
	
  public String                   instanceId;
  public ArrayList<MonitorSample> sampleList;

  
  public InstanceMonHist()
  {
  }
  
  public InstanceMonHist(String instId)
  {
	  instanceId = instId;
	  sampleList = new ArrayList<MonitorSample>();
  }
  
	public void addMonitorSample(MonitorSample sam)
	{
		if ( ! sampleList.isEmpty()) {
			long oldestTime = sampleList.get(0).timeMsecs;
			if (sam.timeMsecs - oldestTime > MAX_HISTORY_MSECS) {
				sampleList.remove(0);
			}
		}
		sampleList.add(sam);
	}

	public String getInstanceId() {
		return instanceId;
	}

	public void setInstanceId(String instanceId) {
		this.instanceId = instanceId;
	}

	public ArrayList<MonitorSample> getSampleList() {
		return sampleList;
	}

	public void setSampleList(ArrayList<MonitorSample> sampleList) {
		this.sampleList = sampleList;
	}

	public static long getMaxHistoryMsecs() {
		return MAX_HISTORY_MSECS;
	}
	
}
