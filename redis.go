package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisConfig struct {
	RedisUrl string
}

func NewRedisConfig(redisUrl string) *RedisConfig {
	return &RedisConfig{
		RedisUrl: redisUrl,
	}
}

func getToken(ctx context.Context) (string, error) {
	cachedToken, err := rdb.Get(ctx, CACHE_TOKEN_KEY).Result()
	if err != nil && err != redis.Nil {
		return "", err
	}
	if cachedToken != "" {
		return cachedToken, nil
	}

	login := NewLoginRequest("wefeho7425@linacit.com", "RTwEe111")
	tokenResponse, err := login.Login()
	if err != nil {
		return "", err
	}

	token := tokenResponse.Data.User.AuthenticationToken

	err = rdb.Set(ctx, CACHE_TOKEN_KEY, token, 3600).Err()
	if err != nil {
		return "", err
	}

	return token, nil
}

type CachedAgent struct {
	ID                   string
	CurrentCustomerCount int
}

func GetCachedAvailableAgents(ctx context.Context) ([]CachedAgent, error) {
	var availableAgents []CachedAgent

	agentIDs, err := rdb.SMembers(ctx, "agent:ids").Result()
	if err != nil {
		return nil, fmt.Errorf("error getting agent IDs: %w", err)
	}

	for _, id := range agentIDs {
		key := fmt.Sprintf("agent:%s:customer_count", id)
		countStr, err := rdb.Get(ctx, key).Result()
		if err != nil {
			if err == redis.Nil {
				availableAgents = append(availableAgents, CachedAgent{
					ID: id, CurrentCustomerCount: 0,
				})
			} else {
				return nil, fmt.Errorf("error getting customer count for %s: %w", id, err)
			}
			continue
		}

		count, _ := strconv.Atoi(countStr)
		if count < int(cfg.WebhookConfig.MaxCurrentCustomer) {
			availableAgents = append(availableAgents, CachedAgent{
				ID: id, CurrentCustomerCount: count,
			})
		}
	}

	return availableAgents, nil
}

func GetAvailableAgentWithCustomerCount(ctx context.Context, roomID string, maxCustomerCount int) (agentID string, err error) {
	const maxRetryDuration = 10 * time.Minute
	const retryInterval = 10 * time.Second

	start := time.Now()

	for time.Since(start) < maxRetryDuration {
		agentIDs, err := rdb.SMembers(ctx, "agents:ids").Result()
		if err != nil {
			log.Printf("Error getting agents %v", err)
			return agentID, fmt.Errorf("SMembers error: %w", err)
		}

		foundUnknownCustomerKey := false
		agentCustomerCount := 0
		for _, id := range agentIDs {
			isOnlineKey := fmt.Sprintf("agent:%s:is_online", id)
			customerCountKey := fmt.Sprintf("agent:%s:customer_count", id)

			isOnline, err := rdb.Get(ctx, isOnlineKey).Bool()
			if err != nil {
				if err != redis.Nil {
					log.Printf("Error getting online status %s err: %v", isOnlineKey, err)
					return agentID, fmt.Errorf("Could not get is_online")
				}
			}

			if !isOnline {
				continue
			}

			customerCount, err := rdb.Get(ctx, customerCountKey).Int()
			if err != nil {
				if err != redis.Nil {
					foundUnknownCustomerKey = true
					continue
				}
				log.Printf("Error getting current customer count %s err: %v", customerCountKey, err)
				return agentID, fmt.Errorf("Could not get customer count")
			}

			if customerCount == -1 {
				foundUnknownCustomerKey = true
				continue
			}

			if customerCount < maxCustomerCount {
				if agentID == "" {
					agentCustomerCount = customerCount
					agentID = id
					continue
				}
				if customerCount <= agentCustomerCount {
					agentCustomerCount = customerCount
					agentID = id
				}
			}
		}

		if agentID != "" {
			log.Printf("Found agent %s for room %s with current customer count %d", agentID, roomID, agentCustomerCount)
			return agentID, nil
		}

		if foundUnknownCustomerKey {
			agentID, customerCount, err := GetAndCacheAvailableAgentWithCustomerCount(ctx, roomID, maxCustomerCount)
			if err != nil {
				fmt.Println("Can not call available agent")
			}
			if agentID != "" {
				log.Printf("Found agent id %s for room %s with customer count %d from source", agentID, roomID, customerCount)
				return agentID, nil
			}
		}

		log.Printf("Retrying allocate agent for room %s", roomID)
		time.Sleep(retryInterval)
	}
	return agentID, fmt.Errorf("Can not find any available agent")
}

func GetAndCacheAvailableAgentWithCustomerCount(ctx context.Context, roomID string, maxCustomerCount int) (agentID string, agentCustomerCount int, err error) {
	availableAgents, err := GetAvailableAgent(roomID)
	if err != nil {
		log.Printf("Error getting available agents: %v", err)
		return agentID, agentCustomerCount, err
	}

	for _, agent := range availableAgents.Data.Agents {
		isOnlineKey := fmt.Sprintf("agent:%d:is_online", agent.ID)
		customerCountKey := fmt.Sprintf("agent:%d:customer_count", agent.ID)

		err = rdb.Set(ctx, isOnlineKey, true, 0).Err()
		if err != nil {
			log.Printf("Error set %s err: %v", isOnlineKey, err)
			return agentID, agentCustomerCount, err
		}

		err = rdb.Set(ctx, customerCountKey, agent.CurrentCustomerCount, 0).Err()
		if err != nil {
			log.Printf("Error set %s err: %v", customerCountKey, err)
			return agentID, agentCustomerCount, err
		}

		if agent.CurrentCustomerCount < maxCustomerCount {
			if agentID == "" {
				agentCustomerCount = agent.CurrentCustomerCount
				agentID = fmt.Sprintf("%d", agent.ID)
				continue
			}
			if agent.CurrentCustomerCount <= agentCustomerCount {
				agentCustomerCount = agent.CurrentCustomerCount
				agentID = fmt.Sprintf("%d", agent.ID)
			}
		}
	}

	return agentID, agentCustomerCount, nil
}
