package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/go-chi/chi/v5"
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
	r.Use(slogMiddleware)
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

func slogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Capture request body
		var bodyData string
		if r.Body != nil {
			bodyBytes, err := io.ReadAll(r.Body)
			if err == nil {
				bodyData = stripPII(string(bodyBytes))
				// Restore body for handler
				r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}
		}

		ww := &responseWriter{ResponseWriter: w, statusCode: 0}
		next.ServeHTTP(ww, r)

		// If no status was explicitly set, default to 200
		status := ww.statusCode
		if status == 0 {
			status = 200
		}

		logAttrs := []slog.Attr{
			slog.String("method", r.Method),
			slog.String("host", r.Host),
			slog.String("path", r.URL.Path),
			slog.String("remote_addr", r.RemoteAddr),
			slog.Int("status", status),
			slog.Int("bytes", ww.bytesWritten),
			slog.Duration("duration", time.Since(start)),
		}

		if bodyData != "" {
			logAttrs = append(logAttrs, slog.String("request_body", bodyData))
		}

		logger.LogAttrs(r.Context(), slog.LevelInfo, "http request", logAttrs...)
	})
}

func stripPII(body string) string {
	// Common PII patterns to redact
	patterns := map[string]string{
		`"password"\s*:\s*"[^"]*"`:      `"password":"[REDACTED]"`,
		`"email"\s*:\s*"[^"]*"`:         `"email":"[REDACTED]"`,
		`"phone"\s*:\s*"[^"]*"`:         `"phone":"[REDACTED]"`,
		`"token"\s*:\s*"[^"]*"`:         `"token":"[REDACTED]"`,
		`"api_key"\s*:\s*"[^"]*"`:       `"api_key":"[REDACTED]"`,
		`"secret"\s*:\s*"[^"]*"`:        `"secret":"[REDACTED]"`,
		`"authorization"\s*:\s*"[^"]*"`: `"authorization":"[REDACTED]"`,
	}

	result := body
	for pattern, replacement := range patterns {
		re := regexp.MustCompile(`(?i)` + pattern)
		result = re.ReplaceAllString(result, replacement)
	}

	// Limit body size to prevent huge logs
	if len(result) > 1000 {
		result = result[:1000] + "...[TRUNCATED]"
	}

	return result
}

type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += n
	return n, err
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
