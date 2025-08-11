package qiscus

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
)

type GetAllAgentResponse struct {
	Data struct {
		Agents []Agent `json:"agents"`
		Meta   struct {
			After      string      `json:"after"`
			Before     interface{} `json:"before"`
			PerPage    int         `json:"per_page"`
			TotalCount int         `json:"total_count"`
		} `json:"meta"`
	} `json:"data"`
	Status int `json:"status"`
}

type Agent struct {
	AvatarURL            string      `json:"avatar_url"`
	CreatedAt            string      `json:"created_at"`
	CurrentCustomerCount int         `json:"current_customer_count"`
	Email                string      `json:"email"`
	ForceOffline         bool        `json:"force_offline"`
	ID                   int         `json:"id"`
	IsAvailable          bool        `json:"is_available"`
	IsReqOtpReset        interface{} `json:"is_req_otp_reset"`
	LastLogin            interface{} `json:"last_login"`
	Name                 string      `json:"name"`
	SdkEmail             string      `json:"sdk_email"`
	SdkKey               string      `json:"sdk_key"`
	Type                 int         `json:"type"`
	TypeAsString         string      `json:"type_as_string"`
	UserChannels         []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"user_channels"`
	UserRoles []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"user_roles"`
}

type GetAvailableAgentResponse struct {
	Data struct {
		Agents []AvailableAgent `json:"agents"`
	} `json:"data"`
	Meta struct {
		After      interface{} `json:"after"`
		Before     interface{} `json:"before"`
		PerPage    int         `json:"per_page"`
		TotalCount interface{} `json:"total_count"`
	} `json:"meta"`
	Status int `json:"status"`
}

type AvailableAgent struct {
	AvatarURL            string      `json:"avatar_url"`
	CreatedAt            string      `json:"created_at"`
	CurrentCustomerCount int         `json:"current_customer_count"`
	Email                string      `json:"email"`
	ForceOffline         bool        `json:"force_offline"`
	ID                   int         `json:"id"`
	IsAvailable          bool        `json:"is_available"`
	IsReqOtpReset        interface{} `json:"is_req_otp_reset"`
	LastLogin            interface{} `json:"last_login"`
	Name                 string      `json:"name"`
	SdkEmail             string      `json:"sdk_email"`
	SdkKey               string      `json:"sdk_key"`
	Type                 int         `json:"type"`
	TypeAsString         string      `json:"type_as_string"`
	UserChannels         []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"user_channels"`
	UserRoles []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"user_roles"`
}

type AssignAgentResponse struct {
	Data struct {
		AddedAgent struct {
			ID                  int         `json:"id"`
			Name                string      `json:"name"`
			Email               string      `json:"email"`
			AuthenticationToken string      `json:"authentication_token"`
			CreatedAt           string      `json:"created_at"`
			UpdatedAt           string      `json:"updated_at"`
			SdkEmail            string      `json:"sdk_email"`
			SdkKey              string      `json:"sdk_key"`
			IsAvailable         bool        `json:"is_available"`
			Type                int         `json:"type"`
			AvatarURL           string      `json:"avatar_url"`
			AppID               int         `json:"app_id"`
			IsVerified          bool        `json:"is_verified"`
			NotificationsRoomID string      `json:"notifications_room_id"`
			BubbleColor         string      `json:"bubble_color"`
			QismoKey            interface{} `json:"qismo_key"`
			TypeAsString        string      `json:"type_as_string"`
			AssignedRules       []string    `json:"assigned_rules"`
		} `json:"added_agent"`
	} `json:"data"`
}

type AllocateAssignAgentResponse struct {
	Data struct {
		Agent struct {
			ID           int         `json:"id"`
			Name         string      `json:"name"`
			SdkEmail     string      `json:"sdk_email"`
			Email        string      `json:"email"`
			SdkKey       string      `json:"sdk_key"`
			Type         int         `json:"type"`
			IsAvailable  bool        `json:"is_available"`
			AvatarURL    interface{} `json:"avatar_url"`
			IsVerified   bool        `json:"is_verified"`
			ForceOffline bool        `json:"force_offline"`
			Count        int         `json:"count"`
		} `json:"agent"`
	} `json:"data"`
}

// GetAllAgent fetches all agents from Qiscus API
func (c *Client) GetAllAgent() (*GetAllAgentResponse, error) {
	req, err := http.NewRequest("GET", c.config.BaseUrl+GET_ALL_AGENT_PATH, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Qiscus-App-Id", c.config.AppID)
	req.Header.Set("Qiscus-Secret-Key", c.config.SecretKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error(
			"failed to make get all agents request",
			slog.Any("error", err),
			slog.String("url", c.config.BaseUrl+GET_ALL_AGENT_PATH),
		)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get all agents, status code: %d", resp.StatusCode)
	}

	var response GetAllAgentResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		c.logger.Error(
			"failed to decode get all agents response",
			slog.Any("error", err),
			slog.Int("status_code", resp.StatusCode),
		)
		return nil, err
	}

	return &response, nil
}

// GetAvailableAgent gets available agents for a specific room
func (c *Client) GetAvailableAgent(roomID string) (*GetAvailableAgentResponse, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s%s?room_id=%s", c.config.BaseUrl, GET_AVAILABLE_AGENT_PATH, roomID), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Qiscus-App-Id", c.config.AppID)
	req.Header.Set("Qiscus-Secret-Key", c.config.SecretKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error(
			"failed to make get available agent request",
			slog.Any("error", err),
			slog.String("room_id", roomID),
			slog.String("url", fmt.Sprintf("%s%s?room_id=%s", c.config.BaseUrl, GET_AVAILABLE_AGENT_PATH, roomID)),
		)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get available agents, status code: %d", resp.StatusCode)
	}

	var response GetAvailableAgentResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		c.logger.Error(
			"failed to decode get available agent response",
			slog.Any("error", err),
			slog.String("room_id", roomID),
			slog.Int("status_code", resp.StatusCode),
		)
		return nil, err
	}

	return &response, nil
}

// AssignAgent assigns an agent to a room
func (c *Client) AssignAgent(roomID string, agentID int) (*AssignAgentResponse, error) {
	params := url.Values{}
	params.Set("room_id", roomID)
	params.Set("agent_id", fmt.Sprintf("%d", agentID))
	params.Set("max_agent", "1")

	payload := bytes.NewBufferString(params.Encode())
	req, err := http.NewRequest("POST", c.config.BaseUrl+ASSIGN_AGENT_PATH, payload)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Qiscus-App-Id", c.config.AppID)
	req.Header.Set("Qiscus-Secret-Key", c.config.SecretKey)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to assign agent, status code: %d", res.StatusCode)
	}

	var response AssignAgentResponse
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

// AllocateAssignAgent auto-allocates and assigns an agent to a room
func (c *Client) AllocateAssignAgent(roomID string) (*AllocateAssignAgentResponse, error) {
	params := url.Values{}
	params.Set("room_id", roomID)
	params.Set("ignore_agent_availability", "false")

	payload := bytes.NewBufferString(params.Encode())
	req, err := http.NewRequest("POST", c.config.BaseUrl+ALLOCATE_ASSIGN_AGENT_PATH, payload)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Qiscus-App-Id", c.config.AppID)
	req.Header.Set("Qiscus-Secret-Key", c.config.SecretKey)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to allocate and assign agent, status code: %d", res.StatusCode)
	}

	var response AllocateAssignAgentResponse
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}