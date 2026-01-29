package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/janhq/jan-server/packages/go-common/testhelpers"
	"github.com/spf13/cobra"
)

var apiTestCmd = &cobra.Command{
	Use:   "api-test",
	Short: "Run API tests from Postman collections",
	Long: `Run API integration tests using Postman collection JSON files.

This is a lightweight cli api test that supports the essential
features needed for Jan Server testing: running collections, setting 
environment variables, and reporting results.

Examples:
  jan-cli api-test run tests/automation/auth-postman-scripts.json
  jan-cli api-test run tests/automation/auth-postman-scripts.json \
    --env-var "kong_url=http://localhost:8000" \
    --env-var "keycloak_admin=admin" \
    --verbose`,
}

var runApiTestCmd = &cobra.Command{
	Use:   "run [collection-file...]",
	Short: "Run a Postman collection",
	Long:  `Execute all requests in one or more Postman collection files and report results.`,
	Args:  cobra.MinimumNArgs(1),
	RunE:  runApiTest,
}

var (
	envVars    []string
	verbose    bool
	debug      bool
	reporters  []string
	timeout    int
	autoAuth   string
	autoModels bool
	envFile    string
	folder     string
	bail       bool
)

func init() {
	apiTestCmd.AddCommand(runApiTestCmd)

	runApiTestCmd.Flags().StringArrayVar(&envVars, "env-var", []string{}, "Environment variable (key=value)")
	runApiTestCmd.Flags().BoolVar(&verbose, "verbose", false, "Verbose output")
	runApiTestCmd.Flags().BoolVar(&debug, "debug", false, "Debug mode - print full request and response details")
	runApiTestCmd.Flags().StringArrayVar(&reporters, "reporters", []string{"cli"}, "Reporters to use")
	runApiTestCmd.Flags().IntVar(&timeout, "timeout-request", 30000, "Request timeout in milliseconds")
	runApiTestCmd.Flags().StringVar(&autoAuth, "auto-auth", "", "Auto-login: 'guest' or 'admin'")
	runApiTestCmd.Flags().BoolVar(&autoModels, "auto-models", false, "Auto-fetch model IDs before running")
	runApiTestCmd.Flags().StringVar(&envFile, "env-file", "", "Load environment variables from file")
	runApiTestCmd.Flags().StringVar(&folder, "folder", "", "Run only requests in this folder")
	runApiTestCmd.Flags().BoolVar(&bail, "bail", false, "Stop on first failure")
}

type PostmanCollection struct {
	Info struct {
		Name   string `json:"name"`
		Schema string `json:"schema"`
	} `json:"info"`
	Item  []PostmanItem  `json:"item"`
	Event []PostmanEvent `json:"event,omitempty"`
	Auth  *PostmanAuth   `json:"auth,omitempty"`
}

type PostmanItem struct {
	Name     string          `json:"name"`
	Request  *PostmanRequest `json:"request,omitempty"`
	Item     []PostmanItem   `json:"item,omitempty"`
	Event    []PostmanEvent  `json:"event,omitempty"`
	Auth     *PostmanAuth    `json:"auth,omitempty"`
	Disabled bool            `json:"disabled,omitempty"`
}

type PostmanRequest struct {
	Method string          `json:"method"`
	Header []PostmanHeader `json:"header"`
	Body   *PostmanBody    `json:"body,omitempty"`
	URL    interface{}     `json:"url"`
	Auth   *PostmanAuth    `json:"auth,omitempty"`
}

type PostmanHeader struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type PostmanAuth struct {
	Type   string                   `json:"type"`
	Bearer []map[string]interface{} `json:"bearer,omitempty"`
}

type PostmanBody struct {
	Mode       string            `json:"mode"`
	Raw        string            `json:"raw,omitempty"`
	Urlencoded []PostmanFormData `json:"urlencoded,omitempty"`
}

type PostmanFormData struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type PostmanEvent struct {
	Listen string        `json:"listen"`
	Script PostmanScript `json:"script"`
}

type PostmanScript struct {
	Type string   `json:"type"`
	Exec []string `json:"exec"`
}

type TestResult struct {
	Name     string
	Passed   bool
	Duration time.Duration
	Error    string
}

func runApiTest(cmd *cobra.Command, args []string) error {
	baseEnv := make(map[string]string)
	for _, ev := range envVars {
		parts := strings.SplitN(ev, "=", 2)
		if len(parts) == 2 {
			baseEnv[parts[0]] = parts[1]
		}
	}

	if envFile != "" {
		if err := loadEnvFile(envFile, baseEnv); err != nil {
			return fmt.Errorf("failed to load env file: %w", err)
		}
	}

	for _, collectionFile := range args {
		envMap := cloneEnvMap(baseEnv)
		if err := runCollection(collectionFile, envMap); err != nil {
			return err
		}
	}

	return nil
}

