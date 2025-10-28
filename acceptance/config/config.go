package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/onsi/ginkgo/v2"
)

const NODE_APP = "../assets/app/nodeApp"
const GO_APP = "../assets/app/go_app/build"

type PerformanceConfig struct {
	AppCount                      int  `json:"app_count"`
	PercentageToScale             int  `json:"app_percentage_to_scale"`
	SetupWorkers                  int  `json:"setup_workers"`
	ExpectedMedianScaleOutSeconds int  `json:"expected_median_scale_out_seconds"`
	ExpectedMedianScaleInSeconds  int  `json:"expected_median_scale_in_seconds"`
	UpdateExistingOrgQuota        bool `json:"update_existing_org_quota"`
}

type Config struct {
	ApiEndpoint                    string  `json:"api"`
	AppsDomain                     string  `json:"apps_domain"`
	UseHttp                        bool    `json:"use_http"`
	AdminUser                      string  `json:"admin_user"`
	AdminPassword                  string  `json:"admin_password"`
	AdminOrigin                    string  `json:"admin_origin"`
	UseExistingUser                bool    `json:"use_existing_user"`
	ShouldKeepUser                 bool    `json:"keep_user_at_suite_end"`
	ExistingUser                   string  `json:"existing_user"`
	ExistingUserPassword           string  `json:"existing_user_password"`
	UserOrigin                     string  `json:"user_origin"`
	ConfigurableTestPassword       string  `json:"test_password"`
	UseExistingOrganization        bool    `json:"use_existing_organization"`
	ExistingOrganization           string  `json:"existing_organization"`
	AddExistingUserToExistingSpace bool    `json:"add_existing_user_to_existing_space"`
	UseExistingSpace               bool    `json:"use_existing_space"`
	ExistingSpace                  string  `json:"existing_space"`
	SkipSSLValidation              bool    `json:"skip_ssl_validation"`
	ArtifactsDirectory             string  `json:"artifacts_directory"`
	DefaultTimeout                 int     `json:"default_timeout"`
	SleepTimeout                   int     `json:"sleep_timeout"`
	DetectTimeout                  int     `json:"detect_timeout"`
	CfPushTimeout                  int     `json:"cf_push_timeout"`
	LongCurlTimeout                int     `json:"long_curl_timeout"`
	BrokerStartTimeout             int     `json:"broker_start_timeout"`
	AsyncServiceOperationTimeout   int     `json:"async_service_operation_timeout"`
	TimeoutScale                   float64 `json:"timeout_scale"`
	JavaBuildpackName              string  `json:"java_buildpack_name"`
	NodejsBuildpackName            string  `json:"nodejs_buildpack_name"`
	BinaryBuildpackName            string  `json:"binary_buildpack_name"`
	NamePrefix                     string  `json:"name_prefix"`
	InstancePrefix                 string  `json:"instance_prefix"`
	AppPrefix                      string  `json:"app_prefix"`
	Prefix                         string  `json:"prefix"`

	AdminClient          string `json:"admin_client"`
	AdminClientSecret    string `json:"admin_client_secret"`
	ExistingClient       string `json:"existing_client"`
	ExistingClientSecret string `json:"existing_client_secret"`

	ServiceBroker     string `json:"service_broker"`
	ServiceName       string `json:"service_name"`
	ServicePlan       string `json:"service_plan"`
	AggregateInterval int    `json:"aggregate_interval"`

	CfJavaTimeout   int `json:"cf_java_timeout"`
	NodeMemoryLimit int `json:"node_memory_limit"`

	ASApiEndpoint       string `json:"autoscaler_api"`
	EnableServiceAccess bool   `json:"enable_service_access"`

	EventgeneratorHealthEndpoint   string `json:"eventgenerator_health_endpoint"`
	ScalingengineHealthEndpoint    string `json:"scalingengine_health_endpoint"`
	OperatorHealthEndpoint         string `json:"operator_health_endpoint"`
	MetricsforwarderHealthEndpoint string `json:"metricsforwarder_health_endpoint"`
	SchedulerHealthEndpoint        string `json:"scheduler_health_endpoint"`

	HealthEndpointsBasicAuthEnabled bool `json:"health_endpoints_basic_auth_enabled"`

	CPUUpperThreshold int64 `json:"cpu_upper_threshold"`

	CPUUtilScalingPolicyTest CPUUtilScalingPolicyTest `json:"cpuutil_scaling_policy_test"`

	Performance PerformanceConfig `json:"performance"`
}

