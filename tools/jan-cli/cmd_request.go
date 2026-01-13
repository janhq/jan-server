package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// SimpleRequest represents a simple JSON request file format
type SimpleRequest struct {
	Name        string             `json:"name"`
	Description string             `json:"description,omitempty"`
	Auth        *SimpleAuth        `json:"auth,omitempty"`
	Request     *SimpleRequestSpec `json:"request,omitempty"` // Single request (optional if chain is used)
	Chain       []ChainedRequest   `json:"chain,omitempty"`   // Chained requests
	Variables   map[string]string  `json:"variables,omitempty"`
}

// ChainedRequest represents a single request in a chain
type ChainedRequest struct {
	Name    string            `json:"name"`
	Request SimpleRequestSpec `json:"request"`
	Extract map[string]string `json:"extract,omitempty"` // Extract values from response using JSON path
}

type SimpleAuth struct {
	Type     string `json:"type"` // "guest", "bearer", "api-key"
	Endpoint string `json:"endpoint,omitempty"`
	Token    string `json:"token,omitempty"`
}

type SimpleRequestSpec struct {
	Method   string            `json:"method"`
	Endpoint string            `json:"endpoint"`
	Headers  map[string]string `json:"headers,omitempty"`
	Body     interface{}       `json:"body,omitempty"`
}

var requestCmd = &cobra.Command{
	Use:   "request",
	Short: "Run API requests from JSON files",
	Long: `Run API requests from simple JSON request files.

This provides a simpler alternative to Postman collections for quick testing.

Examples:
  jan-cli request run tests/requests/response-api-test.json
  jan-cli request run tests/requests/response-api-test.json --base-url http://localhost:8000
  jan-cli request run tests/requests/response-api-test.json --debug`,
}

