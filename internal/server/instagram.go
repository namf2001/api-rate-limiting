package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

// InstagramDownloadRequest represents the request body for the Instagram download endpoint
type InstagramDownloadRequest struct {
	URL string `json:"url" binding:"required"`
}

// InstagramDownloadResponse represents the response for the Instagram download endpoint
type InstagramDownloadResponse struct {
	DownloadURL string `json:"download_url,omitempty"`
	MediaType   string `json:"media_type,omitempty"`
	Error       string `json:"error,omitempty"`
}

// InstagramDownloadHandler handles requests to download Instagram media
func (s *Server) InstagramDownloadHandler(c *gin.Context) {
	var request InstagramDownloadRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, InstagramDownloadResponse{
			Error: "Invalid request: " + err.Error(),
		})
		return
	}

	// Validate Instagram URL
	if !isValidInstagramURL(request.URL) {
		c.JSON(http.StatusBadRequest, InstagramDownloadResponse{
			Error: "Invalid Instagram URL",
		})
		return
	}

	// Extract media URL
	downloadURL, mediaType, err := extractInstagramMediaURL(request.URL)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "private") {
			statusCode = http.StatusForbidden
		} else if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		}

		c.JSON(statusCode, InstagramDownloadResponse{
			Error: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, InstagramDownloadResponse{
		DownloadURL: downloadURL,
		MediaType:   mediaType,
	})
}

// isValidInstagramURL checks if the provided URL is a valid Instagram URL
func isValidInstagramURL(inputURL string) bool {
	// Basic validation for Instagram URLs
	// Support for posts, reels, TV, and stories
	instagramURLPattern := regexp.MustCompile(`^https?://(www\.)?instagram\.com/(p|reel|tv|stories/[\w.]+)/[\w-]+/?(\?.*)?$`)
	return instagramURLPattern.MatchString(inputURL)
}

// extractInstagramMediaURL extracts the direct media URL from an Instagram post URL
func extractInstagramMediaURL(instagramURL string) (string, string, error) {
	// Create an HTTP client with appropriate headers to mimic a browser
	client := &http.Client{}
	req, err := http.NewRequest("GET", instagramURL, nil)
	if err != nil {
		return "", "", fmt.Errorf("error creating request: %v", err)
	}

	// Set headers to mimic a browser request
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("error fetching Instagram page: %v", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode == http.StatusNotFound {
		return "", "", fmt.Errorf("media not found or has been deleted")
	} else if resp.StatusCode == http.StatusForbidden {
		return "", "", fmt.Errorf("media is from a private account")
	} else if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("error reading response body: %v", err)
	}

	bodyStr := string(body)

	// Extract JSON data from the page
	// Instagram embeds data in a script tag with type="application/ld+json"
	jsonDataRegex := regexp.MustCompile(`<script type="application/ld\+json">(.+?)</script>`)
	matches := jsonDataRegex.FindStringSubmatch(bodyStr)
	if len(matches) < 2 {
		// Try alternative method - look for shared_data
		sharedDataRegex := regexp.MustCompile(`window\._sharedData = (.+?);</script>`)
		sharedMatches := sharedDataRegex.FindStringSubmatch(bodyStr)
		if len(sharedMatches) < 2 {
			return "", "", fmt.Errorf("could not extract media data from Instagram page")
		}

		var sharedData map[string]interface{}
		if err := json.Unmarshal([]byte(sharedMatches[1]), &sharedData); err != nil {
			return "", "", fmt.Errorf("error parsing Instagram data: %v", err)
		}

		// Navigate through the shared data structure to find media URL
		entryData, ok := sharedData["entry_data"].(map[string]interface{})
		if !ok {
			return "", "", fmt.Errorf("could not find entry_data in Instagram response")
		}

		// Try to find PostPage or ReelPage
		var mediaData map[string]interface{}
		if postPage, ok := entryData["PostPage"].([]interface{}); ok && len(postPage) > 0 {
			mediaData = postPage[0].(map[string]interface{})
		} else if reelPage, ok := entryData["ReelPage"].([]interface{}); ok && len(reelPage) > 0 {
			mediaData = reelPage[0].(map[string]interface{})
		} else {
			return "", "", fmt.Errorf("could not find media data in Instagram response")
		}

		// Extract media URL from the structure
		// This is a simplified approach and might need adjustments based on Instagram's actual structure
		graphql, ok := mediaData["graphql"].(map[string]interface{})
		if !ok {
			return "", "", fmt.Errorf("could not find graphql data")
		}

		shortcodeMedia, ok := graphql["shortcode_media"].(map[string]interface{})
		if !ok {
			return "", "", fmt.Errorf("could not find shortcode_media data")
		}

		// Check if it's a video
		isVideo, ok := shortcodeMedia["is_video"].(bool)
		if ok && isVideo {
			videoURL, ok := shortcodeMedia["video_url"].(string)
			if ok {
				return videoURL, "video", nil
			}
		}

		// If not a video, try to get the image URL
		displayURL, ok := shortcodeMedia["display_url"].(string)
		if ok {
			return displayURL, "image", nil
		}

		return "", "", fmt.Errorf("could not extract media URL from Instagram data")
	}

	// Parse the JSON data
	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(matches[1]), &jsonData); err != nil {
		return "", "", fmt.Errorf("error parsing Instagram data: %v", err)
	}

	// Extract media URL based on content type
	contentURL, ok := jsonData["contentUrl"].(string)
	if !ok {
		// Try to get thumbnail URL as fallback for images
		thumbnailURL, ok := jsonData["thumbnailUrl"].(string)
		if !ok {
			return "", "", fmt.Errorf("could not find media URL in Instagram data")
		}
		return thumbnailURL, "image", nil
	}

	// Determine media type
	mediaType := "image"
	if strings.Contains(contentURL, ".mp4") || strings.Contains(contentURL, "/video/") {
		mediaType = "video"
	}

	// Decode URL if needed
	decodedURL, err := url.QueryUnescape(contentURL)
	if err != nil {
		// If decoding fails, use the original URL
		decodedURL = contentURL
	}

	return decodedURL, mediaType, nil
}