func runCollection(collectionFile string, envMap map[string]string) error {
	data, err := os.ReadFile(collectionFile)
	if err != nil {
		return fmt.Errorf("failed to read collection file: %w", err)
	}

	var collection PostmanCollection
	if err := json.Unmarshal(data, &collection); err != nil {
		return fmt.Errorf("failed to parse collection: %w", err)
	}

	fmt.Printf("\n==============================\n")
	fmt.Printf(" Jan API Test Runner\n")
	fmt.Printf("==============================\n\n")
	fmt.Printf("Collection: %s\n\n", collection.Info.Name)

	processCollectionEvents(collection.Event, envMap)

	if autoAuth != "" {
		gatewayURL := envMap["gateway_url"]
		if gatewayURL == "" {
			gatewayURL = envMap["kong_url"]
		}
		if gatewayURL == "" {
			gatewayURL = "http://localhost:8000"
		}

		if autoAuth == "admin" || autoAuth == "both" {
			kcURL := firstNonEmpty(envMap["keycloak_base_url"], "http://localhost:8085")
			kcUser := firstNonEmpty(envMap["keycloak_admin"], os.Getenv("KEYCLOAK_ADMIN"))
			kcPass := firstNonEmpty(envMap["keycloak_admin_password"], os.Getenv("KEYCLOAK_ADMIN_PASSWORD"))
			if kcUser != "" && kcPass != "" {
				if token, err := testhelpers.AdminLogin(kcURL, kcUser, kcPass); err == nil && token != "" {
					envMap["kc_admin_access_token"] = token
					envMap["access_token"] = token
				}
			}
		}

		if token, err := testhelpers.GuestLogin(gatewayURL); err == nil && token != "" {
			envMap["guest_access_token"] = token
			if envMap["access_token"] == "" {
				envMap["access_token"] = token
			}
		} else {
			fallbackURL := firstNonEmpty(envMap["llm_api_url"], envMap["llm_api_base_url"])
			if fallbackURL != "" && fallbackURL != gatewayURL {
				if token, err := testhelpers.GuestLogin(fallbackURL); err == nil && token != "" {
					envMap["guest_access_token"] = token
					if envMap["access_token"] == "" {
						envMap["access_token"] = token
					}
				}
			}
		}
	}

	ensureDefaultTokens(envMap)

	if autoModels && envMap["access_token"] != "" {
		gatewayURL := firstNonEmpty(envMap["gateway_url"], envMap["kong_url"], "http://localhost:8000")
		if model, err := testhelpers.GetDefaultModel(gatewayURL, envMap["access_token"]); err == nil && model != "" {
			envMap["model_id"] = model
			envMap["model_id_encoded"] = testhelpers.GetModelEncoded(model)
			envMap["default_model_id"] = model
		}
	}

	results := []TestResult{}
	totalStart := time.Now()

	for _, item := range collection.Item {
		itemResults := runItem(item, envMap, "", collection.Auth)
		results = append(results, itemResults...)
	}

	totalDuration := time.Since(totalStart)

	printResults(results, totalDuration)

	for _, result := range results {
		if !result.Passed {
			return fmt.Errorf("tests failed")
		}
	}

	return nil
}

