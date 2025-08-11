package qiscus

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

type WebhookIncomingMessageRequest struct {
	AppID         string `json:"app_id"`
	Source        string `json:"source"`
	Name          string `json:"name"`
	Email         string `json:"email"`
	AvatarURL     string `json:"avatar_url"`
	Extras        string `json:"extras"`
	IsResolved    bool   `json:"is_resolved"`
	LatestService struct {
		ID                    int         `json:"id"`
		UserID                int         `json:"user_id"`
		RoomLogID             int         `json:"room_log_id"`
		AppID                 int         `json:"app_id"`
		RoomID                string      `json:"room_id"`
		Notes                 interface{} `json:"notes"`
		ResolvedAt            string      `json:"resolved_at"`
		IsResolved            bool        `json:"is_resolved"`
		CreatedAt             string      `json:"created_at"`
		UpdatedAt             string      `json:"updated_at"`
		FirstCommentID        string      `json:"first_comment_id"`
		LastCommentID         string      `json:"last_comment_id"`
		RetrievedAt           string      `json:"retrieved_at"`
		FirstCommentTimestamp interface{} `json:"first_comment_timestamp"`
	} `json:"latest_service"`
	RoomID         string `json:"room_id"`
	CandidateAgent struct {
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
		QismoKey            string      `json:"qismo_key"`
		DirectLoginToken    interface{} `json:"direct_login_token"`
		TypeAsString        string      `json:"type_as_string"`
		AssignedRules       []string    `json:"assigned_rules"`
	} `json:"candidate_agent"`
}

type WebhookMarkAsResolvedRequest struct {
	Service struct {
		ID             int         `json:"id"`
		RoomID         string      `json:"room_id"`
		IsResolved     bool        `json:"is_resolved"`
		Notes          interface{} `json:"notes"`
		FirstCommentID string      `json:"first_comment_id"`
		LastCommentID  string      `json:"last_comment_id"`
		Source         string      `json:"source"`
	} `json:"service"`
	ResolvedBy struct {
		ID          int    `json:"id"`
		Email       string `json:"email"`
		Name        string `json:"name"`
		Type        string `json:"type"`
		IsAvailable bool   `json:"is_available"`
	} `json:"resolved_by"`
	Customer struct {
		UserID string `json:"user_id"`
	} `json:"customer"`
}

type WebhookConfigResponse struct {
	Data struct {
		WebhookConfigs []struct {
			CreatedAt string `json:"created_at"`
			ID        int    `json:"id"`
			IsActive  bool   `json:"is_active"`
			Type      string `json:"type"`
			UpdatedAt string `json:"updated_at"`
			URL       string `json:"url"`
		} `json:"webhook_configs"`
	} `json:"data"`
	Status int `json:"status"`
}

type SetWebHookResponse struct {
	Data struct {
		ID                             int         `json:"id"`
		Name                           string      `json:"name"`
		AppCode                        string      `json:"app_code"`
		SecretKey                      string      `json:"secret_key"`
		CreatedAt                      string      `json:"created_at"`
		UpdatedAt                      string      `json:"updated_at"`
		BotWebhookURL                  string      `json:"bot_webhook_url"`
		IsBotEnabled                   bool        `json:"is_bot_enabled"`
		AllocateAgentWebhookURL        string      `json:"allocate_agent_webhook_url"`
		IsAllocateAgentWebhookEnabled  bool        `json:"is_allocate_agent_webhook_enabled"`
		MarkAsResolvedWebhookURL       string      `json:"mark_as_resolved_webhook_url"`
		IsMarkAsResolvedWebhookEnabled bool        `json:"is_mark_as_resolved_webhook_enabled"`
		IsMobilePnEnabled              bool        `json:"is_mobile_pn_enabled"`
		IsActive                       bool        `json:"is_active"`
		IsSessional                    bool        `json:"is_sessional"`
		IsAgentAllocationEnabled       bool        `json:"is_agent_allocation_enabled"`
		IsAgentTakeoverEnabled         bool        `json:"is_agent_takeover_enabled"`
		IsTokenExpiring                bool        `json:"is_token_expiring"`
		PaidChannelApproved            interface{} `json:"paid_channel_approved"`
		UseLatest                      bool        `json:"use_latest"`
		FreeSessions                   int         `json:"free_sessions"`
		IsForceSendBot                 bool        `json:"is_force_send_bot"`
	} `json:"data"`
}

// GetWebhookConfig gets current webhook configuration
func (c *Client) GetWebhookConfig(ctx context.Context) (*WebhookConfigResponse, error) {
	req, err := http.NewRequest("GET", c.config.BaseUrl+GET_WEBHOOK_CONFIG_PATH, nil)
	if err != nil {
		return nil, err
	}

	token, err := c.GetToken(ctx)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response WebhookConfigResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return &response, nil
}

// SetWebHookMarkAsResolved sets webhook URL for mark-as-resolved events
func (c *Client) SetWebHookMarkAsResolved(webhookUrl string) (*SetWebHookResponse, error) {
	params := url.Values{}
	params.Set("webhook_url", webhookUrl)
	params.Set("is_webhook_enabled", "true")

	payload := bytes.NewBufferString(params.Encode())

	req, err := http.NewRequest("POST", c.config.BaseUrl+SET_WEBHOOK_MARK_AS_RESOLVED, payload)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Qiscus-App-Id", c.config.AppID)
	req.Header.Set("Qiscus-Secret-Key", c.config.SecretKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response SetWebHookResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return &response, nil
}

// SetWebHookIncomingMessage sets webhook URL for incoming message events
func (c *Client) SetWebHookIncomingMessage(webhookUrl string) (*SetWebHookResponse, error) {
	params := url.Values{}
	params.Set("webhook_url", webhookUrl)
	params.Set("is_webhook_enabled", "true")

	payload := bytes.NewBufferString(params.Encode())

	req, err := http.NewRequest("POST", c.config.BaseUrl+SET_WEBHOOK_INCOMING_MESSAGE, payload)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Qiscus-App-Id", c.config.AppID)
	req.Header.Set("Qiscus-Secret-Key", c.config.SecretKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response SetWebHookResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return &response, nil
}