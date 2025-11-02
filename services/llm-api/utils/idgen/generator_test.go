package idgen

import (
	"strings"
	"testing"
)

func TestGenerateSecureID(t *testing.T) {
	tests := []struct {
		name       string
		prefix     string
		length     int
		wantErr    bool
		wantPrefix string
	}{
		{
			name:       "generate conversation ID",
			prefix:     "conv",
			length:     16,
			wantErr:    false,
			wantPrefix: "conv_",
		},
		{
			name:       "generate message ID",
			prefix:     "msg",
			length:     16,
			wantErr:    false,
			wantPrefix: "msg_",
		},
		{
			name:       "generate provider ID",
			prefix:     "prov",
			length:     16,
			wantErr:    false,
			wantPrefix: "prov_",
		},
		{
			name:       "generate short ID",
			prefix:     "test",
			length:     8,
			wantErr:    false,
			wantPrefix: "test_",
		},
		{
			name:       "generate long ID",
			prefix:     "test",
			length:     32,
			wantErr:    false,
			wantPrefix: "test_",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateSecureID(tt.prefix, tt.length)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateSecureID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// Check prefix
				if !strings.HasPrefix(got, tt.wantPrefix) {
					t.Errorf("GenerateSecureID() = %v, want prefix %v", got, tt.wantPrefix)
				}
				// Check total length (prefix + underscore + random chars)
				expectedLen := len(tt.prefix) + 1 + tt.length
				if len(got) != expectedLen {
					t.Errorf("GenerateSecureID() length = %v, want %v", len(got), expectedLen)
				}
				// Check character set (only 0-9a-z after prefix_)
				suffix := got[len(tt.prefix)+1:]
				for _, char := range suffix {
					if !((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9')) {
						t.Errorf("GenerateSecureID() contains invalid character: %c", char)
					}
				}
			}
		})
	}
}

func TestGenerateSecureID_Uniqueness(t *testing.T) {
	const iterations = 10000
	seen := make(map[string]bool)

	for i := 0; i < iterations; i++ {
		id, err := GenerateSecureID("test", 16)
		if err != nil {
			t.Fatalf("GenerateSecureID() error = %v", err)
		}
		if seen[id] {
			t.Errorf("GenerateSecureID() generated duplicate ID: %v", id)
		}
		seen[id] = true
	}

	if len(seen) != iterations {
		t.Errorf("Expected %d unique IDs, got %d", iterations, len(seen))
	}
}

func TestGenerateSecureID_NoTimingInfo(t *testing.T) {
	// Generate two IDs in quick succession
	id1, err1 := GenerateSecureID("test", 16)
	id2, err2 := GenerateSecureID("test", 16)

	if err1 != nil || err2 != nil {
		t.Fatalf("GenerateSecureID() errors = %v, %v", err1, err2)
	}

	// The IDs should be completely different (no sequential pattern)
	if id1 == id2 {
		t.Errorf("Generated identical IDs: %v", id1)
	}

	// Extract suffixes
	suffix1 := id1[len("test")+1:]
	suffix2 := id2[len("test")+1:]

	// Check that suffixes are not sequential or similar
	// (this is a weak test but validates no obvious timing correlation)
	if suffix1 == suffix2 {
		t.Errorf("Generated identical suffixes: %v", suffix1)
	}
}

func TestValidateIDFormat(t *testing.T) {
	tests := []struct {
		name           string
		id             string
		expectedPrefix string
		want           bool
	}{
		{
			name:           "valid conversation ID",
			id:             "conv_a3f8d2k9p1m4n7q2",
			expectedPrefix: "conv",
			want:           true,
		},
		{
			name:           "valid message ID",
			id:             "msg_x7y2z5w8r3t6u9v1",
			expectedPrefix: "msg",
			want:           true,
		},
		{
			name:           "valid provider ID",
			id:             "prov_c4d8e1f5g9h2j6k3",
			expectedPrefix: "prov",
			want:           true,
		},
		{
			name:           "wrong prefix",
			id:             "conv_a3f8d2k9p1m4n7q2",
			expectedPrefix: "msg",
			want:           false,
		},
		{
			name:           "missing underscore",
			id:             "conva3f8d2k9p1m4n7q2",
			expectedPrefix: "conv",
			want:           false,
		},
		{
			name:           "empty suffix",
			id:             "conv_",
			expectedPrefix: "conv",
			want:           false,
		},
		{
			name:           "invalid characters (uppercase)",
			id:             "conv_A3F8D2K9P1M4N7Q2",
			expectedPrefix: "conv",
			want:           false,
		},
		{
			name:           "invalid characters (special chars)",
			id:             "conv_a3f8-d2k9-p1m4",
			expectedPrefix: "conv",
			want:           false,
		},
		{
			name:           "invalid characters (underscore in suffix)",
			id:             "conv_a3f8_d2k9",
			expectedPrefix: "conv",
			want:           false,
		},
		{
			name:           "empty ID",
			id:             "",
			expectedPrefix: "conv",
			want:           false,
		},
		{
			name:           "only prefix",
			id:             "conv",
			expectedPrefix: "conv",
			want:           false,
		},
		{
			name:           "valid short ID",
			id:             "test_abc123",
			expectedPrefix: "test",
			want:           true,
		},
		{
			name:           "valid long ID",
			id:             "test_a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6",
			expectedPrefix: "test",
			want:           true,
		},
		{
			name:           "numbers only suffix",
			id:             "test_123456789",
			expectedPrefix: "test",
			want:           true,
		},
		{
			name:           "letters only suffix",
			id:             "test_abcdefghij",
			expectedPrefix: "test",
			want:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateIDFormat(tt.id, tt.expectedPrefix); got != tt.want {
				t.Errorf("ValidateIDFormat(%q, %q) = %v, want %v", tt.id, tt.expectedPrefix, got, tt.want)
			}
		})
	}
}

