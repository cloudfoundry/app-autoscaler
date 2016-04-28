package org.cloudfoundry.autoscaler.constant;

public final class Constants {
	public static final String VCAP_APPLICATION_ENV = "VCAP_APPLICATION";
	public static final String APP_ID = "appID";
	public static final String APP_NAME = "appName";
	public static final String APPS = "apps";
	public static final String WARDEN_IP = "wardenIP";
	public static final String INSTANCE_INDEX = "instanceIndex";
	public static final String FLAG = "flag";

	public static final String DEA = "dea";
	public static final String ZONE = "zone";
	public static final String INSTANCES = "instances";

	public static final String POLICY_COOLDOWN = "policy.cooldown";
	public static final String POLICY_DURATION = "policy.duration";
	public static final String MIN_INSTANCES = "policy.min_instances";
	public static final String MAX_INSTANCES = "policy.max_instances";
	public static final String THRESHOLD_LOW = "policy.threshold.low.";
	public static final String THRESHOLD_HIGH = "policy.threshold.high.";

	public static final String ZONES = "zones";
	public static final String TYPE = "type";
	public static final String MAX = "max";
	public static final String MIN = "min";
	public static final String POLICIES = "policies";
	public static final String COOL_DOWN = "cool_down";
	public static final String DURATION = "duration";

	public static final int DASHBORAD_TIME_RANGE = 30;

	// metric db rollout frequency
	public static final String HOURLY = "hourly";
	public static final String DAILY = "daily";
	public static final String WEEKLY = "weekly";
	public static final String MONTHLY = "monthly";
	public static final String CUSTOM = "custom";
	public static final String CONTINUOUS = "continuously";

	public static final String USERNAME = "cfClientId";
	public static final String PASSWORD = "cfClientSecret";
	public static final String CFURL = "cfUrl";

	public static final String GUID = "guid";
	public static final String RESOURCES = "resources";
	public static final String ENTITY = "entity";
	public static final String METADATA = "metadata";
	public static final String NAME = "name";
	public static final String METRICS_PROCESSORS = "metrics.processors";
	public static final String METRICS_ACTIONS = "metrics.actions";

	public static final String POLLING_INTERVAL = "reportInterval";
	public static final String POLLING_WAIT = "pollingWaitbeforestart";
	public static final String POLLER_THREAD_COUNT = "pollerThreadCount";

	public static final String APP_TYPE_JAVA = "java";
	public static final String APP_TYPE_RUBY = "ruby";
	public static final String APP_TYPE_RUBY_SINATRA = "ruby_sinatra";
	public static final String APP_TYPE_RUBY_ON_RAILS = "ruby_on_rails";
	public static final String APP_TYPE_NODEJS = "nodejs";
	public static final String APP_TYPE_GO = "go";
	public static final String APP_TYPE_PHP = "php";
	public static final String APP_TYPE_PYTHON = "python";
	public static final String APP_TYPE_DOTNET = "dotnet";
	public static final String APP_TYPE_UNKNOWN = "unknown";

	public static final String APPLICATION_STATE_ENABLED = "enabled";// application state - enable
	public static final String APPLICATION_STATE_DISABLED = "disabled";// application state - disable
	public static final String APPLICATION_STATE_UNBOUND = "unbound";// application state - unbound

	public static final String METRIC_POLLER_MEMORY = "cf-stats#Memory";
	public static final String METRIC_POLLER_MEMORY_QUOATA = "cf-stats#MemoryQuota";
	public static final String METRIC_POLLER_CPU = "cf-stats#CPU";

	// metric.* related entries defined in config.properties
	public static final String METRIC_WILDCARD = "*";
	public static final String METRIC_SEPERATOR = "_";
	public static final String METRIC_PREFIX = "metric";
	public static final String METRIC_SOURCE = "source";
	public static final String METRIC_SOURCE_POLLER = "poller";
	public static final String METRIC_SOURCE_STATUS = "enabled";
	public static final String METRIC_SOURCE_TYPE = "type";
	public static final String REPORT_INTERVAL = "reportInterval";
	public static final String PERSIST_TIME = "persistTimeInDb";
	public static final String METRICS_CONFIG = "metricsConfig";
	public static final String METRICS_TIME_UNIT_MS = "ms";
	public static final String METRICS_TIME_UNIT_NS = "ns";

	// cf state of an app, the valid value is "started" and "stopped"
	public static final String CF_APPLICATION_STATE_STARTED = "STARTED";
	public static final String CF_APPLICATION_STATE_STOPPED = "STOPPED";
	public final static String MemoryQuotaExceeded = "CF-AppMemoryQuotaExceeded";
	public final static String CloudFoundryInternalError = "CloudFoundryInternalError"; // CloudFoundry error

	public enum MESSAGE_KEY {
		RestResponseErrorMsg_build_JSON_error,
		RestResponseErrorMsg_parse_JSON_error,
		RestResponseErrorMsg_config_exist_error,
		RestResponseErrorMsg_policy_not_found_error(404),
		RestResponseErrorMsg_applist_not_empty_error,
		RestResponseErrorMsg_app_not_found_error,
		RestResponseErrorMsg_cloud_error(500),
		RestResponseErrorMsg_metric_not_supported_error,
		RestResponseErrorMsg_database_error(500),
		RestResponseErrorMsg_no_attached_policy_error;

		private int errorCode;

		private MESSAGE_KEY() {
			this.errorCode = 400;
		}

		private MESSAGE_KEY(int errorCode) {
			this.errorCode = errorCode;
		}

		public int getErrorCode() {
			return errorCode;
		}
	}
}
