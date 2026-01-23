package steps

import "regexp"

var (
	reEmail       = regexp.MustCompile(`(?i)[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`)
	rePhone       = regexp.MustCompile(`\+?\d[\d\s().-]{7,}\d`)
	reBearer      = regexp.MustCompile(`(?i)bearer\s+[a-z0-9._\-\~+/=]+`)
	reAuthHeader  = regexp.MustCompile(`(?i)authorization\s*[:=]\s*[^\s]+`)
	reKeyValue    = regexp.MustCompile(`(?i)\b(api[-_]?key|token|secret|password)\b\s*[:=]\s*[^\s]+`)
	reJSONSecret  = regexp.MustCompile(`(?i)"(api[-_]?key|token|secret|password)"\s*:\s*"[^"]+"`)
	reQuerySecret = regexp.MustCompile(`(?i)([?&](?:token|sig|signature|key|auth|expires|x-amz-signature|x-amz-security-token|x-amz-credential|x-amz-date|x-amz-algorithm|x-amz-expires)=)[^&\s]+`)
)

func sanitizeForLog(text string) string {
	if text == "" {
		return text
	}
	sanitized := text
	sanitized = reBearer.ReplaceAllString(sanitized, "Bearer [REDACTED]")
	sanitized = reAuthHeader.ReplaceAllString(sanitized, "authorization=[REDACTED]")
	sanitized = reKeyValue.ReplaceAllString(sanitized, "$1=[REDACTED]")
	sanitized = reJSONSecret.ReplaceAllString(sanitized, "\"$1\":\"[REDACTED]\"")
	sanitized = reQuerySecret.ReplaceAllString(sanitized, "$1[REDACTED]")
	sanitized = reEmail.ReplaceAllString(sanitized, "[REDACTED_EMAIL]")
	sanitized = rePhone.ReplaceAllString(sanitized, "[REDACTED_PHONE]")
	return sanitized
}
