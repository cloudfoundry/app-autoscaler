package config_test

import (
	. "autoscaler/api/config"
	"autoscaler/db"
	"bytes"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	yaml "gopkg.in/yaml.v2"
)

var _ = Describe("Config", func() {

	var (
		conf        *Config
		err         error
		configBytes []byte
	)

	Describe("Load Config", func() {
		JustBeforeEach(func() {
			conf, err = LoadConfig(bytes.NewReader(configBytes))
		})

		Context("with invalid yaml", func() {
			BeforeEach(func() {
				configBytes = []byte(`
server:
	port: 8080,
logging:
  level: debug
broker_username: brokeruser
broker_password: supersecretpassword
db:
  binding_db:
    url: postgres://postgres:postgres@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
catalog: |
  {
    "services": [{
        "id": "autoscaler-guid",
        "name": "autoscaler",
        "description": "Automatically increase or decrease the number of application instances based on a policy you define.",
        "bindable": true,
        "plans": [{
            "id": "autoscaler-free-plan-id",
            "name": "autoscaler-free-plan",
            "description": "This is the free service plan for the Auto-Scaling service."
        }]
    }]
  }`)
			})

			It("returns an error", func() {
				Expect(err).To(MatchError(MatchRegexp("yaml: .*")))
			})
		})
		Context("with valid yaml", func() {
			BeforeEach(func() {
				configBytes = []byte(`
server:
  port: 8080
logging:
  level: debug
broker_username: brokeruser
broker_password: supersecretpassword
db:
  binding_db:
    url: postgres://postgres:postgres@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
catalog: |
  {
    "services": [{
        "id": "autoscaler-guid",
        "name": "autoscaler",
        "description": "Automatically increase or decrease the number of application instances based on a policy you define.",
        "bindable": true,
        "plans": [{
            "id": "autoscaler-free-plan-id",
            "name": "autoscaler-free-plan",
            "description": "This is the free service plan for the Auto-Scaling service."
        }]
    }]
  }`)
			})

			It("It returns the config", func() {
				Expect(err).NotTo(HaveOccurred())

				Expect(conf.Logging.Level).To(Equal("debug"))
				Expect(conf.Server.Port).To(Equal(8080))
				Expect(conf.DB.BindingDB).To(Equal(
					db.DatabaseConfig{
						URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
						MaxOpenConnections:    10,
						MaxIdleConnections:    5,
						ConnectionMaxLifetime: 60 * time.Second,
					}))
				Expect(conf.BrokerUsername).To(Equal("brokeruser"))
				Expect(conf.BrokerPassword).To(Equal("supersecretpassword"))
				Expect(conf.Catalog).To(Equal(`{
  "services": [{
      "id": "autoscaler-guid",
      "name": "autoscaler",
      "description": "Automatically increase or decrease the number of application instances based on a policy you define.",
      "bindable": true,
      "plans": [{
          "id": "autoscaler-free-plan-id",
          "name": "autoscaler-free-plan",
          "description": "This is the free service plan for the Auto-Scaling service."
      }]
  }]
}`))
			})
		})
		Context("with partial config", func() {
			BeforeEach(func() {
				configBytes = []byte(`
broker_username: brokeruser
broker_password: supersecretpassword
db:
  binding_db:
    url: postgres://postgres:postgres@localhost/autoscaler?sslmode=disable
catalog: |
  {
    "services": [{
        "id": "autoscaler-guid",
        "name": "autoscaler",
        "description": "Automatically increase or decrease the number of application instances based on a policy you define.",
        "bindable": true,
        "plans": [{
            "id": "autoscaler-free-plan-id",
            "name": "autoscaler-free-plan",
            "description": "This is the free service plan for the Auto-Scaling service."
        }]
    }]
  }`)
			})
			It("It returns the default values", func() {
				Expect(err).NotTo(HaveOccurred())

				Expect(conf.Logging.Level).To(Equal("info"))
				Expect(conf.Server.Port).To(Equal(8080))
				Expect(conf.DB.BindingDB).To(Equal(
					db.DatabaseConfig{
						URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
						MaxOpenConnections:    0,
						MaxIdleConnections:    0,
						ConnectionMaxLifetime: 0 * time.Second,
					}))
			})
		})

		Context("when it gives a non integer port", func() {
			BeforeEach(func() {
				configBytes = []byte(`
server:
  port: port
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
			})
		})

		Context("when it gives a non integer max_open_connections of policydb", func() {
			BeforeEach(func() {
				configBytes = []byte(`
server:
  port: 8080
logging:
  level: debug
broker_username: brokeruser
broker_password: supersecretpassword
db:
  binding_db:
    url: postgres://postgres:postgres@localhost/autoscaler?sslmode=disable
    max_open_connections: NOT-INTEGER-VALUE
    max_idle_connections: 5
    connection_max_lifetime: 60s
catalog: |
  {
    "services": [{
        "id": "autoscaler-guid",
        "name": "autoscaler",
        "description": "Automatically increase or decrease the number of application instances based on a policy you define.",
        "bindable": true,
        "plans": [{
            "id": "autoscaler-free-plan-id",
            "name": "autoscaler-free-plan",
            "description": "This is the free service plan for the Auto-Scaling service."
        }]
    }]
  }`)
			})
			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
			})
		})
		Context("when it gives a non integer max_idle_connections of policydb", func() {
			BeforeEach(func() {
				configBytes = []byte(`
server:
  port: 8080
logging:
  level: debug
broker_username: brokeruser
broker_password: supersecretpassword
db:
  binding_db:
    url: postgres://postgres:postgres@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: NOT-INTEGER-VALUE
    connection_max_lifetime: 60s
catalog: |
  {
    "services": [{
        "id": "autoscaler-guid",
        "name": "autoscaler",
        "description": "Automatically increase or decrease the number of application instances based on a policy you define.",
        "bindable": true,
        "plans": [{
            "id": "autoscaler-free-plan-id",
            "name": "autoscaler-free-plan",
            "description": "This is the free service plan for the Auto-Scaling service."
        }]
    }]
  }`)
			})
			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
			})
		})
		Context("when it gives a non integer connection_max_lifetime of policydb", func() {
			BeforeEach(func() {
				configBytes = []byte(`
server:
  port: 8080
logging:
  level: debug
broker_username: brokeruser
broker_password: supersecretpassword
db:
  binding_db:
    url: postgres://postgres:postgres@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: NOT-TIME-DURATION
catalog: |
  {
    "services": [{
        "id": "autoscaler-guid",
        "name": "autoscaler",
        "description": "Automatically increase or decrease the number of application instances based on a policy you define.",
        "bindable": true,
        "plans": [{
            "id": "autoscaler-free-plan-id",
            "name": "autoscaler-free-plan",
            "description": "This is the free service plan for the Auto-Scaling service."
        }]
    }]
  }`)
			})
			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into time.Duration")))
			})
		})

	})

	Describe("Validate", func() {
		BeforeEach(func() {
			conf = &Config{}
			conf.DB.BindingDB = db.DatabaseConfig{
				URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
				MaxOpenConnections:    10,
				MaxIdleConnections:    5,
				ConnectionMaxLifetime: 60 * time.Second,
			}
			conf.BrokerUsername = "brokeruser"
			conf.BrokerPassword = "supersecretpassword"
			conf.Catalog = `{
"services": [{
  "id": "autoscaler-guid",
  "name": "autoscaler",
  "description": "Automatically increase or decrease the number of application instances based on a policy you define.",
  "bindable": true,
  "plans": [{
	  "id": "autoscaler-free-plan-id",
	  "name": "autoscaler-free-plan",
	  "description": "This is the free service plan for the Auto-Scaling service."
  }]
}]
}`
		})
		JustBeforeEach(func() {
			err = conf.Validate()
		})

		Context("When all the configs are valid", func() {
			It("should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when bindingdb url is not set", func() {
			BeforeEach(func() {
				conf.DB.BindingDB.URL = ""
			})
			It("should err", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: BindingDB URL is empty")))
			})
		})

		Context("when broker username is not set", func() {
			BeforeEach(func() {
				conf.BrokerUsername = ""
			})
			It("should err", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: BrokerUsername is empty")))
			})
		})

		Context("when broker password is not set", func() {
			BeforeEach(func() {
				conf.BrokerPassword = ""
			})
			It("should err", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: BrokerPassword is empty")))
			})
		})

		Context("when catalog is not set", func() {
			BeforeEach(func() {
				conf.Catalog = ""
			})
			It("should err", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: Catalog is empty")))
			})
		})
	})
})
