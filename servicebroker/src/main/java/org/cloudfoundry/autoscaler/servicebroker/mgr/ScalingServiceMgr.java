package org.cloudfoundry.autoscaler.servicebroker.mgr;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.Iterator;
import java.util.List;
import java.util.Map;
import java.util.UUID;

import javax.ws.rs.core.MediaType;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.common.util.ConfigManager;
import org.cloudfoundry.autoscaler.servicebroker.Constants;
import org.cloudfoundry.autoscaler.servicebroker.data.entity.ApplicationInstance;
import org.cloudfoundry.autoscaler.servicebroker.data.entity.ServiceInstance;
import org.cloudfoundry.autoscaler.servicebroker.data.storeservice.IDataStoreService;
import org.cloudfoundry.autoscaler.servicebroker.exception.AlreadyBoundAnotherServiceException;
import org.cloudfoundry.autoscaler.servicebroker.exception.ScalingServerFailureException;
import org.cloudfoundry.autoscaler.servicebroker.exception.ServerUrlMappingNotFoundException;
import org.cloudfoundry.autoscaler.servicebroker.exception.ServiceBindingNotFoundException;
import org.cloudfoundry.autoscaler.servicebroker.restclient.IRestClient;
import org.cloudfoundry.autoscaler.servicebroker.restclient.RestClientJersey;
import org.cloudfoundry.autoscaler.servicebroker.util.DataSourceUtil;
import org.json.JSONException;
import org.json.JSONObject;

import com.sun.jersey.api.client.ClientResponse;

public class ScalingServiceMgr {

	private static final Logger logger = Logger.getLogger(ScalingServiceMgr.class);
	private static final ScalingServiceMgr instance = new ScalingServiceMgr();

	private IDataStoreService dataService = DataSourceUtil.getStoreProvider();
	private IRestClient restClient = RestClientJersey.getInstance();

	private List<String> serverUrlList = new ArrayList<String>();
	private String apiServerUrl = "";

	public static ScalingServiceMgr getInstance() {
		return instance;
	}

	private ScalingServiceMgr() {
		initServerUrlList();
	}

	public String createService(String serviceId, String orgId, String spaceId) {
		String serverUrl = this.selectServer();
		dataService.createService(serviceId, serverUrl, orgId, spaceId);
		return serverUrl + Constants.DASHBOARD_URL;

	}

	public JSONObject bindService(String appId, String serviceId, String bindingId)
			throws AlreadyBoundAnotherServiceException, ServerUrlMappingNotFoundException,
			ScalingServerFailureException {

		JSONObject credentials = null;

		if (!alreadyHasOtherServiceBound(appId, serviceId, bindingId)) {
			String serverUrl = getServerUrl(serviceId);
			credentials = registerAppWithServer(appId, serviceId, bindingId, serverUrl);
			dataService.bindApplication(appId, serviceId, bindingId);
		}

		return credentials;

	}

	public void unbindService(String serviceId, String bindingId)
			throws ServiceBindingNotFoundException, ScalingServerFailureException {

		String appId = null;
		ApplicationInstance application = dataService.getBoundAppByBindingId(bindingId);

		if (application != null) {
			appId = application.getAppId();
		} else {
			throw new ServiceBindingNotFoundException();
		}

		try {
			String serverUrl = getServerUrl(serviceId);
			unregisterAppWithServer(appId, serviceId, bindingId, serverUrl);
		} catch (ServerUrlMappingNotFoundException e) {
			// ignore as we will move the mapping in the end of this function
		} catch (ScalingServerFailureException e) {
			if (ConfigManager.getBoolean("forceRemove", false))
				logger.error("unbind FORCE REMOVE with error on service " + serviceId + " and binding id" + bindingId
						+ " ,error: " + e.getMessage(), e);
			else
				throw e;
		}
		dataService.unbindApplication(bindingId);
	}

	public void deprovisionService(String serviceId) throws ServiceBindingNotFoundException {
		ServiceInstance service = dataService.getServiceInstanceByServiceId(serviceId);
		if (service != null)
			dataService.deleteService(serviceId);
		else
			throw new ServiceBindingNotFoundException();

	}

	private void initServerUrlList() {
		serverUrlList.clear();
		String defaultHttpProtocol = ConfigManager.get(Constants.CONFIG_ENTRY_HTTP_PROTOCOL, "http");
		String[] serverList = ConfigManager.get(Constants.CONFIG_ENTRY_SERVER_URI_LIST).split(",");
		for (String server : serverList) {
			if (!(server.startsWith("https://") || server.startsWith("http://"))) {
				serverUrlList.add(defaultHttpProtocol + "://" + server.trim());
			} else {
				serverUrlList.add(server.trim());
			}
		}

		String[] apiList = ConfigManager.get(Constants.CONFIG_ENTRY_API_SERVER_URI).split(",");
		apiServerUrl = apiList[0].trim();
		if (!(apiServerUrl.startsWith("https://") || apiServerUrl.startsWith("http://")))
			apiServerUrl = defaultHttpProtocol + "://" + apiServerUrl;

	}

	private String getServerUrl(String serviceId) throws ServerUrlMappingNotFoundException {
		logger.debug("Getting server URL for " + serviceId);
		ServiceInstance service = dataService.getServiceInstanceByServiceId(serviceId);
		if (service != null)
			return service.getServerUrl();
		else
			throw new ServerUrlMappingNotFoundException();
	}

	private String getAPIServiceUrl() {
		return apiServerUrl;
	}

