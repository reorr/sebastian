package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

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
	// time.Sleep(5 * time.Second)
	var wimr WebhookIncomingMessageRequest
	if err = json.Unmarshal(task.Payload(), &wimr); err != nil {
		return err
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}

	isChatRoomExists, err := IsChatRoomExists(ctx, tx, wimr.RoomID)
	if err != nil {
		fmt.Println("Error checking if chat room exists:", err)
		tx.Rollback(ctx)
		return err
	}

	if isChatRoomExists {
		fmt.Println(fmt.Sprintf("Chat room %s already exists, skipping creation", wimr.RoomID))
		tx.Rollback(ctx)
		return nil
	}

	err = CreateChat(ctx, tx, &wimr)
	if err != nil {
		fmt.Println("Error creating chat:", err)
		tx.Rollback(ctx)
		return err
	}

	availableAgentID, err := GetAvailableAgentWithCustomerCount(ctx, wimr.RoomID, int(cfg.WebhookConfig.MaxCurrentCustomer))
	if err != nil {
		fmt.Println("Error find available agent:", err)
		tx.Rollback(ctx)
		return err
	}

	availableAgentIDInt, err := strconv.Atoi(availableAgentID)
	if err != nil {
		fmt.Println("Error parsing available agent id:", err)
		tx.Rollback(ctx)
		return err
	}

	_, err = wimr.AssignAgent(availableAgentIDInt)
	if err != nil {
		fmt.Println("Error allocating agent:", err)
		tx.Rollback(ctx)
		return err
	}

	customerCountKey := fmt.Sprintf("agent:%s:customer_count", availableAgentID)
	err = rdb.Incr(ctx, customerCountKey).Err()
	if err != nil {
		fmt.Println("Error increasing customer count:", err)
		tx.Rollback(ctx)
		return err
	}

	roomAgentKey := fmt.Sprintf("room:%s:agent", wimr.RoomID)
	err = rdb.Set(ctx, roomAgentKey, availableAgentID, 0).Err()
	if err != nil {
		fmt.Println("Error set room agent:", err)
		tx.Rollback(ctx)
		return err
	}

	err = UpdateChat(ctx, tx, &wimr)
	if err != nil {
		fmt.Println("Error updating chat:", err)
		tx.Rollback(ctx)
		return
	}

	go func() {
		time.Sleep(15 * time.Second)
		wimr.Resolve("This is a test message to mark as resolved")
	}()

	err = tx.Commit(ctx)
	if err != nil {
		fmt.Println("Error committing transaction:", err)
		return err
	}

	println("Handling chat assign agent task for:", wimr.RoomID)

	return nil
}
