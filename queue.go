package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/hibiken/asynq"
)

type QueueConfig struct {
	RedisURL string
}

func NewQueueConfig(redisURL string) *QueueConfig {
	return &QueueConfig{
		RedisURL: redisURL,
	}
}

const TypeChatAssignAgent = "chat:assign_agent"

func NewChatAssignAgentTask(wimr *WebhookIncomingMessageRequest) (*asynq.Task, error) {
	payload, err := json.Marshal(wimr)
	if err != nil {
		return nil, err
	}

	return asynq.NewTask(TypeChatAssignAgent, payload), nil
}

func HandleChatAssignAgentTask(ctx context.Context, task *asynq.Task) (err error) {
	var wimr WebhookIncomingMessageRequest
	if err = json.Unmarshal(task.Payload(), &wimr); err != nil {
		logger.Error(
			"failed to unmarshal task payload",
			slog.Any("error", err),
			slog.String("task_type", task.Type()),
		)
		return err
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		logger.Error(
			"failed to begin database transaction",
			slog.Any("error", err),
			slog.String("room_id", wimr.RoomID),
		)
		return err
	}

	isChatRoomExists, err := IsChatRoomExists(ctx, tx, wimr.RoomID)
	if err != nil {
		logger.Error(
			"failed to check if chat room exists",
			slog.Any("error", err),
			slog.String("room_id", wimr.RoomID),
		)
		tx.Rollback(ctx)
		return err
	}

	if isChatRoomExists {
		logger.Info(
			"chat room already exists, skipping creation",
			slog.String("room_id", wimr.RoomID),
		)
		tx.Rollback(ctx)
		return nil
	}

	err = CreateChat(ctx, tx, &wimr)
	if err != nil {
		logger.Error(
			"failed to create chat",
			slog.Any("error", err),
			slog.String("room_id", wimr.RoomID),
		)
		tx.Rollback(ctx)
		return err
	}

	availableAgentID, err := GetAvailableAgentWithCustomerCount(ctx, wimr.RoomID, int(cfg.WebhookConfig.MaxCurrentCustomer))
	if err != nil {
		logger.Error(
			"failed to find available agent",
			slog.Any("error", err),
			slog.String("room_id", wimr.RoomID),
			slog.Int("max_customer", int(cfg.WebhookConfig.MaxCurrentCustomer)),
		)
		tx.Rollback(ctx)
		return err
	}

	availableAgentIDInt, err := strconv.Atoi(availableAgentID)
	if err != nil {
		logger.Error(
			"failed to parse available agent id",
			slog.Any("error", err),
			slog.String("agent_id_str", availableAgentID),
			slog.String("room_id", wimr.RoomID),
		)
		tx.Rollback(ctx)
		return err
	}

	_, err = wimr.AssignAgent(availableAgentIDInt)
	if err != nil {
		logger.Error(
			"failed to assign agent",
			slog.Any("error", err),
			slog.Int("agent_id", availableAgentIDInt),
			slog.String("room_id", wimr.RoomID),
		)
		tx.Rollback(ctx)
		return err
	}

	customerCountKey := fmt.Sprintf("agent:%s:customer_count", availableAgentID)
	err = rdb.Incr(ctx, customerCountKey).Err()
	if err != nil {
		logger.Error(
			"failed to increase customer count",
			slog.Any("error", err),
			slog.String("customer_count_key", customerCountKey),
			slog.String("agent_id", availableAgentID),
		)
		tx.Rollback(ctx)
		return err
	}

	roomAgentKey := fmt.Sprintf("room:%s:agent", wimr.RoomID)
	err = rdb.Set(ctx, roomAgentKey, availableAgentID, 0).Err()
	if err != nil {
		logger.Error(
			"failed to set room agent mapping",
			slog.Any("error", err),
			slog.String("room_agent_key", roomAgentKey),
			slog.String("agent_id", availableAgentID),
			slog.String("room_id", wimr.RoomID),
		)
		tx.Rollback(ctx)
		return err
	}

	err = UpdateChat(ctx, tx, &wimr)
	if err != nil {
		logger.Error(
			"failed to update chat",
			slog.Any("error", err),
			slog.String("room_id", wimr.RoomID),
		)
		tx.Rollback(ctx)
		return err
	}

	// go func() {
	// 	delay := time.Duration(5+rand.Intn(6)) * time.Second

	// 	log.Printf("Waiting %s before resolving chat %s", delay, wimr.RoomID)
	// 	time.Sleep(delay)

	// 	wimr.Resolve("This is a test message to mark as resolved")
	// 	log.Printf("Room %s resolved", wimr.RoomID)
	// }()

	err = tx.Commit(ctx)
	if err != nil {
		logger.Error(
			"failed to commit transaction",
			slog.Any("error", err),
			slog.String("room_id", wimr.RoomID),
		)
		return err
	}

	logger.Info(
		"successfully handled chat assign agent task",
		slog.String("room_id", wimr.RoomID),
		slog.String("assigned_agent_id", availableAgentID),
	)

	return nil
}
