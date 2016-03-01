package org.cloudfoundry.autoscaler.cloudservice.statestore;

public class MonitorSample {

	public long timeMsecs;
	public int  cpuPerc;
	public int  memMB;
	public int  ioMBsec;
	
	public MonitorSample()
	{
	}
	
	public MonitorSample(long tim, int cpu, int mem, int io)
	{
		timeMsecs = tim;
		cpuPerc   = cpu;
		memMB     = mem;
		ioMBsec   = io;
	}

	public long getTimeMsecs() {
		return timeMsecs;
	}
	public void setTimeMsecs(long timeMsecs) {
		this.timeMsecs = timeMsecs;
	}
	public int getCpuPerc() {
		return cpuPerc;
	}
	public void setCpuPerc(int cpuPerc) {
		this.cpuPerc = cpuPerc;
	}
	public int getMemMB() {
		return memMB;
	}
	public void setMemMB(int memMB) {
		this.memMB = memMB;
	}
	public int getIoMBsec() {
		return ioMBsec;
	}
	public void setIoMBsec(int ioMBsec) {
		this.ioMBsec = ioMBsec;
	}
	
}