func runItem(item PostmanItem, envMap map[string]string, prefix string, parentAuth *PostmanAuth) []TestResult {
	results := []TestResult{}

	if item.Disabled {
		if verbose {
			fmt.Printf("%sSkipping %s (disabled)\n", prefix, item.Name)
		}
		return results
	}

	// If this item has nested items (folder), run them
	if len(item.Item) > 0 {
		if folder != "" && item.Name != folder && prefix == "" {
			if verbose {
				fmt.Printf("Skipping folder %s (--folder=%s)\n", item.Name, folder)
			}
			return results
		}

		if verbose {
			fmt.Printf("\n[folder] %s\n", item.Name)
		}
		effectiveAuth := parentAuth
		if item.Auth != nil {
			effectiveAuth = item.Auth
		}
		for _, subItem := range item.Item {
			subResults := runItem(subItem, envMap, prefix+"  ", effectiveAuth)
			results = append(results, subResults...)
		}
		return results
	}

	// This is a request item
	if item.Request == nil {
		return results
	}

	result := TestResult{
		Name:   item.Name,
		Passed: true,
	}

	if envMap["access_token"] == "" {
		envMap["access_token"] = firstNonEmpty(envMap["kc_admin_access_token"], envMap["guest_access_token"], envMap["llm_api_token"])
	}

	processPreRequestScripts(item, envMap)
	if envMap["model_id_encoded"] == "" && envMap["model_id"] != "" {
		envMap["model_id_encoded"] = url.QueryEscape(envMap["model_id"])
	}

	start := time.Now()

	// Build URL
	urlStr := buildURL(item.Request.URL, envMap)

	if verbose {
		fmt.Printf("%s→ %s %s\n", prefix, item.Request.Method, urlStr)
	}

	// Create request
	var bodyReader io.Reader
	var bodyContent string
	if item.Request.Body != nil {
		if item.Request.Body.Mode == "raw" {
			body := replaceVariables(item.Request.Body.Raw, envMap)
			bodyContent = body
			bodyReader = strings.NewReader(body)
		} else if item.Request.Body.Mode == "urlencoded" {
			formData := url.Values{}
			for _, param := range item.Request.Body.Urlencoded {
				key := replaceVariables(param.Key, envMap)
				value := replaceVariables(param.Value, envMap)
				formData.Set(key, value)
			}
			bodyContent = formData.Encode()
			bodyReader = strings.NewReader(bodyContent)
		}
	}

	req, err := http.NewRequest(item.Request.Method, urlStr, bodyReader)
	if err != nil {
		result.Passed = false
		result.Error = fmt.Sprintf("Failed to create request: %v", err)
		result.Duration = time.Since(start)
		results = append(results, result)
		return results
	}

	// Set headers
	for _, header := range item.Request.Header {
		value := replaceVariables(header.Value, envMap)
		req.Header.Set(header.Key, value)
	}

	effectiveAuth := item.Request.Auth
	if effectiveAuth == nil {
		if item.Auth != nil {
			effectiveAuth = item.Auth
		} else {
			effectiveAuth = parentAuth
		}
	}

	if effectiveAuth != nil && strings.EqualFold(effectiveAuth.Type, "bearer") {
		for _, entry := range effectiveAuth.Bearer {
			if token, ok := entry["value"].(string); ok && token != "" {
				req.Header.Set("Authorization", "Bearer "+replaceVariables(token, envMap))
				break
			}
			if token, ok := entry["token"].(string); ok && token != "" {
				req.Header.Set("Authorization", "Bearer "+replaceVariables(token, envMap))
				break
			}
		}
	}

	// Debug: Print full request
	if debug {
		fmt.Printf("\n%s━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n", prefix)
		fmt.Printf("%s🔍 REQUEST DEBUG: %s\n", prefix, item.Name)
		fmt.Printf("%s━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n", prefix)
		fmt.Printf("%s%s %s\n", prefix, item.Request.Method, urlStr)
		fmt.Printf("%s\n%sHeaders:\n", prefix, prefix)
		for key, values := range req.Header {
			for _, value := range values {
				fmt.Printf("%s  %s: %s\n", prefix, key, value)
			}
		}
		if bodyContent != "" {
			fmt.Printf("%s\n%sBody:\n%s%s\n", prefix, prefix, prefix, bodyContent)
		}
		fmt.Printf("%s━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n", prefix)
	}

	// Execute request
	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Millisecond,
	}

	resp, err := client.Do(req)
	if err != nil {
		result.Passed = false
		result.Error = fmt.Sprintf("Request failed: %v", err)
		result.Duration = time.Since(start)
		results = append(results, result)
		return results
	}
	defer resp.Body.Close()

	// Read response
	respBody, _ := io.ReadAll(resp.Body)

	result.Duration = time.Since(start)

	// Debug: Print full response
	if debug {
		fmt.Printf("\n%s━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n", prefix)
		fmt.Printf("%s🔍 RESPONSE DEBUG: %s\n", prefix, item.Name)
		fmt.Printf("%s━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n", prefix)
		fmt.Printf("%sStatus: %d %s\n", prefix, resp.StatusCode, http.StatusText(resp.StatusCode))
		fmt.Printf("%sDuration: %dms\n", prefix, result.Duration.Milliseconds())
		fmt.Printf("%s\n%sHeaders:\n", prefix, prefix)
		for key, values := range resp.Header {
			for _, value := range values {
				fmt.Printf("%s  %s: %s\n", prefix, key, value)
			}
		}
		if len(respBody) > 0 {
			fmt.Printf("%s\n%sBody:\n%s%s\n", prefix, prefix, prefix, string(respBody))
		}
		fmt.Printf("%s━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n", prefix)
	}

	if verbose {
		fmt.Printf("%s  ← %d %s (%dms)\n", prefix, resp.StatusCode, http.StatusText(resp.StatusCode), result.Duration.Milliseconds())
	}

	allowedStatusCodes := getExpectedStatusCodes(item)
	statusAllowed := false
	if len(allowedStatusCodes) == 0 {
		statusAllowed = resp.StatusCode < 400
	} else {
		statusAllowed = intSliceContains(allowedStatusCodes, resp.StatusCode)
		if !statusAllowed && resp.StatusCode == http.StatusCreated && intSliceContains(allowedStatusCodes, http.StatusOK) {
			statusAllowed = true
		}
	}

	if !statusAllowed {
		result.Passed = false
		result.Error = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(respBody))
	} else {
		if jsonRPCFailed, jsonRPCErr := checkJSONRPCFailure(respBody); jsonRPCFailed {
			result.Passed = false
			result.Error = fmt.Sprintf("JSON-RPC error: %s", jsonRPCErr)
		} else {
			// Extract variables from test scripts if the response matched expectations
			extractVariablesFromScripts(item, respBody, resp, envMap)
		}
	}

	if bail && !result.Passed {
		fmt.Printf("\nBail: stopping after first failure\n")
		os.Exit(1)
	}

	results = append(results, result)
	return results
}