type CPUUtilScalingPolicyTest struct {
	AppMemory         string `json:"app_memory"`
	AppCPUEntitlement int    `json:"app_cpu_entitlement"`
}

var defaults = Config{
	AddExistingUserToExistingSpace: true,

	BinaryBuildpackName:          "binary_buildpack",
	JavaBuildpackName:            "java_buildpack",
	NodejsBuildpackName:          "nodejs_buildpack",
	DefaultTimeout:               30, // seconds
	CfPushTimeout:                3,  // minutes
	LongCurlTimeout:              2,  // minutes
	BrokerStartTimeout:           5,  // minutes
	AsyncServiceOperationTimeout: 2,  // minutes
	DetectTimeout:                5,  // minutes
	SleepTimeout:                 30, // seconds
	TimeoutScale:                 1.0,
	ArtifactsDirectory:           filepath.Join("..", "results"),
	NamePrefix:                   "ASATS",
	InstancePrefix:               "service",
	AppPrefix:                    "nodeapp",
	Prefix:                       "autoscaler",
	ServiceBroker:                "autoscaler",

	CfJavaTimeout:                   10,  // minutes
	NodeMemoryLimit:                 128, // MB
	EnableServiceAccess:             true,
	HealthEndpointsBasicAuthEnabled: true,
	CPUUpperThreshold:               100,

	UseExistingOrganization: false,
	ExistingOrganization:    "",

	EventgeneratorHealthEndpoint:   "",
	ScalingengineHealthEndpoint:    "",
	OperatorHealthEndpoint:         "",
	MetricsforwarderHealthEndpoint: "",
	SchedulerHealthEndpoint:        "",

	CPUUtilScalingPolicyTest: CPUUtilScalingPolicyTest{
		AppMemory:         "1GB",
		AppCPUEntitlement: 25,
	},

	Performance: PerformanceConfig{
		AppCount:                      100,
		PercentageToScale:             30,
		SetupWorkers:                  50,  // number of thread/process
		ExpectedMedianScaleInSeconds:  100, // Replace with educated guesses base on test runs
		ExpectedMedianScaleOutSeconds: 100, // Replace with educated guesses base on test run
		UpdateExistingOrgQuota:        true,
	},
}

// TerminateSuite allows to pass different suite termination behaviours,
// matches the function signature of ginkgo.AbortSuite.
type TerminateSuite func(message string, callerSkip ...int)

var DefaultTerminateSuite TerminateSuite = ginkgo.AbortSuite

func LoadConfig(terminateSuite TerminateSuite) *Config {
	path := os.Getenv("CONFIG")
	if path == "" {
		terminateSuite("Must set $CONFIG to point to a json file")
	}

	config := defaults
	err := loadConfigFromPath(path, &config)
	if err != nil {
		terminateSuite(err.Error())
	}
	validate(&config, terminateSuite)
	return &config
}

func validate(c *Config, terminateSuite TerminateSuite) {
	if c.ApiEndpoint == "" {
		terminateSuite("missing configuration 'api'")
	}

	if c.AdminUser == "" {
		terminateSuite("missing configuration 'admin_user'")
	}

	if c.AdminPassword == "" {
		terminateSuite("missing configuration 'admin_password'")
	}

	if c.TimeoutScale <= 0 {
		c.TimeoutScale = 1.0
	}

	if c.ServiceName == "" {
		terminateSuite("missing configuration 'service_name'")
	}

	if c.ServicePlan == "" {
		terminateSuite("missing configuration 'service_plan'")
	}

	if c.AggregateInterval == 0 {
		terminateSuite("missing configuration 'aggregate_interval'")
	} else {
		if c.AggregateInterval < 60 {
			c.AggregateInterval = 60
		}
	}

	if c.ASApiEndpoint == "" {
		terminateSuite("missing configuration 'autoscaler_api'")
	} else {
		c.ASApiEndpoint = normalizeURL(c.ASApiEndpoint, c.UseHttp)
	}

	if c.EventgeneratorHealthEndpoint != "" {
		c.EventgeneratorHealthEndpoint = normalizeURL(
			c.EventgeneratorHealthEndpoint,
			c.UseHttp,
		)
	}

	if c.ScalingengineHealthEndpoint != "" {
		c.ScalingengineHealthEndpoint = normalizeURL(
			c.ScalingengineHealthEndpoint,
			c.UseHttp,
		)
	}

	if c.OperatorHealthEndpoint != "" {
		c.OperatorHealthEndpoint = normalizeURL(
			c.OperatorHealthEndpoint,
			c.UseHttp,
		)
	}

	if c.MetricsforwarderHealthEndpoint != "" {
		c.MetricsforwarderHealthEndpoint = normalizeURL(
			c.MetricsforwarderHealthEndpoint,
			c.UseHttp,
		)
	}

	if c.SchedulerHealthEndpoint != "" {
		c.SchedulerHealthEndpoint = normalizeURL(
			c.SchedulerHealthEndpoint,
			c.UseHttp,
		)
	}
}

