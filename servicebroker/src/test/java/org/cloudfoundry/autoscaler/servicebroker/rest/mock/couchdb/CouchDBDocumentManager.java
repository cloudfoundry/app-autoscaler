package org.cloudfoundry.autoscaler.servicebroker.rest.mock.couchdb;

import static org.cloudfoundry.autoscaler.servicebroker.test.constant.Constants.APPAUTOSCALESTATEDOCTESTID;
import static org.cloudfoundry.autoscaler.servicebroker.test.constant.Constants.APPINSTANCEMETRICSDOCTESTID;
import static org.cloudfoundry.autoscaler.servicebroker.test.constant.Constants.APPLICATIONDOCTESTID;
import static org.cloudfoundry.autoscaler.servicebroker.test.constant.Constants.APPLICATIONINSTANCEDOCTESTID;
import static org.cloudfoundry.autoscaler.servicebroker.test.constant.Constants.AUTOSCALERPOLICYDOCTESTID;
import static org.cloudfoundry.autoscaler.servicebroker.test.constant.Constants.BOUNDAPPDOCTESTID;
import static org.cloudfoundry.autoscaler.servicebroker.test.constant.Constants.BOUNDSTATE;
import static org.cloudfoundry.autoscaler.servicebroker.test.constant.Constants.METRICDBSEGMENTDOCTESTID;
import static org.cloudfoundry.autoscaler.servicebroker.test.constant.Constants.SCALINGHISTORYDOCTESTID;
import static org.cloudfoundry.autoscaler.servicebroker.test.constant.Constants.SERVERNAME;
import static org.cloudfoundry.autoscaler.servicebroker.test.constant.Constants.SERVICECONFIGDOCTESTID;
import static org.cloudfoundry.autoscaler.servicebroker.test.constant.Constants.SERVICEINSTANCEDOCTESTID;
import static org.cloudfoundry.autoscaler.servicebroker.test.constant.Constants.TESTAPPID;
import static org.cloudfoundry.autoscaler.servicebroker.test.constant.Constants.TESTBINDINGID;
import static org.cloudfoundry.autoscaler.servicebroker.test.constant.Constants.TESTORGID;
import static org.cloudfoundry.autoscaler.servicebroker.test.constant.Constants.TESTPOLICYID;
import static org.cloudfoundry.autoscaler.servicebroker.test.constant.Constants.TESTSERVERURL;
import static org.cloudfoundry.autoscaler.servicebroker.test.constant.Constants.TESTSERVICEID;
import static org.cloudfoundry.autoscaler.servicebroker.test.constant.Constants.TESTSPACEID;
import static org.cloudfoundry.autoscaler.servicebroker.test.constant.Constants.TRIGGERRECORDDOCTESTID;

import java.util.HashMap;
import java.util.Map;

import org.json.JSONArray;
import org.json.JSONObject;

public class CouchDBDocumentManager {
	private static CouchDBDocumentManager manager;
	public static CouchDBDocumentManager getInstance(){
		if(manager == null){
			manager = new CouchDBDocumentManager();
		}
		return manager;
	}
	public Map<String, JSONObject> documentMap = new HashMap<String, JSONObject>();
	JSONObject appAutoScaleState = null;
	JSONObject application = null;
	JSONObject autoscalerPolicy = null;
	JSONObject boundApp = null;
	JSONObject metricDBSegment = null;
	JSONObject scalingHistory = null;
	JSONObject triggerRecord = null;
	JSONObject serviceConfig = null;
	JSONObject applicationInstance = null;
	JSONObject serviceInstance = null;
	JSONObject appInstanceMetrics = null;
	private CouchDBDocumentManager(){
		 
		this.initDocuments();
		
	}

