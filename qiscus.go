package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
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

func NewLoginRequest(email, password string) *LoginRequest {
	return &LoginRequest{
		Email:    email,
		Password: password,
	}
}

func (r *LoginRequest) Login() (*LoginResponse, error) {
	client := &http.Client{}

	params := url.Values{}
	params.Set("email", r.Email)
	params.Set("password", r.Password)

	payload := bytes.NewBufferString(params.Encode())
	req, err := http.NewRequest("POST", cfg.QiscusConfig.BaseUrl+AUTH_PATH, payload)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := client.Do(req)
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

func GetAllAgent() (*GetAllAgentResponse, error) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", cfg.QiscusConfig.BaseUrl+GET_ALL_AGENT_PATH, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Qiscus-App-Id", cfg.QiscusConfig.AppID)
	req.Header.Set("Qiscus-Secret-Key", cfg.QiscusConfig.SecretKey)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get all agents, status code: %d", resp.StatusCode)
	}

	var response GetAllAgentResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		fmt.Println("Error decoding response:", err)
		return nil, err
	}

	return &response, nil
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

func GetAvailableAgent(roomID string) (*GetAvailableAgentResponse, error) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", fmt.Sprintf("%s%s?room_id=%s", cfg.QiscusConfig.BaseUrl, GET_AVAILABLE_AGENT_PATH, roomID), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Qiscus-App-Id", cfg.QiscusConfig.AppID)
	req.Header.Set("Qiscus-Secret-Key", cfg.QiscusConfig.SecretKey)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get all agents, status code: %d", resp.StatusCode)
	}

	var response GetAvailableAgentResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		fmt.Println("Error decoding response:", err)
		return nil, err
	}

	return &response, nil
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

func (wimr *WebhookIncomingMessageRequest) AssignAgent(agentID int) (*AssignAgentResponse, error) {
	client := &http.Client{}

	params := url.Values{}
	params.Set("room_id", wimr.RoomID)
	params.Set("agent_id", fmt.Sprintf("%d", agentID))
	params.Set("max_agent", "1")

	payload := bytes.NewBufferString(params.Encode())
	req, err := http.NewRequest("POST", cfg.QiscusConfig.BaseUrl+ASSIGN_AGENT_PATH, payload)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Qiscus-App-Id", cfg.QiscusConfig.AppID)
	req.Header.Set("Qiscus-Secret-Key", cfg.QiscusConfig.SecretKey)

	res, err := client.Do(req)
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

func GetWebhookConfig(ctx context.Context) (*WebhookConfigResponse, error) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", cfg.QiscusConfig.BaseUrl+GET_WEBHOOK_CONFIG_PATH, nil)
	if err != nil {
		return nil, err
	}

	token, err := getToken(ctx)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)

	resp, err := client.Do(req)
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
func (wimr *WebhookIncomingMessageRequest) Resolve(message string) error {
	client := &http.Client{}

	encodedMessage := url.QueryEscape(message)
	var data = strings.NewReader(fmt.Sprintf("room_id=%s&notes=%s&last_comment_id=%s", wimr.RoomID, encodedMessage, wimr.LatestService.LastCommentID))

	req, err := http.NewRequest("POST", cfg.QiscusConfig.BaseUrl+MARK_AS_RESOLVED_PATH, data)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Qiscus-App-Id", cfg.QiscusConfig.AppID)
	req.Header.Set("Qiscus-Secret-Key", cfg.QiscusConfig.SecretKey)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return nil
}

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

func SetWebHookMarkAsResolved(webhookUrl string) (*SetWebHookResponse, error) {
	client := &http.Client{}

	params := url.Values{}
	params.Set("webhook_url", webhookUrl)
	params.Set("is_webhook_enabled", "true")

	payload := bytes.NewBufferString(params.Encode())

	req, err := http.NewRequest("POST", cfg.QiscusConfig.BaseUrl+SET_WEBHOOK_MARK_AS_RESOLVED, payload)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Qiscus-App-Id", cfg.QiscusConfig.AppID)
	req.Header.Set("Qiscus-Secret-Key", cfg.QiscusConfig.SecretKey)

	resp, err := client.Do(req)
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

func SetWebHookIncomingMessage(webhookUrl string) (*SetWebHookResponse, error) {
	client := &http.Client{}

	params := url.Values{}
	params.Set("webhook_url", webhookUrl)
	params.Set("is_webhook_enabled", "true")

	payload := bytes.NewBufferString(params.Encode())

	req, err := http.NewRequest("POST", cfg.QiscusConfig.BaseUrl+SET_WEBHOOK_INCOMING_MESSAGE, payload)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Qiscus-App-Id", cfg.QiscusConfig.AppID)
	req.Header.Set("Qiscus-Secret-Key", cfg.QiscusConfig.SecretKey)

	resp, err := client.Do(req)
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

func (wimr *WebhookIncomingMessageRequest) AllocateAssignAgent() (*AllocateAssignAgentResponse, error) {
	client := &http.Client{}

	params := url.Values{}
	params.Set("room_id", wimr.RoomID)
	params.Set("ignore_agent_availability", "false")

	payload := bytes.NewBufferString(params.Encode())
	req, err := http.NewRequest("POST", cfg.QiscusConfig.BaseUrl+ALLOCATE_ASSIGN_AGENT_PATH, payload)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Qiscus-App-Id", cfg.QiscusConfig.AppID)
	req.Header.Set("Qiscus-Secret-Key", cfg.QiscusConfig.SecretKey)

	res, err := client.Do(req)
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