func normalizeURL(url string, useHttp bool) string {
	url = strings.TrimSuffix(url, "/")
	if !strings.HasPrefix(url, "http") {
		if useHttp {
			url = "http://" + url
		} else {
			url = "https://" + url
		}
	}
	return url
}

func loadConfigFromPath(path string, config *Config) error {
	configFile, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = configFile.Close() }()

	decoder := json.NewDecoder(configFile)
	return decoder.Decode(config)
}

func (c *Config) Protocol() string {
	if c.UseHttp {
		return "http://"
	} else {
		return "https://"
	}
}

func (c *Config) DefaultTimeoutDuration() time.Duration {
	return time.Duration(c.DefaultTimeout) * time.Second
}
func (c *Config) SleepTimeoutDuration() time.Duration {
	return time.Duration(c.SleepTimeout) * time.Second
}

func (c *Config) DetectTimeoutDuration() time.Duration {
	return time.Duration(c.DetectTimeout) * time.Minute
}

func (c *Config) CfPushTimeoutDuration() time.Duration {
	return time.Duration(c.CfPushTimeout) * time.Minute
}

func (c *Config) LongCurlTimeoutDuration() time.Duration {
	return time.Duration(c.LongCurlTimeout) * time.Minute
}

func (c *Config) BrokerStartTimeoutDuration() time.Duration {
	return time.Duration(c.BrokerStartTimeout) * time.Minute
}

func (c *Config) AsyncServiceOperationTimeoutDuration() time.Duration {
	return time.Duration(c.AsyncServiceOperationTimeout) * time.Minute
}

func (c *Config) CFJavaTimeoutDuration() time.Duration {
	return time.Duration(c.CfJavaTimeout) * time.Minute
}

func (c Config) GetScaledTimeout(timeout time.Duration) time.Duration {
	return time.Duration(float64(timeout) * c.TimeoutScale)
}

func (c *Config) GetNodeMemoryLimit() int {
	return c.NodeMemoryLimit
}

func (c *Config) GetAppsDomain() string {
	return c.AppsDomain
}

func (c *Config) GetSkipSSLValidation() bool {
	return c.SkipSSLValidation
}

func (c *Config) GetArtifactsDirectory() string {
	return c.ArtifactsDirectory
}

func (c *Config) GetUseExistingOrganization() bool {
	return c.UseExistingOrganization
}

func (c *Config) GetExistingOrganization() string {
	return c.ExistingOrganization
}

func (c *Config) GetAddExistingUserToExistingSpace() bool {
	return c.AddExistingUserToExistingSpace
}

func (c *Config) GetUseExistingSpace() bool {
	return c.UseExistingSpace
}

func (c *Config) GetExistingSpace() string {
	return c.ExistingSpace
}

func (c *Config) GetNamePrefix() string {
	return c.NamePrefix
}

func (c *Config) GetUseExistingUser() bool {
	return c.UseExistingUser
}

func (c *Config) GetExistingUser() string {
	return c.ExistingUser
}

func (c *Config) GetExistingUserPassword() string {
	return c.ExistingUserPassword
}

func (c *Config) GetUserOrigin() string {
	return c.UserOrigin
}

func (c *Config) GetConfigurableTestPassword() string {
	return c.ConfigurableTestPassword
}

func (c *Config) GetShouldKeepUser() bool {
	return c.ShouldKeepUser
}

func (c *Config) GetAdminUser() string {
	return c.AdminUser
}

func (c *Config) GetAdminPassword() string {
	return c.AdminPassword
}

func (c *Config) GetAdminOrigin() string {
	return c.AdminOrigin
}

func (c *Config) GetApiEndpoint() string {
	return c.ApiEndpoint
}

func (c *Config) ShouldEnableServiceAccess() bool {
	return c.EnableServiceAccess
}

func (c *Config) GetAdminClient() string {
	return c.AdminClient
}

func (c *Config) GetAdminClientSecret() string {
	return c.AdminClientSecret
}

func (c *Config) GetExistingClient() string {
	return c.ExistingClient
}

func (c *Config) GetExistingClientSecret() string {
	return c.ExistingClientSecret
}
