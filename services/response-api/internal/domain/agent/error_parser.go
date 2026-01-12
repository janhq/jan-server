package agent

import (
	"regexp"
)

// CodeExecutionError represents a parsed error from code execution.
type CodeExecutionError struct {
	Type       string   // "ModuleNotFoundError", "ImportError", etc.
	ModuleName string   // The missing module
	Message    string   // Full error message
	Traceback  []string // Stack trace if available
}

// Regex patterns for different Python error types
var (
	moduleNotFoundRegex = regexp.MustCompile(`ModuleNotFoundError: No module named '([^']+)'`)
	importErrorRegex    = regexp.MustCompile(`ImportError: cannot import name '([^']+)' from '([^']+)'`)
	noModuleNamedRegex  = regexp.MustCompile(`No module named '([^']+)'`)
)

// ParseCodeExecutionError parses a code execution error string and extracts
// structured information about the error type and missing module.
func ParseCodeExecutionError(errorText string) *CodeExecutionError {
	// Check for ModuleNotFoundError
	if matches := moduleNotFoundRegex.FindStringSubmatch(errorText); len(matches) > 1 {
		return &CodeExecutionError{
			Type:       "ModuleNotFoundError",
			ModuleName: matches[1],
			Message:    errorText,
		}
	}

	// Check for ImportError with module info
	if matches := importErrorRegex.FindStringSubmatch(errorText); len(matches) > 2 {
		return &CodeExecutionError{
			Type:       "ImportError",
			ModuleName: matches[2], // The package, not the symbol
			Message:    errorText,
		}
	}

	// Fallback: check for generic "No module named" pattern
	if matches := noModuleNamedRegex.FindStringSubmatch(errorText); len(matches) > 1 {
		return &CodeExecutionError{
			Type:       "ModuleNotFoundError",
			ModuleName: matches[1],
			Message:    errorText,
		}
	}

	return nil
}

// IsRetryableWithInstall returns true if the error can be resolved by
// installing a missing package and retrying.
func IsRetryableWithInstall(err *CodeExecutionError) bool {
	if err == nil {
		return false
	}
	return err.Type == "ModuleNotFoundError" || err.Type == "ImportError"
}

// GetMissingModule returns the module name from a code execution error,
// or empty string if no module error was found.
func GetMissingModule(errorText string) string {
	parsed := ParseCodeExecutionError(errorText)
	if parsed != nil {
		return parsed.ModuleName
	}
	return ""
}
