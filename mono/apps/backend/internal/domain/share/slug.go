package share

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
)

const (
	// SlugLength is the length of the random slug (22 chars = ~128 bits entropy in base62)
	SlugLength = 22

	// Base62Charset contains alphanumeric characters for URL-safe slugs
	Base62Charset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

	// MaxSlugRetries is the maximum number of attempts to generate a unique slug
	MaxSlugRetries = 5
)

// SlugGenerator generates cryptographically random slugs for share URLs
type SlugGenerator struct {
	repo ShareRepository
}

// NewSlugGenerator creates a new slug generator
func NewSlugGenerator(repo ShareRepository) *SlugGenerator {
	return &SlugGenerator{repo: repo}
}

// GenerateUniqueSlug generates a unique, cryptographically random 22-character base62 slug
// It retries up to MaxSlugRetries times if a collision is detected
func (g *SlugGenerator) GenerateUniqueSlug(ctx context.Context) (string, error) {
	for i := 0; i < MaxSlugRetries; i++ {
		slug, err := GenerateSlug()
		if err != nil {
			return "", fmt.Errorf("failed to generate slug: %w", err)
		}

		// Check for collision
		exists, err := g.repo.SlugExists(ctx, slug)
		if err != nil {
			return "", fmt.Errorf("failed to check slug existence: %w", err)
		}

		if !exists {
			return slug, nil
		}
		// Collision detected, retry
	}

	return "", fmt.Errorf("failed to generate unique slug after %d attempts", MaxSlugRetries)
}

// GenerateSlug generates a cryptographically random 22-character base62 slug
// This provides approximately 131 bits of entropy (22 * log2(62) ≈ 131)
func GenerateSlug() (string, error) {
	charsetLen := big.NewInt(int64(len(Base62Charset)))
	result := make([]byte, SlugLength)

	for i := 0; i < SlugLength; i++ {
		randomIndex, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", fmt.Errorf("failed to generate random number: %w", err)
		}
		result[i] = Base62Charset[randomIndex.Int64()]
	}

	return string(result), nil
}

// ValidateSlug checks if a slug has the correct format
func ValidateSlug(slug string) bool {
	if len(slug) != SlugLength {
		return false
	}

	for _, c := range slug {
		if !isBase62Char(c) {
			return false
		}
	}

	return true
}

// isBase62Char checks if a character is in the base62 charset
func isBase62Char(c rune) bool {
	return (c >= '0' && c <= '9') || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

// GenerateSharePublicID generates a public ID for a share (shr_xxx format)
func GenerateSharePublicID() (string, error) {
	const idLength = 16
	charsetLen := big.NewInt(int64(len(Base62Charset)))
	result := make([]byte, idLength)

	for i := 0; i < idLength; i++ {
		randomIndex, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", fmt.Errorf("failed to generate random number: %w", err)
		}
		// Use lowercase only for public ID suffix
		idx := randomIndex.Int64()
		if idx < 10 {
			result[i] = byte('0' + idx)
		} else {
			result[i] = byte('a' + (idx - 10) % 26)
		}
	}

	return "shr_" + string(result), nil
}