func buildURL(urlInterface interface{}, envMap map[string]string) string {
	switch v := urlInterface.(type) {
	case string:
		return replaceVariables(v, envMap)
	case map[string]interface{}:
		// Handle Postman URL object format
		if raw, ok := v["raw"].(string); ok {
			url := replaceVariables(raw, envMap)

			// Handle path variables (e.g., :id, :public_id)
			if variables, ok := v["variable"].([]interface{}); ok {
				for _, varInterface := range variables {
					if varMap, ok := varInterface.(map[string]interface{}); ok {
						if key, ok := varMap["key"].(string); ok {
							if value, ok := varMap["value"].(string); ok {
								// Replace :key with actual value
								replacedValue := replaceVariables(value, envMap)
								url = strings.ReplaceAll(url, ":"+key, replacedValue)
							}
						}
					}
				}
			}

			return url
		}
		// Also try "url" field
		if urlStr, ok := v["url"].(string); ok {
			return replaceVariables(urlStr, envMap)
		}
	}
	return fmt.Sprintf("%v", urlInterface)
}

func replaceVariables(text string, envMap map[string]string) string {
	result := text
	for key, value := range envMap {
		result = strings.ReplaceAll(result, "{{"+key+"}}", value)
		result = strings.ReplaceAll(result, "${"+key+"}", value)
	}
	return result
}

func checkJSONRPCFailure(respBody []byte) (bool, string) {
	var payload map[string]interface{}
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return false, ""
	}

	if _, ok := payload["jsonrpc"]; !ok {
		return false, ""
	}

	if errObj, ok := payload["error"].(map[string]interface{}); ok {
		message := ""
		if msg, ok := errObj["message"].(string); ok && msg != "" {
			message = msg
		}
		if code, ok := errObj["code"].(float64); ok {
			if message != "" {
				message = fmt.Sprintf("code %.0f: %s", code, message)
			} else {
				message = fmt.Sprintf("code %.0f", code)
			}
		}
		if message == "" {
			message = "error response"
		}
		return true, message
	}

	if resultObj, ok := payload["result"].(map[string]interface{}); ok {
		if isError, ok := resultObj["isError"].(bool); ok && isError {
			return true, "result.isError=true"
		}
	}

	return false, ""
}

// processCollectionEvents processes collection-level prerequest scripts to initialize variables
func processCollectionEvents(events []PostmanEvent, envMap map[string]string) {
	for _, event := range events {
		if event.Listen != "prerequest" {
			continue
		}

		script := strings.Join(event.Script.Exec, "\n")
		lines := strings.Split(script, "\n")

		for _, line := range lines {
			line = strings.TrimSpace(line)

			// Handle test_user_username
			if strings.Contains(line, "pm.collectionVariables.set('test_user_username'") {
				if _, exists := envMap["test_user_username"]; !exists {
					envMap["test_user_username"] = fmt.Sprintf("automation-user-%d", time.Now().UnixNano())
				}
			}

			// Handle test_user_password
			if strings.Contains(line, "pm.collectionVariables.set('test_user_password'") {
				if _, exists := envMap["test_user_password"]; !exists {
					envMap["test_user_password"] = fmt.Sprintf("Passw0rd!%d", time.Now().UnixNano()%10000)
				}
			}

			// Handle test_user_email
			if strings.Contains(line, "pm.collectionVariables.set('test_user_email'") {
				if _, exists := envMap["test_user_email"]; !exists {
					if username, ok := envMap["test_user_username"]; ok {
						envMap["test_user_email"] = username + "@example.com"
					}
				}
			}

			// Handle test_user_pid
			if strings.Contains(line, "pm.collectionVariables.set('test_user_pid'") {
				if _, exists := envMap["test_user_pid"]; !exists {
					if username, ok := envMap["test_user_username"]; ok {
						envMap["test_user_pid"] = username
					}
				}
			}

			// Handle collection_timestamp
			if strings.Contains(line, "pm.collectionVariables.set('collection_timestamp'") {
				if _, exists := envMap["collection_timestamp"]; !exists {
					envMap["collection_timestamp"] = time.Now().Format(time.RFC3339)
				}
			}
		}
	}
}

