package org.cloudfoundry.autoscaler.data;


public class AppState
{
	public enum InstanceScaleState {READY, REALIZING, COOL_DOWN, MIN_REACHED, MAX_REACHED};
	
	public InstanceScaleState instanceCountState = InstanceScaleState.READY;

	public String lastActionTriggerId;
	public int    lastActionInstanceTarget;
	public long   instanceStepCoolDownEndTime;

}
