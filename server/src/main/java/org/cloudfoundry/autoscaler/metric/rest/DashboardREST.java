package org.cloudfoundry.autoscaler.metric.rest;

import java.util.ArrayList;
import java.util.Collection;
import java.util.Date;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

import javax.ws.rs.GET;
import javax.ws.rs.Path;
import javax.ws.rs.PathParam;
import javax.ws.rs.Produces;
import javax.ws.rs.QueryParam;
import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.Response;
import javax.ws.rs.core.Response.Status;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.bean.InstanceMetrics;
import org.cloudfoundry.autoscaler.bean.Metric;
import org.cloudfoundry.autoscaler.common.util.RestApiResponseHandler;
import org.cloudfoundry.autoscaler.constant.Constants;
import org.cloudfoundry.autoscaler.data.couchdb.AutoScalingDataStoreFactory;
import org.cloudfoundry.autoscaler.data.couchdb.document.AppInstanceMetrics;
import org.cloudfoundry.autoscaler.data.couchdb.document.BoundApp;
import org.cloudfoundry.autoscaler.metric.bean.ApplicationMetrics;
import org.cloudfoundry.autoscaler.metric.monitor.MonitorController;
import org.cloudfoundry.autoscaler.util.MetricConfigManager;

import com.fasterxml.jackson.databind.ObjectMapper;

@Path("/metrics")
public class DashboardREST {
    private static final Logger logger = Logger.getLogger(DashboardREST.class);

    private static final ObjectMapper mapper = new ObjectMapper();

    @GET
    @Path("/{serviceId}")
    @Produces(MediaType.APPLICATION_JSON)
    @SuppressWarnings({ "rawtypes", "unchecked" })
    public Response getAppMetricsLastestData(@PathParam("serviceId") String serviceId, @QueryParam("appId") String appId) {
        try {
        	
            List<AppInstanceMetrics> stats = new ArrayList<AppInstanceMetrics>();
            MonitorController controller = MonitorController.getInstance();
            Collection<BoundApp> apps = controller.getSerivceBoundApps(serviceId);
            if (apps != null && apps.size() > 0) {
            	controller.updateBoundAppName(apps);
                for (BoundApp app : apps) {
                    String boundAppId = app.getAppId();
                    //query all bounded app for this service when appId == null
                    //query for a specific app when appId is defined
                    if ((appId == null) || (appId.equals(boundAppId))) {
                        AppInstanceMetrics appInstanceMetrics = new AppInstanceMetrics();
                        ApplicationMetrics appMetrics = controller.getAppMetrics(boundAppId);
                        if (appMetrics != null) {
                        	//not store to db 
                        	//stale data is allowed for overview page
                        	appInstanceMetrics = appMetrics.mergeToAppInstanceMetrics(false, true);
                        } else {
                        	appInstanceMetrics.setAppId(boundAppId);
                        	appInstanceMetrics.setAppName(app.getAppName());
                        	appInstanceMetrics.setServiceId(app.getServiceId());
                        	appInstanceMetrics.setAppType(app.getAppType());
                        }
                        stats.add(appInstanceMetrics);
                        //get out of the loop if the appId is defined.
                        if ((appId != null) && (appId.equals(boundAppId)))
                        	break; 
                        
                    }
                }
            }
            
            return RestApiResponseHandler.getResponseOk(mapper.writeValueAsString(stats));
        } catch (Exception e) {
            logger.error("Internal_Server_Error", e);
            return RestApiResponseHandler.getResponse(Status.INTERNAL_SERVER_ERROR);
        }

    }