// extractVariablesFromScripts parses test scripts and extracts variables
func extractVariablesFromScripts(item PostmanItem, respBody []byte, resp *http.Response, envMap map[string]string) {
	if len(item.Event) == 0 {
		return
	}

	// Parse response body as JSON if possible
	var responseData map[string]interface{}
	json.Unmarshal(respBody, &responseData) // Ignore error - not all responses are JSON

	// Process each event script
	for _, event := range item.Event {
		if event.Listen != "test" {
			continue
		}

		// Join script lines
		script := strings.Join(event.Script.Exec, "\n")

		// Check for Location header extraction in the script
		locationExtracted := false
		if strings.Contains(script, "pm.response.headers.get('Location')") &&
			strings.Contains(script, "pm.collectionVariables.set('test_user_id'") {
			// Extract user ID from Location header
			if resp.StatusCode == 201 || resp.StatusCode == 204 {
				location := resp.Header.Get("Location")
				if location != "" {
					// Extract ID from location (last segment of path)
					lastSlash := strings.LastIndex(location, "/")
					if lastSlash >= 0 && lastSlash < len(location)-1 {
						userID := location[lastSlash+1:]
						envMap["test_user_id"] = userID
						envMap["teardown_user_id"] = userID
						locationExtracted = true
					}
				}
			}
		}

		// Simple pattern matching for pm.collectionVariables.set calls
		// Pattern: pm.collectionVariables.set('varname', data.field)
		lines := strings.Split(script, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)

			// Look for pm.collectionVariables.set
			if strings.Contains(line, "pm.collectionVariables.set") ||
				strings.Contains(line, "pm.environment.set") ||
				strings.Contains(line, "pm.variables.set") {
				// Extract variable name and source field
				// Example: pm.collectionVariables.set('kc_admin_access_token', data.access_token);
				varName, jsonPath := extractVarSetPattern(line)

				// Skip if we already extracted this variable from Location header
				if varName == "test_user_id" && locationExtracted {
					continue
				}

				if varName != "" && jsonPath != "" {
					// Handle encodeURIComponent(data.field) patterns
					if strings.HasPrefix(jsonPath, "encodeURIComponent(") && strings.HasSuffix(jsonPath, ")") {
						innerPath := strings.TrimPrefix(jsonPath, "encodeURIComponent(")
						innerPath = strings.TrimSuffix(innerPath, ")")
						innerPath = cleanJSONPath(innerPath)
						if value := extractJSONValueWithFallback(responseData, innerPath); value != "" {
							encoded := url.QueryEscape(value)
							if encoded != "" && encoded != "<nil>" {
								envMap[varName] = encoded
							}
						}
						continue
					}

					// Extract value from response data
					if value := extractJSONValueWithFallback(responseData, jsonPath); value != "" && value != "<nil>" {
						envMap[varName] = value
					}
				}
			}
		}
	}

	applyDefaultExtractions(responseData, envMap)
}

// extractVarSetPattern extracts variable name and JSON path from pm.collectionVariables.set line
func extractVarSetPattern(line string) (varName string, jsonPath string) {
	// Remove semicolons and clean up
	line = strings.TrimSuffix(line, ";")
	line = strings.TrimSpace(line)

	prefixes := []string{
		"pm.collectionVariables.set(",
		"pm.environment.set(",
		"pm.variables.set(",
	}

	for _, prefix := range prefixes {
		if idx := strings.Index(line, prefix); idx >= 0 {
			argsStart := idx + len(prefix)
			argsEnd := strings.LastIndex(line, ")")
			if argsEnd > argsStart {
				args := line[argsStart:argsEnd]
				parts := strings.SplitN(args, ",", 2)
				if len(parts) == 2 {
					varName = strings.Trim(strings.TrimSpace(parts[0]), "'\"")
					jsonPath = strings.TrimSpace(parts[1])
					jsonPath = cleanJSONPath(jsonPath)
					break
				}
			}
		}
	}
	return
}

