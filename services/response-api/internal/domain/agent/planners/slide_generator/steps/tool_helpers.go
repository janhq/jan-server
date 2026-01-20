package steps

import (
	"encoding/json"
	"fmt"

	"jan-server/services/response-api/internal/domain/agent"
	"jan-server/services/response-api/internal/domain/status"

	"github.com/rs/zerolog/log"
)

func buildToolArguments(toolName string, params map[string]interface{}, input agent.ExecutionInput, description string) (map[string]interface{}, error) {
	switch toolName {
	case "google_search":
		query := ""
		if q, ok := params["q"].(string); ok && q != "" {
			query = q
		} else if q, ok := params["query"].(string); ok && q != "" {
			query = q
		} else if description != "" {
			query = description
		}

		if query == "" {
			return nil, fmt.Errorf("no search query provided")
		}
		return map[string]interface{}{"q": query}, nil

	case "image_search":
		query := ""
		if q, ok := params["q"].(string); ok && q != "" {
			query = q
		} else if q, ok := params["query"].(string); ok && q != "" {
			query = q
		} else if description != "" {
			query = description
		}

		if query == "" {
			return nil, fmt.Errorf("no image search query provided")
		}

		args := map[string]interface{}{"q": query}
		if num, ok := params["num"].(float64); ok && num > 0 {
			args["num"] = int(num)
		} else {
			args["num"] = 10
		}
		if gl, ok := params["gl"].(string); ok && gl != "" {
			args["gl"] = gl
		}
		if hl, ok := params["hl"].(string); ok && hl != "" {
			args["hl"] = hl
		}
		return args, nil

	case "scrape":
		urls := extractURLsFromPreviousOutput(input.PreviousOutput)
		if len(urls) == 0 {
			if urlParam, ok := params["url"].(string); ok {
				urls = []string{urlParam}
			} else if urlsParam, ok := params["urls"].([]interface{}); ok {
				for _, u := range urlsParam {
					if urlStr, ok := u.(string); ok {
						urls = append(urls, urlStr)
					}
				}
			}
		}
		if len(urls) == 0 {
			return nil, fmt.Errorf("no URLs available to scrape from previous search results")
		}
		return map[string]interface{}{"url": urls[0]}, nil

	default:
		toolArgs := make(map[string]interface{})
		for k, v := range params {
			if k != "tool" && k != "description" {
				toolArgs[k] = v
			}
		}
		return toolArgs, nil
	}
}

func extractURLsFromPreviousOutput(previousOutput json.RawMessage) []string {
	if len(previousOutput) == 0 {
		return nil
	}

	var output map[string]interface{}
	if err := json.Unmarshal(previousOutput, &output); err != nil {
		return nil
	}

	var urls []string
	if content, ok := output["content"].([]interface{}); ok {
		for _, item := range content {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if text, ok := itemMap["text"].(string); ok {
					var textData map[string]interface{}
					if err := json.Unmarshal([]byte(text), &textData); err == nil {
						urls = append(urls, extractURLsFromData(textData)...)
					}
				}
			}
		}
	}

	urls = append(urls, extractURLsFromData(output)...)
	log.Debug().Int("url_count", len(urls)).Msg("[slide_generator] extracted URLs from tool output")
	return urls
}

func extractURLsFromData(data map[string]interface{}) []string {
	var urls []string

	if organic, ok := data["organic"].([]interface{}); ok {
		for _, item := range organic {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if link, ok := itemMap["link"].(string); ok && link != "" {
					urls = append(urls, link)
				}
			}
		}
	}

	if results, ok := data["results"].([]interface{}); ok {
		for _, item := range results {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if url, ok := itemMap["source_url"].(string); ok && url != "" {
					urls = append(urls, url)
				} else if url, ok := itemMap["url"].(string); ok && url != "" {
					urls = append(urls, url)
				} else if link, ok := itemMap["link"].(string); ok && link != "" {
					urls = append(urls, link)
				}
			}
		}
	}

	return urls
}

func isNonCriticalToolForSlides(toolName string) bool {
	nonCriticalTools := map[string]bool{
		"google_search": true,
		"image_search":  true,
		"scrape":        true,
	}
	return nonCriticalTools[toolName]
}

func buildSkippedToolResultForSlides(toolName string, reason string, code string) *agent.ExecutionResult {
	output, _ := json.Marshal(map[string]interface{}{
		"skipped": true,
		"tool":    toolName,
		"reason":  reason,
		"code":    code,
	})
	return &agent.ExecutionResult{
		Status: status.StatusCompleted,
		Output: output,
	}
}

func getModelFromContext(input agent.ExecutionInput) string {
	if input.PlanContext != nil && input.PlanContext.Model != "" {
		return input.PlanContext.Model
	}
	return ""
}
