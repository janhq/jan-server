package tool

import (
	"regexp"
	"strings"
)

const sandboxWritableDir = "/home/user"

func normalizeCodeArguments(toolName string, args map[string]interface{}) map[string]interface{} {
	if toolName != "sandbox_code_execute" || args == nil {
		return args
	}

	code, ok := args["code"].(string)
	if !ok || strings.TrimSpace(code) == "" {
		return args
	}

	updated := normalizeSandboxFilePaths(code)
	if updated == code {
		return args
	}

	cloned := make(map[string]interface{}, len(args))
	for k, v := range args {
		cloned[k] = v
	}
	cloned["code"] = updated
	return cloned
}

func normalizeSandboxFilePaths(code string) string {
	// Helper type for replacement patterns
	// Go's regexp doesn't support backreferences, so we create separate patterns for single/double quotes
	type replacement struct {
		re *regexp.Regexp
		fn func([]string) string
	}

	replacements := []replacement{
		// savefig with double quotes
		{
			re: regexp.MustCompile(`(?i)(\bsavefig\s*\(\s*)(")([^"]+)(")`),
			fn: func(m []string) string {
				return m[1] + m[2] + rewritePath(m[3]) + m[4]
			},
		},
		// savefig with single quotes
		{
			re: regexp.MustCompile(`(?i)(\bsavefig\s*\(\s*)(')([^']+)(')`),
			fn: func(m []string) string {
				return m[1] + m[2] + rewritePath(m[3]) + m[4]
			},
		},
		// to_csv with double quotes
		{
			re: regexp.MustCompile(`(?i)(\bto_csv\s*\(\s*)(")([^"]+)(")`),
			fn: func(m []string) string {
				return m[1] + m[2] + rewritePath(m[3]) + m[4]
			},
		},
		// to_csv with single quotes
		{
			re: regexp.MustCompile(`(?i)(\bto_csv\s*\(\s*)(')([^']+)(')`),
			fn: func(m []string) string {
				return m[1] + m[2] + rewritePath(m[3]) + m[4]
			},
		},
		// to_json with double quotes
		{
			re: regexp.MustCompile(`(?i)(\bto_json\s*\(\s*)(")([^"]+)(")`),
			fn: func(m []string) string {
				return m[1] + m[2] + rewritePath(m[3]) + m[4]
			},
		},
		// to_json with single quotes
		{
			re: regexp.MustCompile(`(?i)(\bto_json\s*\(\s*)(')([^']+)(')`),
			fn: func(m []string) string {
				return m[1] + m[2] + rewritePath(m[3]) + m[4]
			},
		},
		// to_excel with double quotes
		{
			re: regexp.MustCompile(`(?i)(\bto_excel\s*\(\s*)(")([^"]+)(")`),
			fn: func(m []string) string {
				return m[1] + m[2] + rewritePath(m[3]) + m[4]
			},
		},
		// to_excel with single quotes
		{
			re: regexp.MustCompile(`(?i)(\bto_excel\s*\(\s*)(')([^']+)(')`),
			fn: func(m []string) string {
				return m[1] + m[2] + rewritePath(m[3]) + m[4]
			},
		},
		// to_parquet with double quotes
		{
			re: regexp.MustCompile(`(?i)(\bto_parquet\s*\(\s*)(")([^"]+)(")`),
			fn: func(m []string) string {
				return m[1] + m[2] + rewritePath(m[3]) + m[4]
			},
		},
		// to_parquet with single quotes
		{
			re: regexp.MustCompile(`(?i)(\bto_parquet\s*\(\s*)(')([^']+)(')`),
			fn: func(m []string) string {
				return m[1] + m[2] + rewritePath(m[3]) + m[4]
			},
		},
		// to_pickle with double quotes
		{
			re: regexp.MustCompile(`(?i)(\bto_pickle\s*\(\s*)(")([^"]+)(")`),
			fn: func(m []string) string {
				return m[1] + m[2] + rewritePath(m[3]) + m[4]
			},
		},
		// to_pickle with single quotes
		{
			re: regexp.MustCompile(`(?i)(\bto_pickle\s*\(\s*)(')([^']+)(')`),
			fn: func(m []string) string {
				return m[1] + m[2] + rewritePath(m[3]) + m[4]
			},
		},
		// to_feather with double quotes
		{
			re: regexp.MustCompile(`(?i)(\bto_feather\s*\(\s*)(")([^"]+)(")`),
			fn: func(m []string) string {
				return m[1] + m[2] + rewritePath(m[3]) + m[4]
			},
		},
		// to_feather with single quotes
		{
			re: regexp.MustCompile(`(?i)(\bto_feather\s*\(\s*)(')([^']+)(')`),
			fn: func(m []string) string {
				return m[1] + m[2] + rewritePath(m[3]) + m[4]
			},
		},
		// open with double quotes for path and mode
		{
			re: regexp.MustCompile(`(?i)(\bopen\s*\(\s*)(")([^"]+)(")(\s*,\s*)(")([^"]*)(")`),
			fn: func(m []string) string {
				mode := m[7]
				if !isWriteMode(mode) {
					return m[0]
				}
				return m[1] + m[2] + rewritePath(m[3]) + m[4] + m[5] + m[6] + mode + m[8]
			},
		},
		// open with single quotes for path and mode
		{
			re: regexp.MustCompile(`(?i)(\bopen\s*\(\s*)(')([^']+)(')(\s*,\s*)(')([^']*)(')`),
			fn: func(m []string) string {
				mode := m[7]
				if !isWriteMode(mode) {
					return m[0]
				}
				return m[1] + m[2] + rewritePath(m[3]) + m[4] + m[5] + m[6] + mode + m[8]
			},
		},
	}

	updated := code
	for _, repl := range replacements {
		updated = replaceAllStringSubmatchFunc(updated, repl.re, repl.fn)
	}
	return updated
}

func rewritePath(path string) string {
	clean := strings.TrimSpace(path)
	if clean == "" || strings.Contains(clean, "://") {
		return path
	}
	if strings.HasPrefix(clean, "/") || strings.HasPrefix(clean, "~") || strings.Contains(clean, ":\\") {
		return path
	}

	clean = strings.TrimPrefix(clean, "./")
	for strings.HasPrefix(clean, "../") {
		clean = strings.TrimPrefix(clean, "../")
	}
	if clean == "" {
		return path
	}
	return sandboxWritableDir + "/" + clean
}

func isWriteMode(mode string) bool {
	return strings.Contains(mode, "w") || strings.Contains(mode, "a") || strings.Contains(mode, "x")
}

func replaceAllStringSubmatchFunc(input string, re *regexp.Regexp, fn func([]string) string) string {
	matches := re.FindAllStringSubmatchIndex(input, -1)
	if len(matches) == 0 {
		return input
	}

	var b strings.Builder
	last := 0
	for _, match := range matches {
		b.WriteString(input[last:match[0]])
		sub := input[match[0]:match[1]]
		submatches := re.FindStringSubmatch(sub)
		if len(submatches) == 0 {
			b.WriteString(sub)
		} else {
			b.WriteString(fn(submatches))
		}
		last = match[1]
	}
	b.WriteString(input[last:])
	return b.String()
}