// extractJSONValue extracts a value from JSON response using dot notation
func extractJSONValue(data map[string]interface{}, path string) string {
	parts := strings.Split(path, ".")
	var current interface{} = data

	for _, part := range parts {
		key := part
		index := -1
		if bracket := strings.Index(part, "["); bracket >= 0 && strings.HasSuffix(part, "]") {
			key = part[:bracket]
			if idx, err := strconv.Atoi(part[bracket+1 : len(part)-1]); err == nil {
				index = idx
			}
		}

		if key != "" {
			if m, ok := current.(map[string]interface{}); ok {
				current = m[key]
			} else {
				return ""
			}
		}

		if index >= 0 {
			arr, ok := current.([]interface{})
			if !ok || index >= len(arr) || index < 0 {
				return ""
			}
			current = arr[index]
		}
	}

	// Convert to string
	switch v := current.(type) {
	case string:
		return v
	case float64:
		return fmt.Sprintf("%.0f", v)
	case bool:
		return fmt.Sprintf("%t", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func extractJSONValueWithFallback(data map[string]interface{}, path string) string {
	path = cleanJSONPath(path)
	if path == "" {
		return ""
	}

	if value := extractJSONValue(data, path); value != "" && value != "<nil>" {
		return value
	}

	for strings.Contains(path, ".") {
		if idx := strings.Index(path, "."); idx >= 0 {
			path = path[idx+1:]
		} else {
			break
		}
		if path == "" {
			break
		}
		if value := extractJSONValue(data, path); value != "" && value != "<nil>" {
			return value
		}
	}

	return ""
}

func cleanJSONPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, "data.")
	path = strings.TrimPrefix(path, "body.")
	path = strings.TrimPrefix(path, "responseData.")
	path = strings.TrimPrefix(path, "response.")
	path = strings.TrimPrefix(path, "pm.response.")
	path = strings.TrimPrefix(path, "pm.response.json().")
	path = strings.TrimPrefix(path, "pm.response.json().")
	path = strings.TrimPrefix(path, "responseJson.")
	path = strings.TrimPrefix(path, "payload.")
	path = strings.TrimPrefix(path, "guestData.")
	if strings.Contains(path, "||") {
		parts := strings.Split(path, "||")
		path = strings.TrimSpace(parts[0])
	}
	path = strings.Trim(path, "'\"")
	path = strings.Trim(path, "()")
	return path
}

func applyDefaultExtractions(responseData map[string]interface{}, envMap map[string]string) {
	if token, ok := responseData["access_token"].(string); ok && token != "" {
		if envMap["guest_access_token"] == "" {
			envMap["guest_access_token"] = token
		}
		if envMap["kc_admin_access_token"] == "" {
			envMap["kc_admin_access_token"] = token
		}
		if envMap["llm_api_token"] == "" {
			envMap["llm_api_token"] = token
		}
		if envMap["access_token"] == "" {
			envMap["access_token"] = token
		}
	}

	if envMap["model_id"] == "" {
		if dataArr, ok := responseData["data"].([]interface{}); ok && len(dataArr) > 0 {
			if first, ok := dataArr[0].(map[string]interface{}); ok {
				if id, ok := first["id"].(string); ok && id != "" {
					envMap["model_id"] = id
					envMap["model_id_encoded"] = url.QueryEscape(id)
					if envMap["default_model_id"] == "" {
						envMap["default_model_id"] = id
					}
				}
			}
		}
	}

	if id, ok := responseData["id"].(string); ok && strings.HasPrefix(id, "conv_") {
		if envMap["conversation_id"] == "" {
			envMap["conversation_id"] = id
		}
		if envMap["conversationId1"] == "" {
			envMap["conversationId1"] = id
		}
	}

	if conv, ok := responseData["conversation"].(map[string]interface{}); ok {
		if cid, ok := conv["id"].(string); ok && cid != "" {
			if envMap["conversation_id"] == "" {
				envMap["conversation_id"] = cid
			}
			if envMap["conversationId1"] == "" {
				envMap["conversationId1"] = cid
			}
		}
		if title, ok := conv["title"].(string); ok && title != "" && envMap["conversation_title"] == "" {
			envMap["conversation_title"] = title
		}
	}

	if id, ok := responseData["id"].(string); ok && strings.HasPrefix(id, "proj_") {
		if envMap["project_id_1"] == "" {
			envMap["project_id_1"] = id
		}
	}

	if envMap["default_model_id"] == "" && envMap["model_id"] != "" {
		envMap["default_model_id"] = envMap["model_id"]
	}

	if id, ok := responseData["id"].(string); ok && strings.HasPrefix(id, "resp_") {
		if envMap["response_id"] == "" {
			envMap["response_id"] = id
		}
		if envMap["background_response_id"] == "" {
			envMap["background_response_id"] = id
		}
		if envMap["long_task_response_id"] == "" {
			envMap["long_task_response_id"] = id
		}
	}

	if envMap["provider_id"] == "" {
		if id, ok := responseData["provider_id"].(string); ok && id != "" {
			envMap["provider_id"] = id
		} else if id, ok := responseData["id"].(string); ok && strings.HasPrefix(id, "prov_") {
			envMap["provider_id"] = id
		}
		if envMap["provider_id"] == "" {
			if dataArr, ok := responseData["data"].([]interface{}); ok && len(dataArr) > 0 {
				if first, ok := dataArr[0].(map[string]interface{}); ok {
					if id, ok := first["id"].(string); ok && strings.HasPrefix(id, "prov_") {
						envMap["provider_id"] = id
					} else if pid, ok := first["provider_id"].(string); ok && pid != "" {
						envMap["provider_id"] = pid
					}
				}
			}
		}
	}

	if envMap["provider_public_id"] == "" {
		if pid, ok := responseData["public_id"].(string); ok && pid != "" {
			envMap["provider_public_id"] = pid
		}
		if envMap["provider_public_id"] == "" && envMap["provider_id"] != "" {
			envMap["provider_public_id"] = envMap["provider_id"]
		}
	}
}

func ensureDefaultTokens(envMap map[string]string) {
	ensureAdminToken(envMap)
	if envMap["llm_api_token"] != "" {
		return
	}

	if token := firstNonEmpty(envMap["guest_access_token"], envMap["kc_admin_access_token"], envMap["access_token"]); token != "" {
		envMap["llm_api_token"] = token
		if envMap["access_token"] == "" {
			envMap["access_token"] = token
		}
		return
	}

	kongURL := strings.TrimSpace(envMap["kong_url"])
	if kongURL == "" {
		kongURL = "http://localhost:8000"
	}
	loginURL := strings.TrimSuffix(kongURL, "/") + "/auth/guest-login"

	req, err := http.NewRequest(http.MethodPost, loginURL, strings.NewReader("{}"))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return
	}

	token, ok := payload["access_token"].(string)
	if !ok || token == "" {
		return
	}

	if envMap["guest_access_token"] == "" {
		envMap["guest_access_token"] = token
	}
	if envMap["kc_admin_access_token"] == "" {
		envMap["kc_admin_access_token"] = token
	}
	envMap["llm_api_token"] = token
	if envMap["access_token"] == "" {
		envMap["access_token"] = token
	}

	ensureAdminToken(envMap)
}

