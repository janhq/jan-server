package responseapi_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

var (
	baseURL    string
	httpClient *http.Client
)

func init() {
	baseURL = os.Getenv("TEST_RESPONSE_API_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8082"
	}

	httpClient = &http.Client{
		Timeout: 30 * time.Second,
	}
}

// skipIfNoAPI skips the test if the API is not reachable
func skipIfNoAPI(t *testing.T) {
	t.Helper()
	resp, err := httpClient.Get(baseURL + "/health")
	if err != nil {
		t.Skipf("API not reachable at %s: %v", baseURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Skipf("API health check failed: %d", resp.StatusCode)
	}
}

// makeRequest is a helper for making HTTP requests
func makeRequest(t *testing.T, method, path string, body interface{}) (*http.Response, []byte) {
	t.Helper()

	var bodyReader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}
		bodyReader = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequest(method, baseURL+path, bodyReader)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	respBody, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	return resp, respBody
}

// assertStatus checks that the response has the expected status code
func assertStatus(t *testing.T, resp *http.Response, expected int, body []byte) {
	t.Helper()
	if resp.StatusCode != expected {
		t.Errorf("Expected status %d, got %d. Body: %s", expected, resp.StatusCode, string(body))
	}
}

// parseJSON unmarshals JSON response into a map
func parseJSON(t *testing.T, body []byte) map[string]interface{} {
	t.Helper()
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to parse JSON: %v. Body: %s", err, string(body))
	}
	return result
}

// parseJSONArray unmarshals JSON array response
func parseJSONArray(t *testing.T, body []byte) []interface{} {
	t.Helper()
	var result []interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to parse JSON array: %v. Body: %s", err, string(body))
	}
	return result
}

// getString gets a string value from a map
func getString(t *testing.T, m map[string]interface{}, key string) string {
	t.Helper()
	v, ok := m[key]
	if !ok {
		t.Fatalf("Key '%s' not found in map", key)
	}
	s, ok := v.(string)
	if !ok {
		t.Fatalf("Value for key '%s' is not a string: %T", key, v)
	}
	return s
}

// getFloat gets a float64 value from a map
func getFloat(t *testing.T, m map[string]interface{}, key string) float64 {
	t.Helper()
	v, ok := m[key]
	if !ok {
		t.Fatalf("Key '%s' not found in map", key)
	}
	f, ok := v.(float64)
	if !ok {
		t.Fatalf("Value for key '%s' is not a number: %T", key, v)
	}
	return f
}

// getArray gets an array value from a map
func getArray(t *testing.T, m map[string]interface{}, key string) []interface{} {
	t.Helper()
	v, ok := m[key]
	if !ok {
		t.Fatalf("Key '%s' not found in map", key)
	}
	arr, ok := v.([]interface{})
	if !ok {
		t.Fatalf("Value for key '%s' is not an array: %T", key, v)
	}
	return arr
}

// generateTestID generates a unique test ID
func generateTestID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}
