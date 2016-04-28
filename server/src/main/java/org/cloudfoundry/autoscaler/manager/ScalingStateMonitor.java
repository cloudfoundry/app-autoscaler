package org.cloudfoundry.autoscaler.manager;

import java.util.Iterator;
import java.util.concurrent.BlockingQueue;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.LinkedBlockingQueue;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.common.util.CloudFoundryManager;
import org.cloudfoundry.autoscaler.exceptions.AppNotFoundException;
import org.cloudfoundry.autoscaler.exceptions.CloudException;

public class ScalingStateMonitor implements Runnable{
	private static final Logger logger = Logger
			.getLogger(ScalingStateMonitor.class.getName());
	BlockingQueue<ScalingStateMonitorTask> taskQueue = new LinkedBlockingQueue<ScalingStateMonitorTask>();
	private static ScalingStateMonitor monitorInstance= new ScalingStateMonitor();
	private int IS_STOPPED = 0;
	private int IS_RUNNING = 1;
	private int status = IS_STOPPED;
	private final static long THREAD_SLEEP_TIME = 5000;
    private ExecutorService taskExecutor = Executors.newFixedThreadPool(1);
	private ScalingStateMonitor(){
		
	}
	/**
	 * Gets ScalingStateMonitor instance
	 * @return
	 */
	public static ScalingStateMonitor getInstance(){
		return monitorInstance;
	}
	
	/**
	 * Adds a monitor task to the queue
	 * @param task
	 */
	public void monitor(ScalingStateMonitorTask task){
		taskQueue.add(task);
		startMonitor();
	}
	@Override
	public void run() {
		while (true){
			try{
				/** Gets tasks from task queue **/
				Iterator<ScalingStateMonitorTask> taskIter = taskQueue.iterator();
				while (taskIter.hasNext()){
					ScalingStateMonitorTask task = taskIter.next();
					doMonitor (task);
				}
				sleep (THREAD_SLEEP_TIME);
			}catch(Exception e){
				logger.error( e.getMessage(), e);
			}
		}
		
	}
	
	/**
	 * Execute a monitor task
	 * @param task
	 */
	private void doMonitor(ScalingStateMonitorTask task){
		String appId = task.getAppId();
		int targetCount = task.getTargetInstanceCount();
		try {
			CloudFoundryManager manager = CloudFoundryManager.getInstance();
			int runningInstances = manager.getRunningInstances(appId);
			int instances = manager.getAppInstancesByAppId(appId);
			String actionId = task.getScaclingActionId();
			if (runningInstances == instances){
				//Scaling is completed
				logger.info("Scaling is completed for application " + appId + ". Target count is " + targetCount + " and With current running instance number is " + runningInstances);
				ScalingStateManager.getInstance().setScalingStateCompleted(appId, actionId);
				taskQueue.remove(task);
			}
		} catch (CloudException e) {
			logger.error( "An error occurs when monitoring a scaling action for app " + task.getAppId() + "." + e.getMessage(), e);
		} catch (AppNotFoundException e) {
			taskQueue.remove(task);
			logger.error( "The application " + appId + " can not be found. ");
		}catch (Exception e) {
			String message = e.getMessage();
            if (message != null && message.contains("404 Not Found")) {
                logger.warn("Application " + appId + " is not available for now. Stop to monitor the scaling status.");
                taskQueue.remove(task);
            }
            else  
            	logger.error ("An error occurs when monitoring a scaling action for app " + task.getAppId() + ". " + e.getMessage(), e);
		}
	}
	private synchronized void startMonitor(){
		if (status != IS_RUNNING){
			//new Thread(this).start();
			taskExecutor.execute(this);
			setStatus (IS_RUNNING);
		}
	}
	/**
	 * Sets status
	 * @param status
	 */
	private void setStatus (int status){
		this.status = status;
	}
	
	private void sleep(long time){
		try {
			Thread.sleep(time);
		} catch (InterruptedException e) {
			
		}		
	}
	

	
}
