package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/redis/go-redis/v9"
)

func HandleIncomingMessage(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	var data WebhookIncomingMessageRequest
	err = json.Unmarshal(body, &data)
	if err != nil {
		http.Error(w, "Failed to parse request body", http.StatusBadRequest)
		return
	}

	task, err := NewChatAssignAgentTask(&data)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create task: %v", err), http.StatusInternalServerError)
		return
	}

	info, err := queueClient.Enqueue(task)
	if err != nil {
		http.Error(w, fmt.Sprintf("could not enqueue task: %v", err), http.StatusInternalServerError)
	}
	fmt.Printf("enqueued task: id=%s queue=%s", info.ID, info.Queue)

	return
}

func HandleGetAllAgent(w http.ResponseWriter, r *http.Request) {
	agents, err := GetAllAgent()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get all agents: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	err = json.NewEncoder(w).Encode(agents)
	if err != nil {
		http.Error(w, "Failed to encode agents response", http.StatusInternalServerError)
		return
	}
}

func HandlerGetWebhookConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	config, err := GetWebhookConfig(ctx)
	if err != nil {
		http.Error(w, "Failed to get webhook config", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(config); err != nil {
		http.Error(w, "Failed to encode webhook config", http.StatusInternalServerError)
		return
	}
}

func HandlerSetWebhook(w http.ResponseWriter, r *http.Request) {
	res, err := SetWebHookIncomingMessage(cfg.WebhookConfig.BaseUrl + WEBHOOK_INCOMING_MESSAGE_PATH)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to set webhook config: %v", err), http.StatusInternalServerError)
		return
	}

	res, err = SetWebHookMarkAsResolved(cfg.WebhookConfig.BaseUrl + WEBHOOK_MARK_AS_RESOLVED_PATH)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to set webhook config: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(res); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode webhook config response: %v", err), http.StatusInternalServerError)
		return
	}
}

func HandleMarkAsResolved(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	var data WebhookMarkAsResolvedRequest
	err = json.Unmarshal(body, &data)
	if err != nil {
		http.Error(w, "Failed to parse request body", http.StatusBadRequest)
		return
	}

	log.Printf("Webhook mark as resolved: %v", data)

	agentID := data.ResolvedBy.ID

	roomAgentKey := fmt.Sprintf("room:%s:agent", data.Service.RoomID)
	roomAgent, err := rdb.Get(ctx, roomAgentKey).Int()
	if err != nil && err != redis.Nil {
		log.Printf("Failed to find room %s", data.Service.RoomID)
		http.Error(w, "Failed to find room agent", http.StatusBadRequest)
		return
	}

	if roomAgent > 0 {
		log.Printf("Found %s:%d", roomAgentKey, roomAgent)
		agentID = roomAgent
	}

	customerCountKey := fmt.Sprintf("agent:%d:customer_count", agentID)
	customerCount, err := rdb.Get(ctx, customerCountKey).Int()
	if err != nil {
		if err == redis.Nil {
			return
		}
		log.Printf("Failed to find customer count of agent %d", agentID)
		http.Error(w, "Failed to find customer count key", http.StatusBadRequest)
		return
	}

	err = rdb.Decr(ctx, customerCountKey).Err()
	if err != nil {
		log.Printf("Failed to decreasing customer count of agent %d, from %d to %d", agentID, customerCount, customerCount-1)
		http.Error(w, "Failed to decrease customer count", http.StatusBadRequest)
		return
	}

	if roomAgent > 0 {
		rdb.Del(ctx, roomAgentKey)
	}

	log.Printf("Decreasing customer count of agent %d, from %d to %d", agentID, customerCount, customerCount-1)
	return
}
