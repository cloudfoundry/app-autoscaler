package org.cloudfoundry.autoscaler.servicebroker;

import com.fasterxml.jackson.databind.ObjectMapper;

public final class Constants {

	public static final ObjectMapper MAPPER = new ObjectMapper();

	public static final String MSG_ENTRY = "Entry: ";
	public static final String MSG_EXIT = "Exit: ";
	public static final String MSG_SUCCESS = "Success: ";
	public static final String MSG_FAIL = "Failed: ";

	public static final String REGISTER_APP_URL = "/resources/brokercallback";
	public static final String UNREGISTER_APP_URL = "/resources/brokercallback?bindingId={bindingId}&serviceId={serviceId}&appId={appId}";
	public static final String TUNE_SEVICE_ENABLEMENT_URL = "/resources/brokercallback/enablement/{serviceId}";

	public static final String HEADER_AUTHORIZATION = "Authorization";

	public static final String DASHBOARD_URL = "";

	public static final String CONFIG_ENTRY_SERVER_URI_LIST = "serverURIList";
	public static final String CONFIG_ENTRY_API_SERVER_URI = "apiServerURI";
	public static final String CONFIG_ENTRY_HTTP_PROTOCOL = "httpProtocol";
	public static final String CONFIG_ENTRY_DATASTORE_PROVIDER = "storeProvider";
	public static final String CONFIG_ENTRY_DATASTORE_PROVIDER_COUCHDB = "couchdb";

	public static final String COUCHDOCUMENT_TYPE_APPLICATIONINSTANCE = "ApplicationInstance_inBroker";
	public static final String COUCHDOCUMENT_TYPE_SERVICEINSTANCE = "ServiceInstance_inBroker";

	public static final String PROXY_USERNAME = "userName";
	public static final String PROXY_PASSWORD = "password";
	public static final String PROXY_HOST = "host";
	public static final String PROXY_PORT = "port";

	public enum MESSAGE_KEY {
		ParseJSONError(400),
		ServerUrlMappingNotFound(499),
		AlreadyBindedAnotherService(499),
		BindServiceFail(499),
		UnbindServiceFail(499),
		DeleteServiceFail,
		EnableServiceFail,
		QueryServiceEnabledInfoFail;

		private int errorCode;

		private MESSAGE_KEY() {
			this.errorCode = 500;
		}

		private MESSAGE_KEY(int errorCode) {
			this.errorCode = errorCode;
		}

		public int getErrorCode() {
			return errorCode;
		}

	}

}
