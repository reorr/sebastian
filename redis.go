package main

import (
	"context"
	"fmt"
	"log/slog"
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
			logger.Error(
				"failed to get agent ids from redis",
				slog.Any("error", err),
				slog.String("room_id", roomID),
			)
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
					logger.Error(
						"failed to get agent online status",
						slog.Any("error", err),
						slog.String("is_online_key", isOnlineKey),
						slog.String("agent_id", id),
					)
					return agentID, fmt.Errorf("could not get is_online")
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
				logger.Error(
					"failed to get agent customer count",
					slog.Any("error", err),
					slog.String("customer_count_key", customerCountKey),
					slog.String("agent_id", id),
				)
				return agentID, fmt.Errorf("could not get customer count")
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
			logger.Info(
				"found available agent",
				slog.String("agent_id", agentID),
				slog.String("room_id", roomID),
				slog.Int("customer_count", agentCustomerCount),
			)
			return agentID, nil
		}

		if foundUnknownCustomerKey {
			agentID, customerCount, err := GetAndCacheAvailableAgentWithCustomerCount(ctx, roomID, maxCustomerCount)
			if err != nil {
				logger.Error(
					"failed to get and cache available agent",
					slog.Any("error", err),
					slog.String("room_id", roomID),
				)
			}
			if agentID != "" {
				logger.Info(
					"found agent from api source",
					slog.String("agent_id", agentID),
					slog.String("room_id", roomID),
					slog.Int("customer_count", customerCount),
				)
				return agentID, nil
			}
		}

		logger.Warn(
			"retrying agent allocation",
			slog.String("room_id", roomID),
			slog.Duration("elapsed", time.Since(start)),
		)
		time.Sleep(retryInterval)
	}
	return agentID, fmt.Errorf("can not find any available agent")
}

func GetAndCacheAvailableAgentWithCustomerCount(ctx context.Context, roomID string, maxCustomerCount int) (agentID string, agentCustomerCount int, err error) {
	availableAgents, err := GetAvailableAgent(roomID)
	if err != nil {
		logger.Error(
			"failed to get available agents from api",
			slog.Any("error", err),
			slog.String("room_id", roomID),
		)
		return agentID, agentCustomerCount, err
	}

	for _, agent := range availableAgents.Data.Agents {
		isOnlineKey := fmt.Sprintf("agent:%d:is_online", agent.ID)
		customerCountKey := fmt.Sprintf("agent:%d:customer_count", agent.ID)

		err = rdb.Set(ctx, isOnlineKey, true, 0).Err()
		if err != nil {
			logger.Error(
				"failed to set agent online status",
				slog.Any("error", err),
				slog.String("is_online_key", isOnlineKey),
				slog.Int("agent_id", agent.ID),
			)
			return agentID, agentCustomerCount, err
		}

		err = rdb.Set(ctx, customerCountKey, agent.CurrentCustomerCount, 0).Err()
		if err != nil {
			logger.Error(
				"failed to set agent customer count",
				slog.Any("error", err),
				slog.String("customer_count_key", customerCountKey),
				slog.Int("agent_id", agent.ID),
				slog.Int("customer_count", agent.CurrentCustomerCount),
			)
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