	private String selectServer() {
		String selectedServer = null;
		int SmallestlinkedSize = Integer.MAX_VALUE;
		for (String serverUrl : serverUrlList) {
			int linkedServiceCount = dataService.getWorkloadSummaryByServerURL(serverUrl);
			if (SmallestlinkedSize > linkedServiceCount) {
				SmallestlinkedSize = linkedServiceCount;
				selectedServer = serverUrl;
			}
		}
		return selectedServer;
	}

	private boolean alreadyHasOtherServiceBound(String appId, String serviceId, String bindingId)
			throws AlreadyBoundAnotherServiceException {

		List<ApplicationInstance> applications = dataService.getBoundAppByAppId(appId);
		if (applications != null && applications.size() > 0) {
			// if there are binding record for the same application, we need to check whether previous binding happened
			// with the same app and same service, but just the different binding id
			Iterator<ApplicationInstance> iter = applications.iterator();
			while (iter.hasNext()) {
				ApplicationInstance application = iter.next();
				if (application == null || (application.getServiceId().equalsIgnoreCase(serviceId)
						&& application.getBindingId().equalsIgnoreCase(bindingId))) {
					iter.remove();
					continue;
				}
			}
		}
		// after the validation, if the applications array becomes empty, then return false.
		if (applications == null || applications.size() == 0) {
			return false;
		} else {
			throw new AlreadyBoundAnotherServiceException();
		}
	}

	private JSONObject registerAppWithServer(String appId, String serviceId, String bindingId, String serverUrl)
			throws ScalingServerFailureException {

		String registerUrl = serverUrl + Constants.REGISTER_APP_URL;

		JSONObject registerJSON = new JSONObject();
		registerJSON.put("appId", appId);
		registerJSON.put("serviceId", serviceId);
		registerJSON.put("bindingId", bindingId);
		registerJSON.put("agentUsername", "agent");
		registerJSON.put("agentPassword", UUID.randomUUID().toString());

		ServiceInstance serviceInstance = dataService.getServiceInstanceByServiceId(serviceId);
		if (serviceInstance != null) {
			if (serviceInstance.getOrgId() != null && !serviceInstance.getOrgId().isEmpty()) {
				registerJSON.put("orgId", serviceInstance.getOrgId());
			}

			if (serviceInstance.getSpaceId() != null && !serviceInstance.getSpaceId().isEmpty()) {
				registerJSON.put("spaceId", serviceInstance.getSpaceId());
			}
		}

		logger.info("registerUrl is  " + registerUrl + " with appId " + appId);
		logger.info("registerJSON is  " + registerJSON.toString());

		String authorization = ConfigManager.getInternalAuthToken();
		ClientResponse response = restClient.resource(registerUrl).header("Authorization", "Basic " + authorization)
				.type(MediaType.APPLICATION_JSON).post(ClientResponse.class, registerJSON.toString());

		int responseCode = response.getStatus();
		if (responseCode == 201) {
			JSONObject credentials = new JSONObject();
			credentials.put("url", serverUrl);
			credentials.put("app_id", appId);
			credentials.put("service_id", serviceId);
			credentials.put("agentUsername", registerJSON.get("agentUsername"));
			credentials.put("agentPassword", registerJSON.get("agentPassword"));
			credentials.put("api_url", getAPIServiceUrl());
			return credentials;
		} else {
			// handle failure
			String jsonResponse = response.getEntity(String.class);
			logger.error("bind failed for service  " + serviceId + " appId " + appId
					+ " with error returned from service side with code " + responseCode + ", reason " + jsonResponse);
			String errorMsg = null;
			try {
				JSONObject jsonObject = new JSONObject(jsonResponse);
				errorMsg = (String) jsonObject.get("error");
			} catch (JSONException e) {
				errorMsg = jsonResponse;
			}
			throw new ScalingServerFailureException(errorMsg);
		}
	}

	private void unregisterAppWithServer(String appId, String serviceId, String bindingId, String serverUrl)
			throws ScalingServerFailureException {

		String unregisterUrl = serverUrl + Constants.UNREGISTER_APP_URL.replace("{bindingId}", bindingId)
				.replace("{serviceId}", serviceId).replace("{appId}", appId);

		logger.info("unregisterUrl is  " + unregisterUrl);
		String authorization = ConfigManager.getInternalAuthToken();
		ClientResponse response = restClient.resource(unregisterUrl).header("Authorization", "Basic " + authorization)
				.type(MediaType.APPLICATION_JSON).delete(ClientResponse.class);

		int responseCode = response.getStatus();
		if (responseCode != 204) {
			String jsonResponse = response.getEntity(String.class);
			logger.error("unbind failed for service " + serviceId + " and bindingid " + bindingId + " with response: "
					+ responseCode + ", reason" + jsonResponse);
			String errorMsg = null;
			try {
				JSONObject jsonObject = new JSONObject(jsonResponse);
				errorMsg = (String) jsonObject.get("error");
			} catch (JSONException e) {
				errorMsg = jsonResponse;
			}
			throw new ScalingServerFailureException(errorMsg);
		}

	}

	public Map<String, Map<String, List<String>>> getWorkloadStats() {
		Map<String, Map<String, List<String>>> workloadStatsMap = new HashMap<String, Map<String, List<String>>>();

		for (String serverUrl : serverUrlList) {

			Map<String, List<String>> serviceBindingMap = new HashMap<String, List<String>>();
			List<String> serviceIds = dataService.getServiceInstanceIdByServerURL(serverUrl);
			for (String serviceId : serviceIds) {
				List<String> appIds = dataService.getBoundAppIdByServiceId(serviceId);
				serviceBindingMap.put(serviceId, appIds);
			}
			workloadStatsMap.put(serverUrl, serviceBindingMap);
		}

		return workloadStatsMap;

	}

}