var runRequestCmd = &cobra.Command{
	Use:   "run [request-file]",
	Short: "Run a request from JSON file",
	Long:  `Execute a request defined in a JSON file, with optional auto-authentication.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runRequestFile,
}

var (
	reqBaseURL    string
	reqDebug      bool
	reqTimeout    int
	reqSkipAuth   bool
	reqSaveToken  string
	reqFullOutput bool
)

func init() {
	rootCmd.AddCommand(requestCmd)
	requestCmd.AddCommand(runRequestCmd)

	runRequestCmd.Flags().StringVar(&reqBaseURL, "base-url", "http://localhost:8000", "Base URL for API requests")
	runRequestCmd.Flags().BoolVar(&reqDebug, "debug", false, "Print full request and response details")
	runRequestCmd.Flags().BoolVar(&reqFullOutput, "full-output", false, "Parse and display all tool calls, steps, and output items")
	runRequestCmd.Flags().IntVar(&reqTimeout, "timeout", 120, "Request timeout in seconds")
	runRequestCmd.Flags().BoolVar(&reqSkipAuth, "skip-auth", false, "Skip auto-authentication")
	runRequestCmd.Flags().StringVar(&reqSaveToken, "save-token", "", "Save auth token to file for reuse")
}

func runRequestFile(cmd *cobra.Command, args []string) error {
	requestFile := args[0]

	data, err := os.ReadFile(requestFile)
	if err != nil {
		return fmt.Errorf("failed to read request file: %w", err)
	}

	var req SimpleRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return fmt.Errorf("failed to parse request file: %w", err)
	}

	fmt.Printf("\n==============================\n")
	fmt.Printf(" Jan Request Runner\n")
	fmt.Printf("==============================\n\n")
	fmt.Printf("Request: %s\n", req.Name)
	if req.Description != "" {
		fmt.Printf("Description: %s\n", req.Description)
	}
	fmt.Printf("Base URL: %s\n\n", reqBaseURL)

	// Initialize variables map for chain execution
	variables := make(map[string]string)
	if req.Variables != nil {
		for k, v := range req.Variables {
			variables[k] = v
		}
	}

	// Handle authentication
	var authToken string
	if !reqSkipAuth && req.Auth != nil {
		switch req.Auth.Type {
		case "guest":
			fmt.Printf("🔑 Authenticating as guest...\n")
			token, err := guestLogin(reqBaseURL, req.Auth.Endpoint)
			if err != nil {
				return fmt.Errorf("guest login failed: %w", err)
			}
			authToken = token
			fmt.Printf("✓ Guest login successful\n\n")

			// Save token if requested
			if reqSaveToken != "" {
				if err := os.WriteFile(reqSaveToken, []byte(token), 0600); err != nil {
					fmt.Printf("⚠ Failed to save token: %v\n", err)
				} else {
					fmt.Printf("✓ Token saved to %s\n\n", reqSaveToken)
				}
			}

		case "bearer":
			authToken = req.Auth.Token
		case "api-key":
			// Handle as X-API-Key header instead
		}
	}

	// Check if we have chained requests or single request
	if len(req.Chain) > 0 {
		return executeChainedRequests(req.Chain, authToken, variables)
	}

	// Single request mode
	if req.Request == nil {
		return fmt.Errorf("no request or chain defined in request file")
	}

	_, err = executeSingleRequest("Main Request", *req.Request, authToken, variables, nil)
	return err
}

// executeSingleRequest executes a single request and returns the response body
func executeSingleRequest(name string, reqSpec SimpleRequestSpec, authToken string, variables map[string]string, extract map[string]string) ([]byte, error) {
	// Substitute variables in endpoint
	endpoint := substituteVariables(reqSpec.Endpoint, variables)
	fullURL := reqBaseURL + endpoint

	// Prepare body with variable substitution
	var bodyReader io.Reader
	var bodyBytes []byte
	var err error
	if reqSpec.Body != nil {
		bodyBytes, err = json.Marshal(reqSpec.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		// Substitute variables in body
		bodyStr := substituteVariables(string(bodyBytes), variables)
		bodyBytes = []byte(bodyStr)
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// Create HTTP request
	method := reqSpec.Method
	if method == "" {
		method = "POST"
	}

	httpReq, err := http.NewRequest(method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers with variable substitution
	for key, value := range reqSpec.Headers {
		httpReq.Header.Set(key, substituteVariables(value, variables))
	}

	// Set auth header
	if authToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+authToken)
	}

	// Debug output
	if reqDebug {
		fmt.Printf("📤 Request:\n")
		fmt.Printf("   %s %s\n", method, fullURL)
		fmt.Printf("   Headers:\n")
		for key, values := range httpReq.Header {
			for _, v := range values {
				if key == "Authorization" {
					if len(v) > 50 {
						v = v[:50] + "..."
					}
				}
				fmt.Printf("      %s: %s\n", key, v)
			}
		}
		if bodyBytes != nil {
			var prettyBody bytes.Buffer
			if json.Indent(&prettyBody, bodyBytes, "      ", "  ") == nil {
				fmt.Printf("   Body:\n%s\n", indentString(prettyBody.String(), "      "))
			} else {
				fmt.Printf("   Body:\n%s\n", indentString(string(bodyBytes), "      "))
			}
		}
		fmt.Println()
	}

	// Execute request
	fmt.Printf("🚀 Sending %s request to %s...\n", method, endpoint)
	startTime := time.Now()

	client := &http.Client{
		Timeout: time.Duration(reqTimeout) * time.Second,
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	duration := time.Since(startTime)

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Print response
	fmt.Printf("\n📥 Response (took %v):\n", duration.Round(time.Millisecond))
	fmt.Printf("   Status: %s\n", resp.Status)

	if reqDebug {
		fmt.Printf("   Headers:\n")
		for key, values := range resp.Header {
			for _, v := range values {
				fmt.Printf("      %s: %s\n", key, v)
			}
		}
	}

	// Pretty print JSON response
	var prettyResp bytes.Buffer
	if err := json.Indent(&prettyResp, respBody, "   ", "  "); err == nil {
		fmt.Printf("   Body:\n   %s\n", prettyResp.String())
	} else {
		fmt.Printf("   Body: %s\n", string(respBody))
	}

	// Full output mode - parse and display tool calls, steps, and output items
	if reqFullOutput && resp.StatusCode >= 200 && resp.StatusCode < 300 {
		printFullOutput(respBody)
	}

	// Extract variables from response
	if len(extract) > 0 && resp.StatusCode >= 200 && resp.StatusCode < 300 {
		extractVariables(respBody, extract, variables)
	}

	// Summary
	fmt.Println()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		fmt.Printf("✅ Request completed successfully\n")
	} else {
		fmt.Printf("❌ Request failed with status %d\n", resp.StatusCode)
		return respBody, fmt.Errorf("request failed with status %d", resp.StatusCode)
	}

	return respBody, nil
}

// executeChainedRequests executes multiple requests in sequence
func executeChainedRequests(chain []ChainedRequest, authToken string, variables map[string]string) error {
	fmt.Printf("🔗 Executing chain of %d requests\n\n", len(chain))

	for i, chainReq := range chain {
		fmt.Printf("\n" + strings.Repeat("─", 60) + "\n")
		fmt.Printf("📍 Step %d/%d: %s\n", i+1, len(chain), chainReq.Name)
		fmt.Printf(strings.Repeat("─", 60) + "\n")

		_, err := executeSingleRequest(chainReq.Name, chainReq.Request, authToken, variables, chainReq.Extract)
		if err != nil {
			return fmt.Errorf("chain step %d (%s) failed: %w", i+1, chainReq.Name, err)
		}

		// Print extracted variables
		if len(chainReq.Extract) > 0 && reqDebug {
			fmt.Printf("\n📦 Extracted variables:\n")
			for varName := range chainReq.Extract {
				if val, ok := variables[varName]; ok {
					displayVal := val
					if len(displayVal) > 100 {
						displayVal = displayVal[:100] + "..."
					}
					fmt.Printf("   %s = %s\n", varName, displayVal)
				}
			}
		}
	}

	fmt.Printf("\n" + strings.Repeat("═", 60) + "\n")
	fmt.Printf("✅ All %d chain requests completed successfully\n", len(chain))
	fmt.Printf(strings.Repeat("═", 60) + "\n")

	return nil
}

// substituteVariables replaces {{varName}} with variable values
func substituteVariables(input string, variables map[string]string) string {
	result := input
	for key, value := range variables {
		placeholder := "{{" + key + "}}"
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

// extractVariables extracts values from JSON response using simple path notation
func extractVariables(respBody []byte, extract map[string]string, variables map[string]string) {
	var respData map[string]interface{}
	if err := json.Unmarshal(respBody, &respData); err != nil {
		return
	}

	for varName, jsonPath := range extract {
		value := extractJSONPath(respData, jsonPath)
		if value != "" {
			variables[varName] = value
		}
	}
}

// extractJSONPath extracts a value from JSON using simple dot notation (e.g., "id", "data.items")
func extractJSONPath(data map[string]interface{}, path string) string {
	parts := strings.Split(path, ".")
	var current interface{} = data

	for _, part := range parts {
		switch v := current.(type) {
		case map[string]interface{}:
			current = v[part]
		case []interface{}:
			// Handle array index like "items.0"
			var idx int
			if _, err := fmt.Sscanf(part, "%d", &idx); err == nil && idx < len(v) {
				current = v[idx]
			} else {
				return ""
			}
		default:
			return ""
		}
		if current == nil {
			return ""
		}
	}

	// Convert to string
	switch v := current.(type) {
	case string:
		return v
	case float64:
		return fmt.Sprintf("%v", v)
	case bool:
		return fmt.Sprintf("%v", v)
	default:
		// For objects/arrays, marshal to JSON
		if b, err := json.Marshal(v); err == nil {
			return string(b)
		}
		return ""
	}
}

// ResponseOutput represents the parsed response structure
type ResponseOutput struct {
	ID             string                   `json:"id"`
	Status         string                   `json:"status"`
	Model          string                   `json:"model"`
	Input          string                   `json:"input"`
	Output         []map[string]interface{} `json:"output"`
	Usage          map[string]interface{}   `json:"usage"`
	ConversationID string                   `json:"conversation_id"`
	Error          map[string]interface{}   `json:"error"`
}

func printFullOutput(respBody []byte) {
	var resp ResponseOutput
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return
	}

	fmt.Printf("\n" + strings.Repeat("=", 60) + "\n")
	fmt.Printf("📋 FULL OUTPUT BREAKDOWN\n")
	fmt.Printf(strings.Repeat("=", 60) + "\n")

	fmt.Printf("\n🆔 Response ID: %s\n", resp.ID)
	fmt.Printf("📊 Status: %s\n", resp.Status)
	fmt.Printf("🤖 Model: %s\n", resp.Model)
	if resp.ConversationID != "" {
		fmt.Printf("💬 Conversation: %s\n", resp.ConversationID)
	}

	if len(resp.Output) == 0 {
		fmt.Printf("\n⚠️  No output items in response\n")
		return
	}

	fmt.Printf("\n📦 Output Items (%d total):\n", len(resp.Output))
	fmt.Printf(strings.Repeat("-", 60) + "\n")

	for i, item := range resp.Output {
		itemType, _ := item["type"].(string)
		fmt.Printf("\n[%d] Type: %s\n", i+1, itemType)

		switch itemType {
		case "message":
			printMessageItem(item)
		case "function_call", "tool_call":
			printToolCallItem(item)
		case "function_call_output", "tool_result":
			printToolResultItem(item)
		case "reasoning":
			printReasoningItem(item)
		default:
			// Print raw item for unknown types
			itemBytes, _ := json.MarshalIndent(item, "    ", "  ")
			fmt.Printf("    %s\n", string(itemBytes))
		}
	}

	// Print usage stats
	if resp.Usage != nil {
		fmt.Printf("\n" + strings.Repeat("-", 60) + "\n")
		fmt.Printf("📈 Token Usage:\n")
		if prompt, ok := resp.Usage["prompt_tokens"]; ok {
			fmt.Printf("    Prompt tokens: %v\n", prompt)
		}
		if completion, ok := resp.Usage["completion_tokens"]; ok {
			fmt.Printf("    Completion tokens: %v\n", completion)
		}
		if total, ok := resp.Usage["total_tokens"]; ok {
			fmt.Printf("    Total tokens: %v\n", total)
		}
	}

	// Print error if any
	if resp.Error != nil {
		if msg, ok := resp.Error["message"].(string); ok && msg != "" {
			fmt.Printf("\n❌ Error: %s\n", msg)
		}
	}

	fmt.Printf("\n" + strings.Repeat("=", 60) + "\n")
}

func printMessageItem(item map[string]interface{}) {
	role, _ := item["role"].(string)
	fmt.Printf("    Role: %s\n", role)

	if content, ok := item["content"].([]interface{}); ok {
		for _, c := range content {
			if contentMap, ok := c.(map[string]interface{}); ok {
				contentType, _ := contentMap["type"].(string)
				if contentType == "output_text" || contentType == "text" {
					if text, ok := contentMap["text"].(string); ok {
						// Truncate long text
						displayText := text
						if len(displayText) > 500 {
							displayText = displayText[:500] + "... [truncated]"
						}
						fmt.Printf("    📝 Content:\n%s\n", indentString(displayText, "       "))
					}
				}
			}
		}
	} else if content, ok := item["content"].(string); ok {
		displayText := content
		if len(displayText) > 500 {
			displayText = displayText[:500] + "... [truncated]"
		}
		fmt.Printf("    📝 Content:\n%s\n", indentString(displayText, "       "))
	}
}

func printToolCallItem(item map[string]interface{}) {
	if id, ok := item["id"].(string); ok {
		fmt.Printf("    🔧 Tool Call ID: %s\n", id)
	}
	if callID, ok := item["call_id"].(string); ok {
		fmt.Printf("    🔧 Call ID: %s\n", callID)
	}
	if name, ok := item["name"].(string); ok {
		fmt.Printf("    📛 Tool Name: %s\n", name)
	}
	if args, ok := item["arguments"].(string); ok {
		// Try to pretty-print JSON arguments
		var prettyArgs bytes.Buffer
		if json.Indent(&prettyArgs, []byte(args), "       ", "  ") == nil {
			fmt.Printf("    📥 Arguments:\n       %s\n", prettyArgs.String())
		} else {
			displayArgs := args
			if len(displayArgs) > 300 {
				displayArgs = displayArgs[:300] + "... [truncated]"
			}
			fmt.Printf("    📥 Arguments: %s\n", displayArgs)
		}
	}
	if status, ok := item["status"].(string); ok {
		fmt.Printf("    📊 Status: %s\n", status)
	}
}

func printToolResultItem(item map[string]interface{}) {
	if callID, ok := item["call_id"].(string); ok {
		fmt.Printf("    🔧 Call ID: %s\n", callID)
	}
	if output, ok := item["output"].(string); ok {
		displayOutput := output
		if len(displayOutput) > 500 {
			displayOutput = displayOutput[:500] + "... [truncated]"
		}
		fmt.Printf("    📤 Output:\n%s\n", indentString(displayOutput, "       "))
	}
}

func printReasoningItem(item map[string]interface{}) {
	if text, ok := item["text"].(string); ok {
		displayText := text
		if len(displayText) > 500 {
			displayText = displayText[:500] + "... [truncated]"
		}
		fmt.Printf("    💭 Reasoning:\n%s\n", indentString(displayText, "       "))
	}
}

func guestLogin(baseURL, endpoint string) (string, error) {
	if endpoint == "" {
		endpoint = "/auth/guest-login"
	}

	url := baseURL + endpoint

	req, err := http.NewRequest("POST", url, bytes.NewReader([]byte("{}")))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("login failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	// Try different token field names
	if token, ok := result["access_token"].(string); ok {
		return token, nil
	}
	if token, ok := result["token"].(string); ok {
		return token, nil
	}

	return "", fmt.Errorf("no access_token in response: %v", result)
}

func indentString(s, indent string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = indent + line
	}
	return strings.Join(lines, "\n")
}
