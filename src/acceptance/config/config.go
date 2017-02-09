package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const JAVA_APP = "../assets/app/HelloWorldJavaWeb.war"
const NODE_APP = "../assets/app/nodeApp"

type Config struct {
	ApiEndpoint                  string  `json:"api"`
	AppsDomain                   string  `json:"apps_domain"`
	UseHttp                      bool    `json:"use_http"`
	AdminUser                    string  `json:"admin_user"`
	AdminPassword                string  `json:"admin_password"`
	UseExistingUser              bool    `json:"use_existing_user"`
	ShouldKeepUser               bool    `json:"keep_user_at_suite_end"`
	ExistingUser                 string  `json:"existing_user"`
	ExistingUserPassword         string  `json:"existing_user_password"`
	ConfigurableTestPassword     string  `json:"test_password"`
	PersistentAppHost            string  `json:"persistent_app_host"`
	PersistentAppSpace           string  `json:"persistent_app_space"`
	PersistentAppOrg             string  `json:"persistent_app_org"`
	PersistentAppQuotaName       string  `json:"persistent_app_quota_name"`
	SkipSSLValidation            bool    `json:"skip_ssl_validation"`
	ArtifactsDirectory           string  `json:"artifacts_directory"`
	DefaultTimeout               int     `json:"default_timeout"`
	SleepTimeout                 int     `json:"sleep_timeout"`
	DetectTimeout                int     `json:"detect_timeout"`
	CfPushTimeout                int     `json:"cf_push_timeout"`
	LongCurlTimeout              int     `json:"long_curl_timeout"`
	BrokerStartTimeout           int     `json:"broker_start_timeout"`
	AsyncServiceOperationTimeout int     `json:"async_service_operation_timeout"`
	TimeoutScale                 float64 `json:"timeout_scale"`
	JavaBuildpackName            string  `json:"java_buildpack_name"`
	NodejsBuildpackName          string  `json:"nodejs_buildpack_name"`
	NamePrefix                   string  `json:"name_prefix"`

	ServiceName    string `json:"service_name"`
	ServicePlan    string `json:"service_plan"`
	APIUrl         string `json:"api_url"`
	ReportInterval int    `json:"report_interval"`

	CfJavaTimeout   int    `json:"cf_java_timeout"`
	NodeMemoryLimit string `json:"node_memory_limit"`
}

var defaults = Config{
	PersistentAppHost:            "ASATS-persistent-app",
	PersistentAppSpace:           "ASATS-persistent-space",
	PersistentAppOrg:             "ASATS-persistent-org",
	PersistentAppQuotaName:       "ASATS-persistent-quota",
	JavaBuildpackName:            "java_buildpack",
	NodejsBuildpackName:          "nodejs_buildpack",
	DefaultTimeout:               30, // seconds
	CfPushTimeout:                2,  // minutes
	LongCurlTimeout:              2,  // minutes
	BrokerStartTimeout:           5,  // minutes
	AsyncServiceOperationTimeout: 2,  // minutes
	DetectTimeout:                5,  // minutes
	SleepTimeout:                 30, // seconds
	TimeoutScale:                 1.0,
	ArtifactsDirectory:           filepath.Join("..", "results"),
	NamePrefix:                   "ASATS",

	CfJavaTimeout:   10, // minutes
	NodeMemoryLimit: "128M",
}

func LoadConfig(t *testing.T) *Config {
	path := os.Getenv("CONFIG")
	if path == "" {
		t.Fatal("Must set $CONFIG to point to a json file.")
	}

	config := defaults
	err := loadConfigFromPath(path, &config)
	if err != nil {
		t.Fatal(err.Error())
	}
	validate(t, &config)
	return &config
}

func validate(t *testing.T, c *Config) {
	if c.ApiEndpoint == "" {
		t.Fatal("missing configuration 'api'")
	}

	if c.AdminUser == "" {
		t.Fatal("missing configuration 'admin_user'")
	}

	if c.AdminPassword == "" {
		t.Fatal("missing configuration 'admin_password'")
	}

	if c.TimeoutScale <= 0 {
		c.TimeoutScale = 1.0
	}

	if c.ServiceName == "" {
		t.Fatal("missing configuration 'service_name'")
	}

	if c.ServicePlan == "" {
		t.Fatal("missing configuration 'service_plan'")
	}

	if c.APIUrl == "" {
		t.Fatal("missing configuration 'api_url'")
	}
	if c.ReportInterval == 0 {
		t.Fatal("missing configuration 'report_interval'")
	}
}

func loadConfigFromPath(path string, config *Config) error {
	configFile, err := os.Open(path)
	if err != nil {
		return err
	}
	defer configFile.Close()

	decoder := json.NewDecoder(configFile)
	return decoder.Decode(config)
}

func (c Config) Protocol() string {
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

func (c *Config) GetNodeMemoryLimit() string {
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

func (c *Config) GetPersistentAppSpace() string {
	return c.PersistentAppSpace
}
func (c *Config) GetPersistentAppOrg() string {
	return c.PersistentAppOrg
}
func (c *Config) GetPersistentAppQuotaName() string {
	return c.PersistentAppQuotaName
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

func (c *Config) GetApiEndpoint() string {
	return c.ApiEndpoint
}
