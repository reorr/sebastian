package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
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
	ALLOCATE_AGENT_PATH          = "/api/v1/admin/service/allocate_agent"
	ALLOCATE_ASSIGN_AGENT_PATH   = "/api/v1/admin/service/allocate_assign_agent"
	ASSIGN_AGENT_PATH            = "/api/v1/admin/service/assign_agent"
	GET_AVAILABLE_AGENT_PATH     = "/api/v2/admin/service/available_agents"
	MARK_AS_RESOLVED_PATH        = "/api/v1/admin/service/mark_as_resolved"
	GET_ALL_AGENT_PATH           = "/api/v2/admin/agents?limit=1000"
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
	logger      *slog.Logger
	err         error
)

func main() {
	logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

	var configFileName, exec string
	flag.StringVar(&exec, "e", "webhook", "Service to run. Use 'webhook' or 'worker'.")
	flag.StringVar(&configFileName, "c", "config.yml", "Config file name")

	flag.Parse()

	cfg.loadFromEnv()

	if len(configFileName) > 0 {
		err := loadConfigFromFile(configFileName, &cfg)
		if err != nil {
			logger.Warn(
				"can not load config from file",
				slog.String("file name", configFileName),
				slog.Any("error", err),
			)
		}
	}

	logger.Debug(
		"config loaded",
		slog.Any("config", cfg),
	)

	ctx := context.Background()

	logger.Info(
		"serving with webhook base url",
		slog.String("url", cfg.WebhookConfig.BaseUrl),
	)

	rdb = redis.NewClient(&redis.Options{
		Addr: cfg.RedisConfig.Url,
	})
	if rdb == nil {
		err = errors.New("could not connect to redis")
		logger.Error(
			err.Error(),
			slog.Any("error", err),
		)
		panic(err)
	}

	queueClient = asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.RedisConfig.Url})
	if queueClient == nil {
		err = errors.New("could not connect to redis queue")
		logger.Error(
			err.Error(),
			slog.Any("error", err),
		)
		panic(err)
	}

	pool, err = pgxpool.New(ctx, cfg.DBConfig.ConnectionString)
	if err != nil {
		logger.Error(
			"could not connect to database",
			slog.Any("error", err),
		)
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
	r.Get("/agents", HandleGetAllAgent)
	r.Post("/set-webhook", HandlerSetWebhook)

	listenPort := fmt.Sprintf(":%d", port)
	logger.Info(
		"webhook running",
		slog.String("port", listenPort),
	)

	http.ListenAndServe(listenPort, r)
}

func runWorker() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger.Info("starting worker")

	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: cfg.RedisConfig.Url},
		asynq.Config{
			Concurrency: 1,
		},
	)

	if err := CacheAgentStatus(ctx); err != nil {
		logger.Error(
			"could not initiate populate agent",
			slog.Any("error", err),
		)
		panic(fmt.Errorf("initial agent cache update failed: %w", err))
	}
	InitAgents(ctx)

	mux := asynq.NewServeMux()
	mux.HandleFunc(TypeChatAssignAgent, HandleChatAssignAgentTask)

	if err := srv.Run(mux); err != nil {
		logger.Error(
			"failed to start worker",
			slog.Any("error", err),
		)
		panic(fmt.Sprintf("could not start worker: %v", err))
	}
}

func CacheAgentStatus(ctx context.Context) error {

	agents, err := GetAllAgent()
	if err != nil {
		logger.Error(
			"could not get agents to cache",
			slog.Any("error", err),
		)
		return err
	}

	currentAgentIDs := make(map[string]struct{})
	for _, agent := range agents.Data.Agents {
		idStr := strconv.Itoa(agent.ID)
		currentAgentIDs[idStr] = struct{}{}

		err := rdb.SAdd(ctx, "agents:ids", idStr).Err()
		if err != nil {
			logger.Error(
				"could not cache agent id",
				slog.String("agent_id", idStr),
				slog.Any("error", err),
			)
			return fmt.Errorf("SAdd error: %w", err)
		}

		err = rdb.Set(ctx, fmt.Sprintf("agent:%s:is_online", idStr), agent.IsAvailable, 0).Err()
		if err != nil {
			logger.Error(
				"could not cache agent online status",
				slog.String("agent_id", idStr),
				slog.Any("error", err),
			)
			return fmt.Errorf("set is_online error: %w", err)
		}

		_, err = rdb.Get(ctx, fmt.Sprintf("agent:%s:customer_count", idStr)).Int()
		if err != nil {
			if err == redis.Nil {
				err = rdb.Set(ctx, fmt.Sprintf("agent:%s:customer_count", idStr), -1, 0).Err()
				if err != nil {
					logger.Error(
						"could not cache initiate agent customer count",
						slog.String("agent_id", idStr),
						slog.Any("error", err),
					)
					return fmt.Errorf("set customer_count error: %w", err)
				}
			} else {
				logger.Error(
					"could not get cache agents customer count",
					slog.String("agent_id", idStr),
					slog.Any("error", err),
				)
				return fmt.Errorf("get customer_count error: %w", err)
			}
		}
	}

	existingIDs, err := rdb.SMembers(ctx, "agents:ids").Result()
	if err != nil {
		logger.Error(
			"could not get cache agent ids",
			slog.Any("error", err),
		)
		return fmt.Errorf("SMembers error: %w", err)
	}

	for _, id := range existingIDs {
		if _, found := currentAgentIDs[id]; !found {
			err = rdb.Del(ctx, fmt.Sprintf("agent:%s:is_online", id)).Err()
			if err != nil {
				logger.Error(
					"could not remove cache agent online status",
					slog.String("agent_id", id),
					slog.Any("error", err),
				)
			}
			err = rdb.SRem(ctx, "agents:ids", id).Err()
			if err != nil {
				logger.Error(
					"could not remove cache agent id",
					slog.String("agent_id", id),
					slog.Any("error", err),
				)
			}
		}
	}

	return nil
}

func InitAgents(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)

	go func() {
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := CacheAgentStatus(ctx); err != nil {
					logger.Error(
						"agent cache update failed",
						slog.Any("error", err),
					)
				}
				logger.Info("agent cache updated")
			case <-ctx.Done():
				logger.Info("stopping agent status updater")
				return
			}
		}
	}()
}