    @GET
    @Path("/{serviceId}/{appId}")
    @Produces(MediaType.APPLICATION_JSON)
    public Response getAppHistoryMetrics(@PathParam("serviceId") String serviceId, @PathParam("appId") String appId,
            @QueryParam("newerThan") long newerThan, @QueryParam("timeRange") long timeRange) {

    	long now = new Date().getTime(); 
    	
        if (timeRange == 0)
            timeRange = Constants.DASHBORAD_TIME_RANGE;

        if (newerThan == 0) {
            newerThan = now  - timeRange * 60 * 1000;
        }

        Map<String, Object> responseMap = new HashMap<String, Object>();

        List<AppInstanceMetrics> stats = new ArrayList<AppInstanceMetrics>();
        ApplicationMetrics appMetrics = MonitorController.getInstance().getAppMetrics(appId);
        try {
        	if (MonitorController.getInstance().sotore2db()) {
                stats = AutoScalingDataStoreFactory
        				.getAutoScalingDataStore().getAppStatsHistoryByAppIdAfter(appId, newerThan/*,
                        runningInstIndexes*/);
            }

        	//if the app is running, but the data is not stored to db yet. then return the latest value in app Metric map.
        	if ( (appMetrics != null) && (stats.size() == 0) ) {
            	//not store to db 
            	//stale data is not forbidden for history page to avoid duplication.
        		AppInstanceMetrics lastestMetrics = appMetrics.mergeToAppInstanceMetrics(false, false);
        		if (lastestMetrics.getInstanceMetrics().size() > 0)
        			stats.add(lastestMetrics);
        		else{
               		//if no non-stored metric in the required duration, create a fake data to indiccate the latest metrifc report is null.
        			int instanceNumber = appMetrics.getPollerMetricsMap().size(); 
                	stats.add(createFakeMetrics(instanceNumber, now));
        		}
        	}
        	 //if the app is stopped, but has history metric in db, create a fake data to indicate the latest metric report is null.
        	if ( (appMetrics == null) && ( stats.size() > 0) ) {
        		int instanceNumber = stats.get(stats.size()-1).getInstanceMetrics().size();
            	stats.add(createFakeMetrics(instanceNumber, now));
        	}
        	
            responseMap.put("data", stats);

            MetricConfigManager configService = MetricConfigManager.getInstance();
            String appType = null;
            BoundApp boundApp = MonitorController.getInstance().getBoundApp(serviceId, appId);
            if (boundApp != null) {
                appType = boundApp.getAppType();
            }

            
            Map<String, Object> configMap = configService.loadDefaultConfig(appType, appId);
            responseMap.put("config", configMap);

            return RestApiResponseHandler.getResponseOk(mapper.writeValueAsString(responseMap));
        } catch (Exception e) {
            logger.error("Internal_Server_Error", e);
            return RestApiResponseHandler.getResponse(Status.INTERNAL_SERVER_ERROR);
        }

    }
    
    @GET
    @Path("/default/{appType}")
    @Produces(MediaType.APPLICATION_JSON)
    public Response getAppHistoryMetrics(@PathParam("serviceId") String serviceId, @PathParam("appType") String appType) {

    	try {	
    	   Map<String, Object> responseMap = new HashMap<String, Object>();

            MetricConfigManager configService = MetricConfigManager.getInstance();
   
            Map<String, Object> configMap = configService.loadDefaultConfig(appType);
            responseMap.put("config", configMap);

            return RestApiResponseHandler.getResponseOk(mapper.writeValueAsString(responseMap));
        } catch (Exception e) {
            logger.error("Internal_Server_Error", e);
            return RestApiResponseHandler.getResponse(Status.INTERNAL_SERVER_ERROR);
        }

    }
    
    private AppInstanceMetrics createFakeMetrics(int instanceNumber, long timestamp){
    	List<InstanceMetrics> fakeMetrics = new ArrayList<InstanceMetrics>(instanceNumber);
    	for (InstanceMetrics fakeMetric : fakeMetrics){
    		fakeMetric.setTimestamp(timestamp);
    		fakeMetric.setMetrics(new ArrayList<Metric>());
    	}

    	AppInstanceMetrics fakeInstanceMetrics = new AppInstanceMetrics();
    	fakeInstanceMetrics.setInstanceMetrics(fakeMetrics);
    	return fakeInstanceMetrics;
    	
    }
}
