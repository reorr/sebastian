package qiscus

const (
	// API Endpoints
	AUTH_PATH                    = "/api/v1/auth"
	ALLOCATE_AGENT_PATH          = "/api/v1/admin/service/allocate_agent"
	ALLOCATE_ASSIGN_AGENT_PATH   = "/api/v1/admin/service/allocate_assign_agent"
	ASSIGN_AGENT_PATH            = "/api/v1/admin/service/assign_agent"
	GET_AVAILABLE_AGENT_PATH     = "/api/v2/admin/service/available_agents"
	MARK_AS_RESOLVED_PATH        = "/api/v1/admin/service/mark_as_resolved"
	GET_ALL_AGENT_PATH           = "/api/v2/admin/agents?limit=1000"
	GET_WEBHOOK_CONFIG_PATH      = "/api/v2/admin/webhook_config"
	SET_WEBHOOK_MARK_AS_RESOLVED = "/api/v1/app/webhook/mark_as_resolved"
	SET_WEBHOOK_INCOMING_MESSAGE = "/api/v1/app/webhook/agent_allocation"

	// Cache Keys
	CACHE_TOKEN_KEY = "token"
)