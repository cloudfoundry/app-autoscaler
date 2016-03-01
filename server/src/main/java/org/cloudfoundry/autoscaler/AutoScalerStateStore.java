package org.cloudfoundry.autoscaler;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.Iterator;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.data.AppState;
import org.cloudfoundry.autoscaler.data.PolicyData;



public class AutoScalerStateStore
{
	private static final String CLASS_NAME = AutoScalerStateStore.class.getName();
	private static final Logger logger     = Logger.getLogger(CLASS_NAME); 

	private static final boolean POPULATE_WITH_DEBUG_DATA = true;
	private static final boolean PRINT_CONFIG             = false;
	
	private static class ConfigData {
		private ArrayList<String> appIdList = new ArrayList<String>();
		private PolicyData     appConfigData;
		private ConfigData(PolicyData data) {
			appConfigData = data;
		}
	}
	
	private static class AppData {
		private String   configId;
		private AppState stateData;
		private AppData(String id) {
			configId  = id;
			stateData = new AppState();
		}
	}
	
	
	private HashMap<String,ConfigData> configTable;   // table of configs, searched by config ID
	private HashMap<String,AppData>    appTable;      // table of apps, searched by app ID
	

	public AutoScalerStateStore()
	{
		configTable = new HashMap<String,ConfigData>();
		appTable    = new HashMap<String,AppData>();
		
		if (POPULATE_WITH_DEBUG_DATA) {
			this.addAppGroup(new String[]{"app1","app2","app3"},"config1",new PolicyData());
			this.addApp("app4","config2",new PolicyData());
			this.addConfig("config0",new PolicyData());
		}
	}
	
	// basic operations on configs
	
	// add (define) a new config; will not do anything if the config already exists
	public boolean addConfig(String configId, PolicyData config)
	{
		if (configTable.containsKey(configId)) {
			return true; 
		}
		configTable.put(configId,new ConfigData(config));
		return false;
	}
	
	// remove an existing config; this fails if the config has one or more associated apps
	public boolean removeConfig(String configId)
	{
		ConfigData configData = configTable.get(configId);
		if (configData == null) {
			return true; 
		}
		if ( ! configData.appIdList.isEmpty()) {
			return true; 
		}
		configTable.remove(configId);
		return false;
	}
	
	// replace an existing config; will not do anything if the config does not exist
	public boolean replaceConfig(String configId, PolicyData config)
	{
		ConfigData configData = configTable.get(configId);
		if (configData == null) {
			return true; 
		}
		configData.appConfigData = config;
		return false;
	}
	
	// check if a certain config exists
	public boolean configExists(String configId)
	{
		return configTable.containsKey(configId);
	}
	
	// get the config for a certain config ID; returns null if the config does not exist
	public PolicyData getConfig(String configId)
	{ 
		ConfigData configData = configTable.get(configId);
		if (configData == null) {
			return null;
		}
		return configData.appConfigData;
	}
	
	// get the list of config ID's
	public String[] getConfigIdList()
	{
		if (PRINT_CONFIG) {
		  this.printConfigs();
		}
		return (String[])configTable.keySet().toArray(new String[0]);
	}

	public String[] getConfigAppList(String configId) 
	{
		ConfigData configData = configTable.get(configId);
		if (configData == null) {
			return null;
		}
		return (String[])configData.appIdList.toArray(new String[0]);
	}
	
	// basic operations on apps
	
	// add a single app with an existing config
	public boolean addApp(String appId, String configId)
	{
		ConfigData configData = configTable.get(configId);
		if (configData == null) {
			return true; 
		}
		configData.appIdList.add(appId);
		appTable.put(appId,new AppData(configId));
		return false;
	}
	
	// remove an app; this will not remove the config, even if there are no apps associated with it anymore
	public boolean removeApp(String appId)
	{
		AppData appData = appTable.remove(appId);
		if (appData == null) {
			logger.warn("removeApp: app not found in app table: "+appId);
			return true;
		}
		ConfigData configData = configTable.get(appData.configId);
		if (configData == null) {
			logger.warn("removeApp: app not found in config table: "+appId);
			return true;
		}
		configData.appIdList.remove(appId);
		return false;
	}
	
	// check if a certain app exists
	public boolean appExists(String appId)
	{
		return appTable.containsKey(appId);
	}
	
	// get the config ID for a certain app
	public String getConfigId(String appId)
	{
		AppData appData = appTable.get(appId);
		if (appData == null) {
			return null;
		}
		return appData.configId;
	}
	
	// get the list of app ID's
	public String[] getAppList()
	{
		return (String[])appTable.keySet().toArray((new String[0]));
	}
	
	// put the state for a certain app
	public boolean putAppState(String appId, AppState state)
	{
		AppData appData = appTable.get(appId);
		if (appData == null) {
			return true; 
		}
		appData.stateData = state;
		appTable.put(appId,appData);
		return false;
	}
	
	// get the state for a certain app
	public AppState getAppState(String appId)
	{
		AppData appData = appTable.get(appId);
		if (appData == null) {
			return null;
		}
		return appData.stateData;
	}
	
	// compound operations
	
	public boolean addApp(String appId, String configId, PolicyData config)
	{
		if (this.addConfig(configId,config)) {
			return true;
		}
		if (this.addApp(appId,configId)) {
			this.removeConfig(configId);
			return true;
		}
		return false;
	}
	
	public boolean addAppGroup(String[] appIdList, String configId, PolicyData config)
	{
		if (this.addConfig(configId,config)) {
			return true;
		}
		for (String appId : appIdList) {
			this.addApp(appId,configId);
		}
		return false;
	}
	
	public void removeAppGroup(String configId) 
	{
		String[] appIdList = this.getConfigAppList(configId);
		if (appIdList == null) {
			logger.warn("Attempting to remove non-existing app group: configId = "+configId);
			return;
		}
		for (String appId : appIdList) {
			this.removeApp(appId);
		}
		this.removeConfig(configId);
	}
	
	
	public PolicyData getAppConfig(String appId)
	{
		String configId = this.getConfigId(appId);
		if (configId == null) {
			return null;
		}
		return this.getConfig(configId);
	}

	
	private void printConfigs()
	{
		Iterator<String> configIdIter = configTable.keySet().iterator();
		while (configIdIter.hasNext()) {
			String     configId   = configIdIter.next();
			ConfigData configData =  configTable.get(configId);
			logger.info( "Config ID: " + configId);
			logger.info( "  metricType = "+configData.appConfigData.getMetricType());
			logger.info( "  apps ("+configData.appIdList.size()+") = ");
			Iterator<String> appIdIter = configData.appIdList.iterator();
			while (appIdIter.hasNext()) {
				String appId = appIdIter.next();
				logger.info( "  "+appId);
			}
			logger.info( "");
		}
	}
	
}
