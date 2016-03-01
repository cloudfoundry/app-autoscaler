package org.cloudfoundry.autoscaler.data;


import com.fasterxml.jackson.annotation.JsonIgnoreProperties;


@JsonIgnoreProperties(ignoreUnknown = true)
public class ScaleEvent
{

	private long time;
	private int  oldInstanceCount;
	private int  newInstanceCount;

	
	public ScaleEvent()
	{
	}
	
	public ScaleEvent(long tim, int oldCount, int newCount)
	{
		time             = tim;
		oldInstanceCount = oldCount;
		newInstanceCount = newCount;
	}
	
	public long getTime() {
		return time;
	}

	public void setTime(long time) {
		this.time = time;
	}

	public int getOldInstanceCount() {
		return oldInstanceCount;
	}

	public void setOldInstanceCount(int oldInstanceCount) {
		this.oldInstanceCount = oldInstanceCount;
	}

	public int getNewInstanceCount() {
		return newInstanceCount;
	}

	public void setNewInstanceCount(int newInstanceCount) {
		this.newInstanceCount = newInstanceCount;
	}

}