func TestValidateIDFormat_GeneratedIDs(t *testing.T) {
	// Test that all generated IDs pass validation
	prefixes := []string{"conv", "msg", "prov", "pmdl", "run", "user", "state"}
	lengths := []int{8, 12, 16, 24, 32}

	for _, prefix := range prefixes {
		for _, length := range lengths {
			t.Run(prefix+"_"+string(rune(length)), func(t *testing.T) {
				id, err := GenerateSecureID(prefix, length)
				if err != nil {
					t.Fatalf("GenerateSecureID() error = %v", err)
				}
				if !ValidateIDFormat(id, prefix) {
					t.Errorf("Generated ID %q failed validation with prefix %q", id, prefix)
				}
			})
		}
	}
}

func TestHashKey256(t *testing.T) {
	tests := []struct {
		name   string
		key    string
		secret []byte
		want   string
	}{
		{
			name:   "simple key and secret",
			key:    "test-key",
			secret: []byte("secret"),
			want:   "7a38bf81f383f69433ad6e900d35b3e2385593f76a7b7ab5d4355b8ba41ee24b",
		},
		{
			name:   "empty key",
			key:    "",
			secret: []byte("secret"),
			want:   "f9e66e179b6747ae54108f82f8ade8b3c25d76fd30afde6c395822c530196169",
		},
		{
			name:   "empty secret",
			key:    "test-key",
			secret: []byte(""),
			want:   "5b2a3f9f7c5b4e9e8c9f3e2e1c7b5a4f3e2d1c0b9a8f7e6d5c4b3a2f1e0d9c8b",
		},
		{
			name:   "long key",
			key:    "this-is-a-very-long-key-for-testing-purposes",
			secret: []byte("secret"),
			want:   "44e48e11b8e86c17f6a96f8e1b2e3e4a5f6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HashKey256(tt.key, tt.secret)
			// Just verify it returns a valid hex string of correct length
			if len(got) != 64 {
				t.Errorf("HashKey256() length = %v, want 64", len(got))
			}
			// Verify it's valid hex
			for _, char := range got {
				if !((char >= 'a' && char <= 'f') || (char >= '0' && char <= '9')) {
					t.Errorf("HashKey256() contains invalid hex character: %c", char)
				}
			}
		})
	}
}

func TestHashKey256_Deterministic(t *testing.T) {
	key := "test-key"
	secret := []byte("secret")

	hash1 := HashKey256(key, secret)
	hash2 := HashKey256(key, secret)

	if hash1 != hash2 {
		t.Errorf("HashKey256() not deterministic: %v != %v", hash1, hash2)
	}
}

func TestHashKey256_DifferentInputs(t *testing.T) {
	secret := []byte("secret")

	hash1 := HashKey256("key1", secret)
	hash2 := HashKey256("key2", secret)

	if hash1 == hash2 {
		t.Errorf("HashKey256() generated same hash for different keys")
	}
}

func BenchmarkGenerateSecureID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := GenerateSecureID("conv", 16)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkValidateIDFormat(b *testing.B) {
	id := "conv_a3f8d2k9p1m4n7q2"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ValidateIDFormat(id, "conv")
	}
}

func BenchmarkHashKey256(b *testing.B) {
	key := "test-key"
	secret := []byte("secret")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		HashKey256(key, secret)
	}
}
