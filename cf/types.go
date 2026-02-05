package cf

import (
	"errors"
	"time"
)

const (
	TokenTypeBearer   = "Bearer"
	ProcessTypeWeb    = "web"
	ProcessTypeWorker = "worker"
	CCAdminScope      = "cloud_controller.admin"
)

const (
	RoleOrganisationUser           RoleType = "organization_user"
	RoleOrganizationAuditor        RoleType = "organization_auditor"
	RoleOrganizationManager        RoleType = "organization_manager"
	RoleOrganizationBillingManager RoleType = "organization_billing_manager"
	RoleSpaceAuditor               RoleType = "space_auditor"
	RoleSpaceDeveloper             RoleType = "space_developer"
	RoleSpaceManager               RoleType = "space_manager"
	RoleSpaceSupporter             RoleType = "space_supporter"
)

type (
	// App the app information from cf for full version look at https://v3-apidocs.cloudfoundry.org/version/3.122.0/index.html#apps
	App struct {
		Guid          string        `json:"guid"`
		Name          string        `json:"name"`
		State         string        `json:"state"`
		CreatedAt     time.Time     `json:"created_at"`
		UpdatedAt     time.Time     `json:"updated_at"`
		Relationships Relationships `json:"relationships"`
		Metadata      `json:"metadata"`
	}

	AppAndProcesses struct {
		App       *App
		Processes Processes
	}

	Relationships struct {
		Space *Space `json:"space"`
	}
	SpaceData struct {
		Guid SpaceId `json:"guid"`
	}
	Space struct {
		Data SpaceData `json:"data"`
	}

	// Process information for an App from cf for full version look at https://v3-apidocs.cloudfoundry.org/version/3.122.0/index.html#processes
	Process struct {
		Guid       string    `json:"guid"`
		Type       string    `json:"type"`
		Instances  int       `json:"instances"`
		MemoryInMb int       `json:"memory_in_mb"`
		DiskInMb   int       `json:"disk_in_mb"`
		CreatedAt  time.Time `json:"created_at"`
		UpdatedAt  time.Time `json:"updated_at"`
	}
	Processes []Process
)

func (p Processes) GetInstances() int {
	instances := 0
	for _, process := range p {
		instances += process.Instances
	}
	return instances
}

type (
	ServicePlanData struct {
		Guid string `json:"guid"`
	}
	ServicePlanRelation struct {
		Data ServicePlanData `json:"data"`
	}
	ServiceInstanceRelationships struct {
		ServicePlan ServicePlanRelation `json:"service_plan"`
	}
	ServiceInstance struct {
		Guid          string                       `json:"guid"`
		Type          string                       `json:"type"`
		Relationships ServiceInstanceRelationships `json:"relationships"`
	}
)

type ServicePlanEntity struct {
	UniqueId string `json:"unique_id"`
}

type ServicePlanResource struct {
	Entity ServicePlanEntity `json:"entity"`
}

type (
	ServicePlan struct {
		Guid          string        `json:"guid"`
		BrokerCatalog BrokerCatalog `json:"broker_catalog"`
	}
	BrokerCatalog struct {
		Id string `json:"id"`
	}
)

type (
	Endpoints struct {
		CloudControllerV3 Href `json:"cloud_controller_v3"`
		NetworkPolicyV0   Href `json:"network_policy_v0"`
		NetworkPolicyV1   Href `json:"network_policy_v1"`
		Login             Href `json:"login"`
		Uaa               Href `json:"uaa"`
		Routing           Href `json:"routing"`
		Logging           Href `json:"logging"`
		LogCache          Href `json:"log_cache"`
		LogStream         Href `json:"log_stream"`
		AppSsh            Href `json:"app_ssh"`
	}
	EndpointsResponse struct {
		Links Endpoints `json:"links"`
	}
)

type (
	Pagination struct {
		TotalResults int  `json:"total_results"`
		TotalPages   int  `json:"total_pages"`
		First        Href `json:"first"`
		Last         Href `json:"last"`
		Next         Href `json:"next"`
		Previous     Href `json:"previous"`
	}
	Href struct {
		Url string `json:"href"`
	}

	// Response for example https://v3-apidocs.cloudfoundry.org/version/3.122.0/index.html#list-processes
	Response[T any] struct {
		Pagination Pagination `json:"pagination"`
		Resources  []T        `json:"resources"`
	}
)

type (
	Role struct {
		Guid string   `json:"guid"`
		Type RoleType `json:"type"`
	}
	RoleType string
)

type Roles []Role

type SpaceId string
type UserId string

func (r Roles) HasRole(roleType RoleType) bool {
	for _, role := range r {
		if role.Type == roleType {
			return true
		}
	}
	return false
}

// Error types
var (
	ErrUnauthorized       = errors.New("Unauthorized")
	ErrInvalidTokenFormat = errors.New("invalid token format")
)
