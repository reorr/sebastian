package qiscus

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Resolve marks a room/service as resolved
func (c *Client) Resolve(roomID, lastCommentID, message string) error {
	encodedMessage := url.QueryEscape(message)
	data := strings.NewReader(fmt.Sprintf("room_id=%s&notes=%s&last_comment_id=%s", roomID, encodedMessage, lastCommentID))

	req, err := http.NewRequest("POST", c.config.BaseUrl+MARK_AS_RESOLVED_PATH, data)
	if err != nil {
		return err
	}
	
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Qiscus-App-Id", c.config.AppID)
	req.Header.Set("Qiscus-Secret-Key", c.config.SecretKey)

	resp, err := c.httpClient.Do(req)
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