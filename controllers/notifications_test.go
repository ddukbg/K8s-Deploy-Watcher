package controllers

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSlackNotification(t *testing.T) {
	tests := []struct {
		name        string
		serverFunc  func(w http.ResponseWriter, r *http.Request)
		expectError bool
	}{
		{
			name: "성공적인 Slack 알림",
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
				w.WriteHeader(http.StatusOK)
			},
			expectError: false,
		},
		{
			name: "Slack 서버 에러",
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 테스트 서버 생성
			server := httptest.NewServer(http.HandlerFunc(tt.serverFunc))
			defer server.Close()

			// 알림 전송 테스트
			err := sendSlackNotification(server.URL, "test message")

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
