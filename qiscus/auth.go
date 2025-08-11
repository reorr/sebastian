package qiscus

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/redis/go-redis/v9"
)

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Data struct {
		User struct {
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
			NotificationsRoomID interface{} `json:"notifications_room_id"`
			BubbleColor         interface{} `json:"bubble_color"`
			QismoKey            string      `json:"qismo_key"`
			DirectLoginToken    string      `json:"direct_login_token"`
			LastLogin           string      `json:"last_login"`
			ForceOffline        bool        `json:"force_offline"`
			DeletedAt           interface{} `json:"deleted_at"`
			IsTocAgree          bool        `json:"is_toc_agree"`
			TotpToken           string      `json:"totp_token"`
			IsReqOtpReset       interface{} `json:"is_req_otp_reset"`
			LastPasswordUpdate  string      `json:"last_password_update"`
			TypeAsString        string      `json:"type_as_string"`
			AssignedRules       interface{} `json:"assigned_rules"`
			App                 struct {
				AppCode                        string      `json:"app_code"`
				SecretKey                      string      `json:"secret_key"`
				Name                           string      `json:"name"`
				BotWebhookURL                  interface{} `json:"bot_webhook_url"`
				IsBotEnabled                   bool        `json:"is_bot_enabled"`
				IsAllocateAgentWebhookEnabled  bool        `json:"is_allocate_agent_webhook_enabled"`
				AllocateAgentWebhookURL        interface{} `json:"allocate_agent_webhook_url"`
				MarkAsResolvedWebhookURL       interface{} `json:"mark_as_resolved_webhook_url"`
				IsMarkAsResolvedWebhookEnabled bool        `json:"is_mark_as_resolved_webhook_enabled"`
				IsActive                       bool        `json:"is_active"`
				IsSessional                    bool        `json:"is_sessional"`
				IsAgentAllocationEnabled       bool        `json:"is_agent_allocation_enabled"`
				IsAgentTakeoverEnabled         bool        `json:"is_agent_takeover_enabled"`
				UseLatest                      bool        `json:"use_latest"`
				IsBulkAssignmentEnabled        bool        `json:"is_bulk_assignment_enabled"`
			} `json:"app"`
		} `json:"user"`
		Details struct {
			IsIntegrated bool `json:"is_integrated"`
			SdkUser      struct {
				ID          int    `json:"id"`
				Token       string `json:"token"`
				Email       string `json:"email"`
				DisplayName string `json:"display_name"`
				AvatarURL   string `json:"avatar_url"`
				Extras      struct {
					Type            string      `json:"type"`
					UserBubbleColor interface{} `json:"user_bubble_color"`
				} `json:"extras"`
			} `json:"sdk_user"`
			App struct {
				AppCode                        string      `json:"app_code"`
				SecretKey                      string      `json:"secret_key"`
				Name                           string      `json:"name"`
				BotWebhookURL                  interface{} `json:"bot_webhook_url"`
				IsBotEnabled                   bool        `json:"is_bot_enabled"`
				IsAllocateAgentWebhookEnabled  bool        `json:"is_allocate_agent_webhook_enabled"`
				AllocateAgentWebhookURL        interface{} `json:"allocate_agent_webhook_url"`
				MarkAsResolvedWebhookURL       interface{} `json:"mark_as_resolved_webhook_url"`
				IsMarkAsResolvedWebhookEnabled bool        `json:"is_mark_as_resolved_webhook_enabled"`
				IsActive                       bool        `json:"is_active"`
				IsSessional                    bool        `json:"is_sessional"`
				IsAgentAllocationEnabled       bool        `json:"is_agent_allocation_enabled"`
				IsAgentTakeoverEnabled         bool        `json:"is_agent_takeover_enabled"`
				UseLatest                      bool        `json:"use_latest"`
				IsBulkAssignmentEnabled        bool        `json:"is_bulk_assignment_enabled"`
			} `json:"app"`
		} `json:"details"`
		LongLivedToken string `json:"long_lived_token"`
		UserConfigs    struct {
			Notifagentjoining           interface{} `json:"notifagentjoining"`
			IsNotifagentjoiningEnabled  bool        `json:"is_notifagentjoining_enabled"`
			Notifmessagecoming          interface{} `json:"notifmessagecoming"`
			IsNotifmessagecomingEnabled bool        `json:"is_notifmessagecoming_enabled"`
		} `json:"user_configs"`
		Use2Fa       bool `json:"use_2fa"`
		NeedSetupOtp bool `json:"need_setup_otp"`
	} `json:"data"`
}

// NewLoginRequest creates a new login request
func NewLoginRequest(email, password string) *LoginRequest {
	return &LoginRequest{
		Email:    email,
		Password: password,
	}
}

// Login authenticates with the Qiscus API
func (c *Client) Login(email, password string) (*LoginResponse, error) {
	params := url.Values{}
	params.Set("email", email)
	params.Set("password", password)

	payload := bytes.NewBufferString(params.Encode())
	req, err := http.NewRequest("POST", c.config.BaseUrl+AUTH_PATH, payload)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to login, status code: %d", res.StatusCode)
	}

	var response LoginResponse
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

// GetToken gets cached authentication token or logs in
func (c *Client) GetToken(ctx context.Context) (string, error) {
	token, err := c.redis.Get(ctx, CACHE_TOKEN_KEY).Result()
	if err != nil {
		if err == redis.Nil {
			c.logger.Info("token not found in cache, logging in")
			
			loginResponse, err := c.Login(c.config.Email, c.config.Password)
			if err != nil {
				c.logger.Error(
					"failed to login",
					slog.Any("error", err),
				)
				return "", err
			}

			token = loginResponse.Data.User.AuthenticationToken
			err = c.redis.Set(ctx, CACHE_TOKEN_KEY, token, 0).Err()
			if err != nil {
				c.logger.Error(
					"failed to cache token",
					slog.Any("error", err),
				)
				return "", err
			}

			c.logger.Info("successfully logged in and cached token")
			return token, nil
		}

		c.logger.Error(
			"failed to get token from cache",
			slog.Any("error", err),
		)
		return "", err
	}

	return token, nil
}