func ensureAdminToken(envMap map[string]string) {
	if envMap["kc_admin_access_token"] != "" {
		return
	}

	adminUser := firstNonEmpty(envMap["keycloak_admin"], envMap["KEYCLOAK_ADMIN"], os.Getenv("KEYCLOAK_ADMIN"))
	adminPass := firstNonEmpty(envMap["keycloak_admin_password"], envMap["KEYCLOAK_ADMIN_PASSWORD"], os.Getenv("KEYCLOAK_ADMIN_PASSWORD"))
	if adminUser == "" || adminPass == "" {
		return
	}

	kcURL := firstNonEmpty(envMap["keycloak_admin_url"], envMap["keycloak_base_url"], os.Getenv("KEYCLOAK_ADMIN_URL"), os.Getenv("KEYCLOAK_BASE_URL"))
	if kcURL == "" {
		kcURL = "http://localhost:8085"
	}
	tokenURL := strings.TrimSuffix(kcURL, "/") + "/realms/master/protocol/openid-connect/token"

	form := url.Values{}
	form.Set("grant_type", "password")
	form.Set("client_id", "admin-cli")
	form.Set("username", adminUser)
	form.Set("password", adminPass)

	req, err := http.NewRequest(http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return
	}
	token, _ := payload["access_token"].(string)
	if token == "" {
		return
	}
	envMap["kc_admin_access_token"] = token
	if envMap["access_token"] == "" {
		envMap["access_token"] = token
	}
}

func loadEnvFile(path string, envMap map[string]string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var jsonEnv map[string]string
	if err := json.Unmarshal(data, &jsonEnv); err == nil {
		for k, v := range jsonEnv {
			envMap[k] = v
		}
		return nil
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.Trim(strings.TrimSpace(parts[1]), "\"'")
			envMap[key] = value
		}
	}
	return nil
}

