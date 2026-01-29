package domain

// AuthMethod describes how a caller authenticated with the API.
type AuthMethod string

const (
	AuthMethodJWT    AuthMethod = "jwt"
	AuthMethodAPIKey AuthMethod = "apikey"
)

// Principal captures normalized caller identity independent of auth mechanism.
type Principal struct {
	ID              string
	AuthMethod      AuthMethod
	Subject         string
	Issuer          string
	AuthorizedParty string
	Audience        []string
	Username        string
	Email           string
	Name            string
	Roles           []string
	Groups          []string
	FeatureFlags    []string
	Scopes          []string
	Attributes      map[string]any
	Credentials     map[string]string
}

// HasScope checks if the principal possesses a scope.
func (p Principal) HasScope(scope string) bool {
	for _, s := range p.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}
