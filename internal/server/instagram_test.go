package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestInstagramDownloadHandler(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a new server instance
	s := &Server{}

	// Create a new Gin router
	r := gin.New()
	r.POST("/instagram/download", s.InstagramDownloadHandler)

	// Test cases
	testCases := []struct {
		name           string
		requestURL     string
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "Invalid URL format",
			requestURL:     "not-a-url",
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response InstagramDownloadResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				if err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if response.Error == "" {
					t.Errorf("Expected error message in response, got empty string")
				}
			},
		},
		{
			name:           "Non-Instagram URL",
			requestURL:     "https://www.instagram.com/stories/fcbarcelona/3661468037567118112/",
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response InstagramDownloadResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				if err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if response.Error != "Invalid Instagram URL" {
					t.Errorf("Expected 'Invalid Instagram URL' error, got: %s", response.Error)
				}
			},
		},
		{
			name:           "Instagram Story URL",
			requestURL:     "https://www.instagram.com/stories/fcbarcelona/3661468037567118112/",
			expectedStatus: http.StatusInternalServerError, // This will likely fail with an internal error since we can't actually access the content in a test
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				// Just check that it's not rejected as an invalid URL format
				var response InstagramDownloadResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				if err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if response.Error == "Invalid Instagram URL" {
					t.Errorf("URL should be recognized as valid Instagram URL but got: %s", response.Error)
				}
			},
		},
		// Note: We can't reliably test actual Instagram URLs in unit tests
		// as they require network access and Instagram's response format may change.
		// These would be better tested in integration tests.
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create request body
			requestBody, err := json.Marshal(InstagramDownloadRequest{URL: tc.requestURL})
			if err != nil {
				t.Fatalf("Failed to marshal request: %v", err)
			}

			// Create request
			req, err := http.NewRequest("POST", "/instagram/download", bytes.NewBuffer(requestBody))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			w := httptest.NewRecorder()

			// Serve the request
			r.ServeHTTP(w, req)

			// Check status code
			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, w.Code)
			}

			// Run custom response checks
			tc.checkResponse(t, w)
		})
	}
}

// This is a manual test function that can be run to test with real Instagram URLs
// It's commented out because it requires network access and shouldn't be run in automated tests
/*
func TestWithRealInstagramURL(t *testing.T) {
	// Skip in automated testing
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a new server instance
	s := &Server{}

	// Create a new Gin router
	r := gin.New()
	r.POST("/instagram/download", s.InstagramDownloadHandler)

	// Create request with a real Instagram URL
	// Replace with a valid Instagram URL for testing
	requestBody, _ := json.Marshal(InstagramDownloadRequest{URL: "https://www.instagram.com/p/EXAMPLE_POST_ID/"})
	req, _ := http.NewRequest("POST", "/instagram/download", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	// Create response recorder
	w := httptest.NewRecorder()

	// Serve the request
	r.ServeHTTP(w, req)

	// Print the response for manual inspection
	t.Logf("Status: %d", w.Code)
	t.Logf("Body: %s", w.Body.String())
}
*/
