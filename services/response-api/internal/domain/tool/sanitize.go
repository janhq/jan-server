package tool

import (
	"regexp"
	"strings"
)

const sandboxWritableDir = "/home/user"

func normalizeCodeArguments(toolName string, args map[string]interface{}) map[string]interface{} {
	if toolName != "aio_code_execute" || args == nil {
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
	replacements := []struct {
		re *regexp.Regexp
		fn func([]string) string
	}{
		{
			re: regexp.MustCompile(`(?i)(\bsavefig\s*\(\s*)(['"])([^'"]+)\2`),
			fn: func(m []string) string {
				return m[1] + m[2] + rewritePath(m[3]) + m[2]
			},
		},
		{
			re: regexp.MustCompile(`(?i)(\bto_csv\s*\(\s*)(['"])([^'"]+)\2`),
			fn: func(m []string) string {
				return m[1] + m[2] + rewritePath(m[3]) + m[2]
			},
		},
		{
			re: regexp.MustCompile(`(?i)(\bto_json\s*\(\s*)(['"])([^'"]+)\2`),
			fn: func(m []string) string {
				return m[1] + m[2] + rewritePath(m[3]) + m[2]
			},
		},
		{
			re: regexp.MustCompile(`(?i)(\bto_excel\s*\(\s*)(['"])([^'"]+)\2`),
			fn: func(m []string) string {
				return m[1] + m[2] + rewritePath(m[3]) + m[2]
			},
		},
		{
			re: regexp.MustCompile(`(?i)(\bto_parquet\s*\(\s*)(['"])([^'"]+)\2`),
			fn: func(m []string) string {
				return m[1] + m[2] + rewritePath(m[3]) + m[2]
			},
		},
		{
			re: regexp.MustCompile(`(?i)(\bto_pickle\s*\(\s*)(['"])([^'"]+)\2`),
			fn: func(m []string) string {
				return m[1] + m[2] + rewritePath(m[3]) + m[2]
			},
		},
		{
			re: regexp.MustCompile(`(?i)(\bto_feather\s*\(\s*)(['"])([^'"]+)\2`),
			fn: func(m []string) string {
				return m[1] + m[2] + rewritePath(m[3]) + m[2]
			},
		},
		{
			re: regexp.MustCompile(`(?i)(\bopen\s*\(\s*)(['"])([^'"]+)\2(\s*,\s*)(['"])([^'"]*)(['"])`),
			fn: func(m []string) string {
				mode := m[6]
				if !isWriteMode(mode) {
					return m[0]
				}
				return m[1] + m[2] + rewritePath(m[3]) + m[2] + m[4] + m[5] + mode + m[7]
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
