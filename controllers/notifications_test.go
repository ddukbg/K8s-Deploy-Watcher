package controllers

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSlackNotification(t *testing.T) {
	// 원래 함수 백업
	originalSendSlack := sendSlackNotification
	defer func() {
		sendSlackNotification = originalSendSlack
	}()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	// 실제 HTTP 요청을 하는 함수로 교체
	sendSlackNotification = func(webhookURL string, message string) error {
		resp, err := http.Post(server.URL, "application/json", bytes.NewBuffer([]byte(message)))
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("slack notification failed: %d", resp.StatusCode)
		}
		return nil
	}

	err := sendSlackNotification(server.URL, "test message")
	assert.Error(t, err)
}