	public void initDocuments() {
		appAutoScaleState = new JSONObject("{\"_id\":\"" + APPAUTOSCALESTATEDOCTESTID
				+ "\",\"_rev\":\"1-5aaadd5aeab14c80e40f474ac79218c4\",\"type\":\"AppAutoScaleState\",\"appId\":\""
				+ TESTAPPID
				+ "\",\"instanceCountState\":3,\"lastActionInstanceTarget\":1,\"instanceStepCoolDownEndTime\":0,\"lastActionStartTime\":1459220686013,\"lastActionEndTime\":1459220688081,\"historyId\":\""
				+ SCALINGHISTORYDOCTESTID + "\"}");
		 application = new JSONObject("{\"_id\":\"" + APPLICATIONDOCTESTID
				+ "\",\"_rev\":\"1-7b85e10b68782b6eac543fbf72c841aa\",\"type\":\"Application\",\"appId\":\""
				+ TESTAPPID + "\",\"serviceId\":\"" + TESTSERVICEID
				+ "\",\"bindingId\":\"TESTBINDINGID\",\"appType\":\"java\",\"orgId\":\"" + TESTORGID
				+ "\",\"spaceId\":\"" + TESTSPACEID + "\",\"bindTime\":1459219932743,\"state\":\"enabled\"}");
		 autoscalerPolicy = new JSONObject("{\"_id\":\"" + AUTOSCALERPOLICYDOCTESTID
				+ "\",\"_rev\":\"1-b4be650314e0a51011ab13624d3b03cf\",\"type\":\"AutoScalerPolicy\",\"policyId\":\""
				+ TESTPOLICYID
				+ "\",\"instanceMinCount\":1,\"instanceMaxCount\":5,\"timezone\":\"(GMT +08:00) Asia/Shanghai\",\"policyTriggers\":[{\"metricType\":\"Memory\",\"statType\":\"average\",\"statWindow\":300,\"breachDuration\":600,\"lowerThreshold\":30,\"upperThreshold\":80,\"instanceStepCountDown\":-1,\"instanceStepCountUp\":2,\"stepDownCoolDownSecs\":600,\"stepUpCoolDownSecs\":600,\"startTime\":0,\"endTime\":0,\"startSetNumInstances\":10,\"endSetNumInstances\":10,\"unit\":\"percent\",\"scaleInAdjustment\":null,\"scaleOutAdjustment\":null}],\"scheduledPolicies\":{}}");
		 boundApp = new JSONObject("{\"_id\":\"" + BOUNDAPPDOCTESTID
				+ "\",\"_rev\":\"1-9004adbcb4bf08a2894c92cb26e9954f\",\"type\":\"BoundApp\",\"appId\":\""
				+ TESTAPPID + "\",\"serviceId\":\"" + TESTSERVICEID + "\",\"serverName\":\"AutoScaling\"}");
		 metricDBSegment = new JSONObject("{\"_id\":\"" + METRICDBSEGMENTDOCTESTID
				+ "\",\"_rev\":\"1-8f56ad4bf9cc7c63a71aa1340e25af80\",\"type\":\"MetricDBSegment\",\"startTimestamp\":1458806213698,\"endTimestamp\":1458806214698,\"metricDBPostfix\":\"continuously\",\"segmentSeq\":0,\"serverName\":\"AutoScaling\"}");
		 scalingHistory = new JSONObject("{\"_id\":\"" + SCALINGHISTORYDOCTESTID
				+ "\",\"_rev\":\"1-ad4f534c893fb259a5e150067e0ae47a\",\"type\":\"ScalingHistory\",\"appId\":\""
				+ TESTAPPID
				+ "\",\"status\":3,\"adjustment\":-1,\"instances\":1,\"startTime\":1458875975389,\"endTime\":1458875976357,\"trigger\":{\"metrics\":null,\"threshold\":0,\"thresholdType\":null,\"breachDuration\":0,\"triggerType\":1}}");
		 triggerRecord = new JSONObject("{\"_id\":\"" + TRIGGERRECORDDOCTESTID
				+ "\",\"_rev\":\"1-72e4e42202d5ee07e075710bb4bc5c47\",\"type\":\"TriggerRecord\",\"appName\":\"\",\"appId\":\""
				+ TESTAPPID + "\",\"trigger\":{\"appName\":\"\",\"appId\":\"" + TESTAPPID
				+ "\",\"triggerId\":\"lower\",\"metric\":\"Memory\",\"statWindowSecs\":120,\"breachDurationSecs\":120,\"metricThreshold\":30.0,\"thresholdType\":\"less_than\",\"callbackUrl\":\"http://localhost:9998/events\",\"unit\":\"percent\",\"conditionList\":[],\"statType\":\"avg\"},\"serverName\":\"AutoScaling\"}");
		 serviceConfig = new JSONObject("{\"total_rows\":0,\"offset\":0,\"rows\":[]}");
		 appInstanceMetrics = new JSONObject(
				"{\"_id\":\"" + APPINSTANCEMETRICSDOCTESTID+ "\",\"_rev\":\"1-03c54e3d751336bf5ffe3cb99be8d964\",\"type\":\"AppInstanceMetrics\",\"appId\":\"" + TESTAPPID+ "\",\"appName\":\"ScalingTestApp\",\"appType\":\"java\",\"timestamp\":1458813078531,\"memQuota\":256.0,\"instanceMetrics\":[{\"timestamp\":1458813078531,\"instanceIndex\":0,\"instanceId\":\"0\",\"metrics\":[{\"name\":\"Memory\",\"value\":\"177.5078125\",\"category\":\"cf-stats\",\"group\":\"Memory\",\"timestamp\":1458813078000,\"unit\":\"MB\",\"desc\":null}]}]}");
		 applicationInstance = new JSONObject("{\"_id\":\"" + APPLICATIONINSTANCEDOCTESTID + "\",\"_rev\":\"1-28fe2ae7cd08c99f55f621e75743370a\",\"type\":\"ApplicationInstance_inBroker\",\"bindingId\":\"" + TESTBINDINGID + "\",\"serviceId\":\"" + TESTSERVICEID + "\",\"appId\":\"" + TESTAPPID + "\",\"timestamp\":1456759218805}");
		 serviceInstance = new JSONObject("{\"_id\":\"" + SERVICEINSTANCEDOCTESTID + "\",\"_rev\":\"1-57abaeb827756fc5635abacb88bd4c75\",\"type\":\"ServiceInstance_inBroker\",\"serviceId\":\"" + TESTSERVICEID + "\",\"serverUrl\":\"" + TESTSERVERURL + "\",\"orgId\":\"" + TESTORGID + "\",\"spaceId\":\"" + TESTSPACEID + "\",\"timestamp\":1456800826562}");
		// init basic document
		documentMap.put(APPAUTOSCALESTATEDOCTESTID, appAutoScaleState);
		documentMap.put(APPLICATIONDOCTESTID, application);
		documentMap.put(AUTOSCALERPOLICYDOCTESTID, autoscalerPolicy);
		documentMap.put(BOUNDAPPDOCTESTID, boundApp);
		documentMap.put(METRICDBSEGMENTDOCTESTID, metricDBSegment);
		documentMap.put(SCALINGHISTORYDOCTESTID, scalingHistory);
		documentMap.put(SERVICECONFIGDOCTESTID, serviceConfig);
		documentMap.put(TRIGGERRECORDDOCTESTID, triggerRecord);
		documentMap.put(APPINSTANCEMETRICSDOCTESTID, appInstanceMetrics);
		documentMap.put(APPLICATIONINSTANCEDOCTESTID, applicationInstance);
		documentMap.put(SERVICEINSTANCEDOCTESTID, serviceInstance);
	}
	private JSONObject AppAutoScaleState_ByAppId(){
		JSONObject jo = new JSONObject();
		JSONArray rows = new JSONArray();
		
		JSONObject rowObject = new JSONObject();
		jo.put("total_rows", 3);
		jo.put("offset", 0);
		
		rowObject.put("id", String.valueOf(System.currentTimeMillis()));
		rowObject.put("key", TESTAPPID);
		rowObject.put("value", APPAUTOSCALESTATEDOCTESTID);
		rowObject.put("doc", this.documentMap.get(APPAUTOSCALESTATEDOCTESTID));
		rows.put(rowObject);
		jo.put("rows", rows);
		return jo;
	}
	private JSONObject AppAutoScaleState_byAll(){
		JSONObject jo = new JSONObject();
		JSONArray rows = new JSONArray();
		JSONArray complexKey = new JSONArray();
		JSONObject rowObject = new JSONObject();
		jo.put("total_rows", 3);
		jo.put("offset", 0);
		rowObject = new JSONObject();
		rowObject.put("id", String.valueOf(System.currentTimeMillis()));
		complexKey.put(TESTAPPID);
		complexKey.put(3);
		rowObject.put("key", complexKey);
		rowObject.put("value", APPAUTOSCALESTATEDOCTESTID);
		rowObject.put("doc", this.documentMap.get(APPAUTOSCALESTATEDOCTESTID));
		rows.put(rowObject);
		jo.put("rows", rows);
		return jo;
	}
	private JSONObject Application_ByAppId(){
		/* Application_ByAppId */
		JSONObject jo = new JSONObject();
		JSONArray rows = new JSONArray();
		JSONObject rowObject = new JSONObject();
		jo.put("total_rows", 3);
		jo.put("offset", 0);
		rowObject.put("id", String.valueOf(System.currentTimeMillis()));
		rowObject.put("key", TESTAPPID);
		rowObject.put("value", APPLICATIONDOCTESTID);
		rowObject.put("doc", this.documentMap.get(APPLICATIONDOCTESTID));
		rows.put(rowObject);
		jo.put("rows", rows);
		return jo;
	}
	private JSONObject Application_ByBindingId(){
		/* Application_ByBindingId */
		JSONObject jo = new JSONObject();
		JSONArray rows = new JSONArray();
		JSONObject rowObject = new JSONObject();
		jo.put("total_rows", 3);
		jo.put("offset", 0);
		rowObject.put("id", String.valueOf(System.currentTimeMillis()));
		rowObject.put("key", TESTBINDINGID);
		rowObject.put("value", APPLICATIONDOCTESTID);
		rowObject.put("doc", this.documentMap.get(APPLICATIONDOCTESTID));
		rows.put(rowObject);
		jo.put("rows", rows);
		return jo;
	}
	private JSONObject Application_ByPolicyId(){
		/* Application_ByPolicyId */
		JSONObject jo = new JSONObject();
		JSONArray rows = new JSONArray();
		JSONObject rowObject = new JSONObject();
		jo.put("total_rows", 3);
		jo.put("offset", 0);
		rowObject.put("id", String.valueOf(System.currentTimeMillis()));
		rowObject.put("key", TESTPOLICYID);
		rowObject.put("value", APPLICATIONDOCTESTID);
		rowObject.put("doc", this.documentMap.get(APPLICATIONDOCTESTID));
		rows.put(rowObject);
		jo.put("rows", rows);
		return jo;
	}
	private JSONObject Application_ByServiceId(){
		/* Application_ByServiceId */
		JSONObject jo = new JSONObject();
		JSONArray rows = new JSONArray();
		JSONObject rowObject = new JSONObject();
		jo.put("total_rows", 3);
		jo.put("offset", 0);
		rowObject.put("id", String.valueOf(System.currentTimeMillis()));
		rowObject.put("key", TESTSERVICEID);
		rowObject.put("value", APPLICATIONDOCTESTID);
		rowObject.put("doc", this.documentMap.get(APPLICATIONDOCTESTID));
		rows.put(rowObject);
		jo.put("rows", rows);
		return jo;
	}
	private JSONObject Application_ByServiceId_State(){
		/* Application_ByServiceId_State */
		JSONObject jo = new JSONObject();
		JSONArray rows = new JSONArray();
		JSONObject rowObject = new JSONObject();
		jo.put("total_rows", 3);
		jo.put("offset", 0);
		rowObject.put("id", String.valueOf(System.currentTimeMillis()));
		rowObject.put("key", TESTSERVICEID);
		rowObject.put("value", APPLICATIONDOCTESTID);
		rowObject.put("doc", this.documentMap.get(APPLICATIONDOCTESTID));
		rows.put(rowObject);
		jo.put("rows", rows);
		return jo;
	}
	private JSONObject Application_byAll(){
		JSONObject jo = new JSONObject();
		JSONArray rows = new JSONArray();
		JSONArray complexKey = new JSONArray();
		JSONObject rowObject = new JSONObject();
		/* Application_byAll */
		jo.put("total_rows", 3);
		jo.put("offset", 0);
		rowObject.put("id", String.valueOf(System.currentTimeMillis()));
		complexKey.put(TESTAPPID);
		complexKey.put(TESTSERVICEID);
		complexKey.put(TESTBINDINGID);
		complexKey.put(TESTPOLICYID);
		complexKey.put(BOUNDSTATE);
		rowObject.put("key", complexKey);
		rowObject.put("value", APPLICATIONDOCTESTID);
		rowObject.put("doc", this.documentMap.get(APPLICATIONDOCTESTID));
		rows.put(rowObject);
		jo.put("rows", rows);
		return jo;
		
	}
	private JSONObject AutoScalerPolicy_ByPolicyId(){
		/* AutoScalerPolicy_ByPolicyId */
		JSONObject jo = new JSONObject();
		JSONArray rows = new JSONArray();
		JSONObject rowObject = new JSONObject();		
		jo.put("total_rows", 3);
		jo.put("offset", 0);
		rowObject.put("id", String.valueOf(System.currentTimeMillis()));
		rowObject.put("key", TESTPOLICYID);
		rowObject.put("value", AUTOSCALERPOLICYDOCTESTID);
		rowObject.put("doc", this.documentMap.get(AUTOSCALERPOLICYDOCTESTID));
		rows.put(rowObject);
		jo.put("rows", rows);
		return jo;
	}
	private JSONObject AutoScalerPolicy_byAll(){
		/* AutoScalerPolicy_byAll */
		JSONObject jo = new JSONObject();
		JSONArray rows = new JSONArray();
		JSONArray complexKey = new JSONArray();
		JSONObject rowObject = new JSONObject();
		jo.put("total_rows", 3);
		jo.put("offset", 0);
		rowObject.put("id", String.valueOf(System.currentTimeMillis()));
		complexKey.put(TESTPOLICYID);
		complexKey.put(1);
		complexKey.put(5);
		rowObject.put("key", complexKey);
		rowObject.put("value", AUTOSCALERPOLICYDOCTESTID);
		rowObject.put("doc", this.documentMap.get(AUTOSCALERPOLICYDOCTESTID));
		rows.put(rowObject);
		jo.put("rows", rows);
		return jo;
	}
	private JSONObject BoundApp_ByAppId(){
		/* BoundApp_ByAppId */
		JSONObject jo = new JSONObject();
		JSONArray rows = new JSONArray();
		JSONArray complexKey = new JSONArray();
		JSONObject rowObject = new JSONObject();
		jo.put("total_rows", 3);
		jo.put("offset", 0);
		rowObject.put("id", String.valueOf(System.currentTimeMillis()));
		complexKey.put(TESTAPPID);
		rowObject.put("key", complexKey);
		rowObject.put("value", BOUNDAPPDOCTESTID);
		rowObject.put("doc", this.documentMap.get(BOUNDAPPDOCTESTID));
		rows.put(rowObject);
		jo.put("rows", rows);
		return jo;
	}
	private JSONObject BoundApp_ByServerName(){
		/* BoundApp_ByServerName */
		JSONObject jo = new JSONObject();
		JSONArray rows = new JSONArray();
		JSONArray complexKey = new JSONArray();
		JSONObject rowObject = new JSONObject();
		jo.put("total_rows", 3);
		jo.put("offset", 0);
		rowObject.put("id", String.valueOf(System.currentTimeMillis()));
		complexKey.put(SERVERNAME);
		rowObject.put("key", complexKey);
		rowObject.put("value", BOUNDAPPDOCTESTID);
		rowObject.put("doc", this.documentMap.get(BOUNDAPPDOCTESTID));
		rows.put(rowObject);
		jo.put("rows", rows);
		return jo;
	}
	private JSONObject BoundApp_ByServiceId(){
		/* BoundApp_ByServiceId */
		JSONObject jo = new JSONObject();
		JSONArray rows = new JSONArray();
		JSONArray complexKey = new JSONArray();
		JSONObject rowObject = new JSONObject();
		jo.put("total_rows", 3);
		jo.put("offset", 0);
		rowObject.put("id", String.valueOf(System.currentTimeMillis()));
		complexKey.put(TESTSERVICEID);
		rowObject.put("key", complexKey);
		rowObject.put("value", BOUNDAPPDOCTESTID);
		rowObject.put("doc", this.documentMap.get(BOUNDAPPDOCTESTID));
		rows.put(rowObject);
		jo.put("rows", rows);
		return jo;
	}
	private JSONObject BoundApp_ByServiceId_AppId(){
		/* BoundApp_ByServiceId_AppId */
		JSONObject jo = new JSONObject();
		JSONArray rows = new JSONArray();
		JSONArray complexKey = new JSONArray();
		JSONObject rowObject = new JSONObject();
		jo.put("total_rows", 3);
		jo.put("offset", 0);
		rowObject.put("id", String.valueOf(System.currentTimeMillis()));
		complexKey.put(TESTSERVICEID);
		complexKey.put(TESTAPPID);
		rowObject.put("key", complexKey);
		rowObject.put("value", BOUNDAPPDOCTESTID);
		rowObject.put("doc", this.documentMap.get(BOUNDAPPDOCTESTID));
		rows.put(rowObject);
		jo.put("rows", rows);
		return jo;
	}
	private JSONObject BoundApp_byAll(){
		/* BoundApp_byAll */
		JSONObject jo = new JSONObject();
		JSONArray rows = new JSONArray();
		JSONArray complexKey = new JSONArray();
		JSONObject rowObject = new JSONObject();
		jo.put("total_rows", 3);
		jo.put("offset", 0);
		rowObject.put("id", String.valueOf(System.currentTimeMillis()));
		complexKey.put(TESTAPPID);
		complexKey.put(SERVERNAME);
		rowObject.put("key", complexKey);
		rowObject.put("value", BOUNDAPPDOCTESTID);
		rowObject.put("doc", this.documentMap.get(BOUNDAPPDOCTESTID));
		rows.put(rowObject);
		jo.put("rows", rows);
		return jo;
	}
	private JSONObject MetricDBSegment_ByPostfix(){
		/* MetricDBSegment_ByPostfix */
		JSONObject jo = new JSONObject();
		JSONArray rows = new JSONArray();
		JSONObject rowObject = new JSONObject();
		jo.put("total_rows", 3);
		jo.put("offset", 0);
		rowObject.put("id", String.valueOf(System.currentTimeMillis()));
		rowObject.put("key", "continuously");
		rowObject.put("value", METRICDBSEGMENTDOCTESTID);
		rowObject.put("doc", this.documentMap.get(METRICDBSEGMENTDOCTESTID));
		rows.put(rowObject);
		jo.put("rows", rows);
		return jo;
	}
	private JSONObject MetricDBSegment_ByServerName(){
		/* MetricDBSegment_ByServerName */
		JSONObject jo = new JSONObject();
		JSONArray rows = new JSONArray();
		JSONObject rowObject = new JSONObject();
		jo.put("total_rows", 3);
		jo.put("offset", 0);
		rowObject.put("id", String.valueOf(System.currentTimeMillis()));
		rowObject.put("key", SERVERNAME);
		rowObject.put("value", METRICDBSEGMENTDOCTESTID);
		rowObject.put("doc", this.documentMap.get(METRICDBSEGMENTDOCTESTID));
		rows.put(rowObject);
		jo.put("rows", rows);
		return jo;
	}
	private JSONObject MetricDBSegment_ByServerName_SegmentSeq(){
		/* MetricDBSegment_ByServerName_SegmentSeq */
		JSONObject jo = new JSONObject();
		JSONArray rows = new JSONArray();
		JSONArray complexKey = new JSONArray();
		JSONObject rowObject = new JSONObject();
		jo.put("total_rows", 3);
		jo.put("offset", 0);
		rowObject.put("id", String.valueOf(System.currentTimeMillis()));
		complexKey.put(SERVERNAME);
		complexKey.put(0);
		rowObject.put("key", complexKey);
		rowObject.put("value", METRICDBSEGMENTDOCTESTID);
		rowObject.put("doc", this.documentMap.get(METRICDBSEGMENTDOCTESTID));
		rows.put(rowObject);
		jo.put("rows", rows);
		return jo;
	}
	private JSONObject MetricDBSegment_byAll(){
		/* MetricDBSegment_byAll */
		JSONObject jo = new JSONObject();
		JSONArray rows = new JSONArray();
		JSONArray complexKey = new JSONArray();
		JSONObject rowObject = new JSONObject();
		jo.put("total_rows", 3);
		jo.put("offset", 0);
		rowObject.put("id", String.valueOf(System.currentTimeMillis()));
		complexKey.put("continuously");
		complexKey.put(SERVERNAME);
		complexKey.put(0);
		rowObject.put("key", complexKey);
		rowObject.put("value", METRICDBSEGMENTDOCTESTID);
		rowObject.put("doc", this.documentMap.get(METRICDBSEGMENTDOCTESTID));
		rows.put(rowObject);
		jo.put("rows", rows);
		return jo;
	}
	private JSONObject ScalingHistory_ByScalingTime(){
		/* ScalingHistory_ByScalingTime */
		JSONObject jo = new JSONObject();
		JSONArray rows = new JSONArray();
		JSONArray complexKey = new JSONArray();
		JSONObject rowObject = new JSONObject();
		jo.put("total_rows", 3);
		jo.put("offset", 0);
		rowObject.put("id", String.valueOf(System.currentTimeMillis()));
		complexKey.put(TESTAPPID);
		complexKey.put(System.currentTimeMillis());
		rowObject.put("key", complexKey);
		rowObject.put("value", SCALINGHISTORYDOCTESTID);
		rowObject.put("doc", this.documentMap.get(SCALINGHISTORYDOCTESTID));
		rows.put(rowObject);
		jo.put("rows", rows);
		return jo;
	}
	private JSONObject ScalingHistory_byAll(){
		/* ScalingHistory_byAll */
		JSONObject jo = new JSONObject();
		JSONArray rows = new JSONArray();
		JSONArray complexKey = new JSONArray();
		JSONObject rowObject = new JSONObject();
		jo.put("total_rows", 3);
		jo.put("offset", 0);
		rowObject.put("id", String.valueOf(System.currentTimeMillis()));
		complexKey.put(TESTAPPID);
		complexKey.put(3);
		rowObject.put("key", complexKey);
		rowObject.put("value", SCALINGHISTORYDOCTESTID);
		rowObject.put("doc", this.documentMap.get(SCALINGHISTORYDOCTESTID));
		rows.put(rowObject);
		jo.put("rows", rows);
		return jo;
	}
	private JSONObject TriggerRecord_ByAppId(){
		/* TriggerRecord_ByAppId */
		JSONObject jo = new JSONObject();
		JSONArray rows = new JSONArray();
		JSONArray complexKey = new JSONArray();
		JSONObject rowObject = new JSONObject();
		jo.put("total_rows", 3);
		jo.put("offset", 0);
		rowObject.put("id", String.valueOf(System.currentTimeMillis()));
		complexKey.put(TESTAPPID);
		rowObject.put("key", complexKey);
		rowObject.put("value", TRIGGERRECORDDOCTESTID);
		rowObject.put("doc", this.documentMap.get(TRIGGERRECORDDOCTESTID));
		rows.put(rowObject);
		jo.put("rows", rows);
		return jo;
	}
	private JSONObject TriggerRecord_ByServerName(){
		/* TriggerRecord_ByServerName */
		JSONObject jo = new JSONObject();
		JSONArray rows = new JSONArray();
		JSONArray complexKey = new JSONArray();
		JSONObject rowObject = new JSONObject();
		jo.put("total_rows", 3);
		jo.put("offset", 0);
		rowObject.put("id", String.valueOf(System.currentTimeMillis()));
		complexKey.put(SERVERNAME);
		rowObject.put("key", complexKey);
		rowObject.put("value", TRIGGERRECORDDOCTESTID);
		rowObject.put("doc", this.documentMap.get(TRIGGERRECORDDOCTESTID));
		rows.put(rowObject);
		jo.put("rows", rows);
		return jo;
	}
	private JSONObject TriggerRecord_byAll(){
		/* TriggerRecord_byAll */
		JSONObject jo = new JSONObject();
		JSONArray rows = new JSONArray();
		JSONArray complexKey = new JSONArray();
		JSONObject rowObject = new JSONObject();
		
		jo.put("total_rows", 3);
		jo.put("offset", 0);
		
		rowObject.put("id", String.valueOf(System.currentTimeMillis()));
		
		complexKey.put(TESTAPPID);
		complexKey.put(SERVERNAME);
		rowObject.put("key", complexKey);
		rowObject.put("value", TRIGGERRECORDDOCTESTID);
		rowObject.put("doc", this.documentMap.get(TRIGGERRECORDDOCTESTID));
		rows.put(rowObject);
		jo.put("rows", rows);
		return jo;
	}
	private JSONObject AppInstanceMetrics_ByAppId(){
		/* AppInstanceMetrics_ByAppId */
		JSONObject jo = new JSONObject();
		JSONArray rows = new JSONArray();
		JSONArray complexKey = new JSONArray();
		JSONObject rowObject = new JSONObject();
		
		jo.put("total_rows", 3);
		jo.put("offset", 0);
		
		rowObject.put("id", String.valueOf(System.currentTimeMillis()));
		
		complexKey.put(TESTAPPID);
		rowObject.put("key", complexKey);
		rowObject.put("value", APPINSTANCEMETRICSDOCTESTID);
		rowObject.put("doc", this.documentMap.get(APPINSTANCEMETRICSDOCTESTID));
		rows.put(rowObject);
		jo.put("rows", rows);
		return jo;
	}
	private JSONObject AppInstanceMetrics_ByAppIdBetween(){
		/* AppInstanceMetrics_ByAppIdBetween */
		JSONObject jo = new JSONObject();
		JSONArray rows = new JSONArray();
		JSONArray complexKey = new JSONArray();
		JSONObject rowObject = new JSONObject();
		
		jo.put("total_rows", 3);
		jo.put("offset", 0);
		
		rowObject.put("id", String.valueOf(System.currentTimeMillis()));
		
		complexKey.put(TESTAPPID);
		complexKey.put(System.currentTimeMillis());
		rowObject.put("key", complexKey);
		rowObject.put("value", APPINSTANCEMETRICSDOCTESTID);
		rowObject.put("doc", this.documentMap.get(APPINSTANCEMETRICSDOCTESTID));
		rows.put(rowObject);
		jo.put("rows", rows);
		return jo;
	}
	private JSONObject AppInstanceMetrics_ByServiceId(){
		/* AppInstanceMetrics_ByServiceId */
		JSONObject jo = new JSONObject();
		JSONArray rows = new JSONArray();
		JSONArray complexKey = new JSONArray();
		JSONObject rowObject = new JSONObject();
		jo.put("total_rows", 3);
		jo.put("offset", 0);
		
		rowObject.put("id", String.valueOf(System.currentTimeMillis()));
		
		complexKey.put(TESTSERVICEID);
		complexKey.put(System.currentTimeMillis());
		rowObject.put("key", complexKey);
		rowObject.put("value", APPINSTANCEMETRICSDOCTESTID);
		rowObject.put("doc", this.documentMap.get(APPINSTANCEMETRICSDOCTESTID));
		rows.put(rowObject);
		jo.put("rows", rows);
		return jo;
	}
	private JSONObject AppInstanceMetrics_byAll(){
		/* AppInstanceMetrics_byAll */
		JSONObject jo = new JSONObject();
		JSONArray rows = new JSONArray();
		JSONArray complexKey = new JSONArray();
		JSONObject rowObject = new JSONObject();
		
		jo.put("total_rows", 3);
		jo.put("offset", 0);
		
		rowObject.put("id", String.valueOf(System.currentTimeMillis()));
		
		complexKey.put(TESTAPPID);
		complexKey.put("java");
		complexKey.put(System.currentTimeMillis());
		rowObject.put("key", complexKey);
		rowObject.put("value", APPINSTANCEMETRICSDOCTESTID);
		rowObject.put("doc", this.documentMap.get(APPINSTANCEMETRICSDOCTESTID));
		rows.put(rowObject);
		jo.put("rows", rows);
		return jo;
	}
	private JSONObject ApplicationInstance_ByAppId(){
		/* ApplicationInstance_ByAppId */
		JSONObject jo = new JSONObject();
		JSONArray rows = new JSONArray();
		
		JSONObject rowObject = new JSONObject();
		
		jo.put("total_rows", 3);
		jo.put("offset", 0);
		
		rowObject.put("id", String.valueOf(System.currentTimeMillis()));
		rowObject.put("key", TESTAPPID);
		rowObject.put("value", APPLICATIONINSTANCEDOCTESTID);
		rowObject.put("doc", this.documentMap.get(APPLICATIONINSTANCEDOCTESTID));
		rows.put(rowObject);
		jo.put("rows", rows);
		return jo;
	}
	private JSONObject ApplicationInstance_ByBindingId(){
		/* ApplicationInstance_ByBindingId */
		JSONObject jo = new JSONObject();
		JSONArray rows = new JSONArray();
		
		JSONObject rowObject = new JSONObject();
		
		jo.put("total_rows", 3);
		jo.put("offset", 0);
		
		rowObject.put("id", String.valueOf(System.currentTimeMillis()));
		rowObject.put("key", TESTBINDINGID);
		rowObject.put("value", APPLICATIONINSTANCEDOCTESTID);
		rowObject.put("doc", this.documentMap.get(APPLICATIONINSTANCEDOCTESTID));
		rows.put(rowObject);
		jo.put("rows", rows);
		return jo;
	}
	private JSONObject ApplicationInstance_ByServiceId(){
		/* ApplicationInstance_ByServiceId */
		JSONObject jo = new JSONObject();
		JSONArray rows = new JSONArray();		
		JSONObject rowObject = new JSONObject();
		jo.put("total_rows", 3);
		jo.put("offset", 0);
		rowObject.put("id", String.valueOf(System.currentTimeMillis()));
		rowObject.put("key", TESTSERVICEID);
		rowObject.put("value", APPLICATIONINSTANCEDOCTESTID);
		rowObject.put("doc", this.documentMap.get(APPLICATIONINSTANCEDOCTESTID));
		rows.put(rowObject);
		jo.put("rows", rows);
		return jo;
	}
	private JSONObject ServiceInstance_ByServerURL_noDocs(){
		/* ServiceInstance_ByServerURL_noDocs */
		JSONObject jo = new JSONObject();
		JSONArray rows = new JSONArray();
		
		JSONObject rowObject = new JSONObject();
		
		jo.put("total_rows", 3);
		jo.put("offset", 0);
		
		rowObject.put("id", String.valueOf(System.currentTimeMillis()));
		rowObject.put("key", TESTSERVERURL);
		rowObject.put("value", SERVICEINSTANCEDOCTESTID);
//		rowObject.put("doc", serviceInstance);
		rows.put(rowObject);
		jo.put("rows", rows);
		return jo;
	}
	private JSONObject ServiceInstance_ByServerURL(){
		/* ServiceInstance_ByServerURL */
		JSONObject jo = new JSONObject();
		JSONArray rows = new JSONArray();
		
		JSONObject rowObject = new JSONObject();
		
		jo.put("total_rows", 3);
		jo.put("offset", 0);
		
		rowObject.put("id", String.valueOf(System.currentTimeMillis()));
		rowObject.put("key", TESTSERVERURL);
		rowObject.put("value", SERVICEINSTANCEDOCTESTID);
		rowObject.put("doc", this.documentMap.get(SERVICEINSTANCEDOCTESTID));
		rows.put(rowObject);
		jo.put("rows", rows);
		return jo;
	}
	private JSONObject ServiceInstance_ByServiceId(){
		/* ServiceInstance_ByServiceId */
		JSONObject jo = new JSONObject();
		JSONArray rows = new JSONArray();
		
		JSONObject rowObject = new JSONObject();
		
		jo.put("total_rows", 3);
		jo.put("offset", 0);
		
		rowObject.put("id", String.valueOf(System.currentTimeMillis()));
		rowObject.put("key", TESTSERVICEID);
		rowObject.put("value", SERVICEINSTANCEDOCTESTID);
		rowObject.put("doc", this.documentMap.get(SERVICEINSTANCEDOCTESTID));
		rows.put(rowObject);
		jo.put("rows", rows);
		return jo;
	}
	public JSONObject getDocument(String docId,String includeDocs) {
		System.out.println("get docId=" + docId + ",includeDocs=" + includeDocs);
		JSONObject jo = null;
		switch(docId){
		case "AppAutoScaleState_byAll":
			jo = this.AppAutoScaleState_byAll();
			break;
		case "AppAutoScaleState_ByAppId":
			jo = this.AppAutoScaleState_ByAppId();
			break;
		case "Application_ByAppId":
			jo = this.Application_ByAppId();
			break;
		case "Application_ByBindingId":
			jo = this.Application_ByBindingId();
			break;
		case "Application_ByPolicyId":
			jo = this.Application_ByPolicyId();
			break;
		case "Application_ByServiceId":
			jo = this.Application_ByServiceId();
			break;
		case "Application_ByServiceId_State":
			jo = this.Application_ByServiceId_State();
			break;
		case "Application_byAll":
			jo = this.Application_byAll();
			break;
		case "AutoScalerPolicy_ByPolicyId":
			jo = this.AutoScalerPolicy_ByPolicyId();
			break;
		case "AutoScalerPolicy_byAll":
			jo = this.AutoScalerPolicy_byAll();
			break;
		case "BoundApp_ByAppId":
			jo = this.BoundApp_ByAppId();
			break;
		case "BoundApp_ByServerName":
			jo = this.BoundApp_ByServerName();
			break;
		case "BoundApp_ByServiceId":
			jo = this.BoundApp_ByServiceId();
			break;
		case "BoundApp_ByServiceId_AppId":
			jo = this.BoundApp_ByServiceId_AppId();
			break;
		case "BoundApp_byAll":
			jo = this.BoundApp_byAll();
			break;
		case "MetricDBSegment_ByPostfix":
			jo = this.MetricDBSegment_ByPostfix();
			break;
		case "MetricDBSegment_ByServerName":
			jo = this.MetricDBSegment_ByServerName();
			break;
		case "MetricDBSegment_ByServerName_SegmentSeq":
			jo = this.MetricDBSegment_ByServerName_SegmentSeq();
			break;
		case "MetricDBSegment_byAll":
			jo = this.MetricDBSegment_byAll();
			break;
		case "ScalingHistory_ByScalingTime":
			jo = this.ScalingHistory_ByScalingTime();
			break;
		case "ScalingHistory_byAll":
			jo = this.ScalingHistory_byAll();
			break;
		case "TriggerRecord_ByAppId":
			jo = this.TriggerRecord_ByAppId();
			break;
		case "TriggerRecord_ByServerName":
			jo = this.TriggerRecord_ByServerName();
			break;
		case "TriggerRecord_byAll":
			jo = this.TriggerRecord_byAll();
			break;
		case "AppInstanceMetrics_ByAppId":
			jo = this.AppInstanceMetrics_ByAppId();
			break;
		case "AppInstanceMetrics_ByAppIdBetween":
			jo = this.AppInstanceMetrics_ByAppIdBetween();
			break;
		case "AppInstanceMetrics_ByServiceId":
			jo = this.AppInstanceMetrics_ByServiceId();
			break;
		case "AppInstanceMetrics_byAll":
			jo = this.AppInstanceMetrics_byAll();
			break;
		case "ApplicationInstance_ByAppId":
			jo = this.ApplicationInstance_ByAppId();
			break;
		case "ApplicationInstance_ByBindingId":
			jo = this.ApplicationInstance_ByBindingId();
			break;
		case "ApplicationInstance_ByServiceId":
			jo = this.ApplicationInstance_ByServiceId();
			break;
		case "ServiceInstance_ByServerURL":
			if("true".equals(includeDocs)){
				
				jo = this.ServiceInstance_ByServerURL();
			}
			else{
				jo = this.ServiceInstance_ByServerURL_noDocs();
			}
			
			break;
		case "ServiceInstance_ByServiceId":
			jo = this.ServiceInstance_ByServiceId();
			break;
		default:
			jo = this.documentMap.get(docId);
			break;
		}
		
		if(null == jo || ("true".equals(includeDocs) && jo.getJSONArray("rows").length() > 0 && !jo.getJSONArray("rows").getJSONObject(0).has("doc"))){
			JSONObject res = new JSONObject("{\"total_rows\":13,\"offset\":5,\"rows\":[]}");
			
			if(docId.startsWith("123456")){
				res = new JSONObject("{\"error\":\"not_found\",\"reason\":\"missing\"}");
			}
			return res;
		}
		else{
			return jo;
		}

	}
	public JSONObject addDocument(String jsonStr){
		JSONObject jo = new JSONObject(jsonStr);
		String docType = jo.getString("type");
		JSONObject doc = null;
		switch(docType){
		case "AppAutoScaleState":
			this.documentMap.put(APPAUTOSCALESTATEDOCTESTID,appAutoScaleState);
			doc = this.documentMap.get(APPAUTOSCALESTATEDOCTESTID);
			break;
		case "Application":
			this.documentMap.put(APPLICATIONDOCTESTID,application);
			doc = this.documentMap.get(APPLICATIONDOCTESTID);
			break;
		case "AutoScalerPolicy":
			this.documentMap.put(AUTOSCALERPOLICYDOCTESTID,autoscalerPolicy);
			doc = this.documentMap.get(AUTOSCALERPOLICYDOCTESTID);
			break;
		case "BoundApp":
			this.documentMap.put(BOUNDAPPDOCTESTID,boundApp);
			doc = this.documentMap.get(BOUNDAPPDOCTESTID);
			break;
		case "MetricDBSegment":
			this.documentMap.put(METRICDBSEGMENTDOCTESTID,metricDBSegment);
			doc = this.documentMap.get(METRICDBSEGMENTDOCTESTID);
			break;
		case "ScalingHistory":
			this.documentMap.put(SCALINGHISTORYDOCTESTID,scalingHistory);
			doc = this.documentMap.get(SCALINGHISTORYDOCTESTID);
			break;
		case "TriggerRecord":
			this.documentMap.put(TRIGGERRECORDDOCTESTID,triggerRecord);
			doc = this.documentMap.get(TRIGGERRECORDDOCTESTID);
			break;
		case "ServiceConfig":
			this.documentMap.put(SERVICECONFIGDOCTESTID,serviceConfig);
			doc = this.documentMap.get(SERVICECONFIGDOCTESTID);
			break;
		case "AppInstanceMetrics":
			this.documentMap.put(APPINSTANCEMETRICSDOCTESTID,appInstanceMetrics);
			doc = this.documentMap.get(APPINSTANCEMETRICSDOCTESTID);
			break;
		case "ApplicationInstance_inBroker":
			this.documentMap.put(APPLICATIONINSTANCEDOCTESTID,applicationInstance);
			doc = this.documentMap.get(APPLICATIONINSTANCEDOCTESTID);
			break;
		case "ServiceInstance_inBroker":
			this.documentMap.put(SERVICEINSTANCEDOCTESTID,serviceInstance);
			doc = this.documentMap.get(SERVICEINSTANCEDOCTESTID);
			break;
			
		}
		JSONObject result = new JSONObject();
		result.put("ok", true);
		result.put("id", doc.get("_id"));
		result.put("rev", doc.get("_rev"));
		return result;
	}
	public JSONObject updateDocument(String docId,String jsonStr){
		JSONObject jo = this.documentMap.get(docId);
		JSONObject result = new JSONObject();
		if(jo == null){
			result = this.addDocument(jsonStr);
		}
		else{
			JSONObject newObj = new JSONObject(jsonStr);
			for(String key : newObj.keySet()){
				if(!"".equals(key) && !"".equals(key)){
					jo.put(key, newObj.get(key));
				}
			}
			this.documentMap.put(docId, jo);
			String rev = jo.getString("_rev");
			String newRev = (Integer.valueOf(rev.substring(0, 1)) + 1) + rev.substring(1, rev.length());
			this.documentMap.get(docId).put("_rev", newRev);
			String id = jo.getString("_id");
			rev = jo.getString("_rev");
			
			result.put("id", id);
			result.put("rev", rev);
			result.put("ok", true);
		}
		
		return result;
	}
	
	public JSONObject deleteDocument(String docId){
		System.out.println("delete docId=" + docId);;
		JSONObject jo = this.documentMap.get(docId);
		String id = jo.getString("_id");
		String rev = jo.getString("_rev");
		this.documentMap.remove(docId);
		JSONObject result = new JSONObject();
		result.put("id", id);
		result.put("rev", rev);
		result.put("ok", true);
		return result;
	}

}