func cloneEnvMap(source map[string]string) map[string]string {
	clone := make(map[string]string, len(source))
	for k, v := range source {
		clone[k] = v
	}
	return clone
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func getExpectedStatusCodes(item PostmanItem) []int {
	statuses := []int{}
	statusRegexes := []*regexp.Regexp{
		regexp.MustCompile(`pm\.response\.to\.have\.status\((\d+)\)`),
		regexp.MustCompile(`pm\.expect\(\s*pm\.response\.code\s*\)\.to\.eql\((\d+)\)`),
	}

	for _, event := range item.Event {
		if event.Listen != "test" {
			continue
		}
		script := strings.Join(event.Script.Exec, "\n")

		for _, re := range statusRegexes {
			matches := re.FindAllStringSubmatch(script, -1)
			for _, match := range matches {
				if code, err := strconv.Atoi(match[1]); err == nil {
					statuses = append(statuses, code)
				}
			}
		}

		if strings.Contains(script, "to.include(status)") || strings.Contains(script, "to.include(pm.response.code)") {
			arrayRegex := regexp.MustCompile(`pm\.expect\(\s*\[([0-9,\s]+)\]\s*\)\.to\.include`)
			if match := arrayRegex.FindStringSubmatch(script); len(match) > 1 {
				statuses = append(statuses, parseStatusList(match[1])...)
			}
		}

		oneOfRegex := regexp.MustCompile(`pm\.expect\(\s*pm\.response\.code\s*\)\.to\.be\.oneOf\(\s*\[([0-9,\s]+)\]\s*\)`)
		if match := oneOfRegex.FindStringSubmatch(script); len(match) > 1 {
			statuses = append(statuses, parseStatusList(match[1])...)
		}
	}

	return statuses
}

func parseStatusList(raw string) []int {
	parts := strings.Split(raw, ",")
	statuses := make([]int, 0, len(parts))
	for _, part := range parts {
		if code, err := strconv.Atoi(strings.TrimSpace(part)); err == nil {
			statuses = append(statuses, code)
		}
	}
	return statuses
}

func intSliceContains(list []int, value int) bool {
	for _, v := range list {
		if v == value {
			return true
		}
	}
	return false
}

func processPreRequestScripts(item PostmanItem, envMap map[string]string) {
	for _, event := range item.Event {
		if event.Listen != "prerequest" {
			continue
		}

		script := strings.Join(event.Script.Exec, "\n")

		if strings.Contains(script, "upgrade_username") {
			upgradeUsername := envMap["guest_upgrade_username"]
			if upgradeUsername == "" {
				base := envMap["guest_username"]
				if base == "" {
					base = fmt.Sprintf("guest-%d", time.Now().UnixNano())
				}
				upgradeUsername = fmt.Sprintf("%s-upgraded", base)
				envMap["guest_upgrade_username"] = upgradeUsername
				envMap["guest_upgrade_email"] = fmt.Sprintf("%s@example.com", upgradeUsername)
			}

			envMap["upgrade_username"] = upgradeUsername
			if email := envMap["guest_upgrade_email"]; email != "" {
				envMap["upgrade_email"] = email
			} else {
				envMap["upgrade_email"] = fmt.Sprintf("%s@example.com", upgradeUsername)
			}
		}

		if strings.Contains(script, "teardown_user_id") {
			if userID := envMap["test_user_id"]; userID != "" {
				envMap["teardown_user_id"] = userID
			}
		}
	}
}

func printResults(results []TestResult, totalDuration time.Duration) {
	passed := 0
	failed := 0

	for _, result := range results {
		if result.Passed {
			passed++
		} else {
			failed++
		}
	}

	fmt.Printf("\n┌─────────────────────────────────────────────────────────────────────┐\n")
	fmt.Printf("│ Test Results                                                        │\n")
	fmt.Printf("└─────────────────────────────────────────────────────────────────────┘\n\n")

	// Print all test results with visual indicators
	for _, result := range results {
		if result.Passed {
			fmt.Printf("  ✓✓✓ %s (%dms)\n", result.Name, result.Duration.Milliseconds())
		} else {
			fmt.Printf("  ✗✗✗ %s (%dms)\n", result.Name, result.Duration.Milliseconds())
			if result.Error != "" {
				fmt.Printf("      %s\n", result.Error)
			}
		}
	}

	fmt.Printf("\n")
	fmt.Printf("Summary:\n")
	fmt.Printf("  Total:    %d tests\n", len(results))
	fmt.Printf("  Passed:   %d ✓✓✓\n", passed)
	if failed > 0 {
		fmt.Printf("  Failed:   %d ✗✗✗\n", failed)
	}
	fmt.Printf("  Duration: %dms\n\n", totalDuration.Milliseconds())

	if failed == 0 {
		fmt.Printf("✓✓✓ All tests passed!\n\n")
	} else {
		fmt.Printf("✗✗✗ Some tests failed\n\n")
	}
}
