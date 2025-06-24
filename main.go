package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const (
	AUTH_PATH                    = "/api/v1/auth"
	MARK_AS_RESOLVED_PATH        = "/api/v1/admin/service/mark_as_resolved"
	ALLOCATE_AGENT_PATH          = "/api/v1/admin/service/allocate_agent"
	ALLOCATE_ASSIGN_AGENT_PATH   = "/api/v1/admin/service/allocate_assign_agent"
	ASSIGN_AGENT_PATH            = "/api/v1/admin/service/assign_agent"
	GET_ALL_AGENT_PATH           = "/api/v2/admin/agents?limit=1000"
	GET_AVAILABLE_AGENT_PATH     = "/api/v2/admin/service/available_agents"
	GET_WEBHOOK_CONFIG_PATH      = "/api/v2/admin/webhook_config"
	SET_WEBHOOK_MARK_AS_RESOLVED = "/api/v1/app/webhook/mark_as_resolved"
	SET_WEBHOOK_INCOMING_MESSAGE = "/api/v1/app/webhook/agent_allocation"
	CACHE_TOKEN_KEY              = "token"

	WEBHOOK_MARK_AS_RESOLVED_PATH = "/webhook-mark-as-resolved"
	WEBHOOK_INCOMING_MESSAGE_PATH = "/webhook-incoming-message"
)

var (
	cfg         = defaultConfig()
	rdb         *redis.Client
	queueClient *asynq.Client
	pool        *pgxpool.Pool
	err         error
)

func main() {
	// runArg := os.Args[1]
	// if runArg != "webhook" && runArg != "worker" {
	// 	fmt.Println("Invalid argument. Use 'webhook' or 'worker'.")
	// 	os.Exit(1)
	// }

	var configFileName, exec string
	flag.StringVar(&exec, "e", "webhook", "Service to run. Use 'webhook' or 'worker'.")
	flag.StringVar(&configFileName, "c", "config.yml", "Config file name")

	flag.Parse()

	cfg.loadFromEnv()

	if len(configFileName) > 0 {
		err := loadConfigFromFile(configFileName, &cfg)
		if err != nil {
			// log.Warn().Str("file", configFileName).Err(err).Msg("cannot load config file, use defaults")
		}
	}

	// log.Debug().Any("config", cfg).Msg("config loaded")

	ctx := context.Background()

	fmt.Println("Webhook base url: ", cfg.WebhookConfig.BaseUrl)

	rdb = redis.NewClient(&redis.Options{
		Addr: cfg.RedisConfig.Url,
	})

	queueClient = asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.RedisConfig.Url})
	if queueClient == nil {
		fmt.Println("Error creating Asynq client")
		panic("Failed to create Asynq client")
	}

	pool, err = pgxpool.New(ctx, cfg.DBConfig.ConnectionString)
	if err != nil {
		fmt.Printf("Error connecting to database: %v\n", err.Error())
		panic(err)
	}

	switch exec {
	case "webhook":
		runServer(int(cfg.Listen.Port))
	case "worker":
		runWorker()
	default:
		fmt.Println("Invalid argument. Use 'webhook' or 'worker'.")
	}
}

func runServer(port int) {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Post(WEBHOOK_INCOMING_MESSAGE_PATH, HandleIncomingMessage)
	r.Post(WEBHOOK_MARK_AS_RESOLVED_PATH, HandleMarkAsResolved)
	// r.Get("/agents", HandleGetAllAgent)
	// r.Get("/webhook-config", HandlerGetWebhookConfig)
	r.Post("/set-webhook", HandlerSetWebhook)
	// r.Get("/get-available-agent", HandlerGetAvailableAgent)

	listenPort := fmt.Sprintf(":%d", port)
	fmt.Printf("Listening on port: %s\n", listenPort)

	http.ListenAndServe(listenPort, r)
}

func runWorker() {
	fmt.Println("Starting worker...")
	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: cfg.RedisConfig.Url},
		asynq.Config{
			Concurrency: 1,
		},
	)

	if err := CacheAgentStatus(); err != nil {
		panic(fmt.Errorf("Initial agent cache update failed: %w", err))
	}
	go InitAgents()

	mux := asynq.NewServeMux()
	mux.HandleFunc(TypeChatAssignAgent, HandleChatAssignAgentTask)

	if err := srv.Run(mux); err != nil {
		panic(fmt.Sprintf("could not run server: %v", err))
	}
}

func CacheAgentStatus() error {
	ctx := context.Background()

	agents, err := GetAllAgent()
	if err != nil {
		return err
	}

	currentAgentIDs := make(map[string]struct{})
	for _, agent := range agents.Data.Agents {
		idStr := strconv.Itoa(agent.ID)
		currentAgentIDs[idStr] = struct{}{}

		err := rdb.SAdd(ctx, "agents:ids", idStr).Err()
		if err != nil {
			return fmt.Errorf("SAdd error: %w", err)
		}

		err = rdb.Set(ctx, fmt.Sprintf("agent:%s:is_online", idStr), agent.IsAvailable, 0).Err()
		if err != nil {
			return fmt.Errorf("Set is_online error: %w", err)
		}

		_, err = rdb.Get(ctx, fmt.Sprintf("agent:%s:customer_count", idStr)).Int()
		if err != nil {
			if err == redis.Nil {
				err = rdb.Set(ctx, fmt.Sprintf("agent:%s:customer_count", idStr), -1, 0).Err()
				if err != nil {
					return fmt.Errorf("Set customer_count error: %w", err)
				}
			} else {
				return fmt.Errorf("Get customer_count error: %w", err)
			}
		}
	}

	existingIDs, err := rdb.SMembers(ctx, "agents:ids").Result()
	if err != nil {
		return fmt.Errorf("SMembers error: %w", err)
	}

	for _, id := range existingIDs {
		if _, found := currentAgentIDs[id]; !found {
			rdb.Del(ctx, fmt.Sprintf("agent:%s:is_online", id))
			rdb.SRem(ctx, "agents:ids", id)
		}
	}

	return nil
}

func InitAgents() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := CacheAgentStatus(); err != nil {
				log.Println("Agent cache update failed:", err)
			}
		}
	}
}
