package argon2

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid password",
			password: "mySecurePassword123!",
			wantErr:  false,
		},
		{
			name:     "simple password",
			password: "password",
			wantErr:  false,
		},
		{
			name:     "long password",
			password: strings.Repeat("a", 1000),
			wantErr:  false,
		},
		{
			name:     "password with special characters",
			password: "pä$$w0rd!@#$%^&*()_+-=[]{}|;':\",./<>?",
			wantErr:  false,
		},
		{
			name:     "empty password",
			password: "",
			wantErr:  true,
			errMsg:   "password cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HashPassword(tt.password)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Empty(t, hash)
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, hash)

			assert.True(t, strings.HasPrefix(hash, "$argon2id$v="))
			parts := strings.Split(hash, "$")
			assert.Len(t, parts, 6)
			assert.Equal(t, "argon2id", parts[1])

			assert.Contains(t, hash, "m=65536,t=3,p=2")

			valid, err := VerifyPassword(tt.password, hash)
			require.NoError(t, err)
			assert.True(t, valid)
		})
	}
}

func TestHashPassword_UniqueHashes(t *testing.T) {
	password := "testPassword123"

	hashes := make([]string, 10)
	for i := 0; i < 10; i++ {
		hash, err := HashPassword(password)
		require.NoError(t, err)
		hashes[i] = hash
	}

	for i := 0; i < len(hashes); i++ {
		for j := i + 1; j < len(hashes); j++ {
			assert.NotEqual(t, hashes[i], hashes[j], "hashes should be unique due to random salt")
		}
	}

	for i, hash := range hashes {
		valid, err := VerifyPassword(password, hash)
		require.NoError(t, err, "hash %d should be valid", i)
		assert.True(t, valid, "hash %d should verify successfully", i)
	}
}

func TestVerifyPassword(t *testing.T) {
	password := "testPassword123"
	hash, err := HashPassword(password)
	require.NoError(t, err)

	tests := []struct {
		name        string
		password    string
		hash        string
		expected    bool
		wantErr     bool
		errContains string
	}{
		{
			name:     "valid password and hash",
			password: password,
			hash:     hash,
			expected: true,
			wantErr:  false,
		},
		{
			name:     "wrong password",
			password: "wrongPassword",
			hash:     hash,
			expected: false,
			wantErr:  false,
		},
		{
			name:     "case sensitive password",
			password: "TestPassword123",
			hash:     hash,
			expected: false,
			wantErr:  false,
		},
		{
			name:        "empty password",
			password:    "",
			hash:        hash,
			expected:    false,
			wantErr:     true,
			errContains: "password cannot be empty",
		},
		{
			name:        "empty hash",
			password:    password,
			hash:        "",
			expected:    false,
			wantErr:     true,
			errContains: "hash cannot be empty",
		},
		{
			name:        "invalid hash format - too few parts",
			password:    password,
			hash:        "$argon2id$v=19$m=65536",
			expected:    false,
			wantErr:     true,
			errContains: "invalid hash format",
		},
		{
			name:        "invalid hash format - too many parts",
			password:    password,
			hash:        "$argon2id$v=19$m=65536,t=3,p=2$salt$hash$extra",
			expected:    false,
			wantErr:     true,
			errContains: "invalid hash format",
		},
		{
			name:        "unsupported hash type",
			password:    password,
			hash:        "$bcrypt$v=19$m=65536,t=3,p=2$salt$hash",
			expected:    false,
			wantErr:     true,
			errContains: "unsupported hash type",
		},
		{
			name:        "invalid version format",
			password:    password,
			hash:        "$argon2id$version=19$m=65536,t=3,p=2$salt$hash",
			expected:    false,
			wantErr:     true,
			errContains: "failed to parse version",
		},
		{
			name:        "invalid parameters format",
			password:    password,
			hash:        "$argon2id$v=19$memory=65536,time=3,parallel=2$salt$hash",
			expected:    false,
			wantErr:     true,
			errContains: "failed to parse parameters",
		},
		{
			name:        "invalid salt encoding",
			password:    password,
			hash:        "$argon2id$v=19$m=65536,t=3,p=2$invalid_base64!$hash",
			expected:    false,
			wantErr:     true,
			errContains: "failed to decode salt",
		},
		{
			name:        "invalid hash encoding",
			password:    password,
			hash:        "$argon2id$v=19$m=65536,t=3,p=2$dGVzdA==$invalid_base64!",
			expected:    false,
			wantErr:     true,
			errContains: "failed to decode hash",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := VerifyPassword(tt.password, tt.hash)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.False(t, result)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVerifyPassword_ConstantTime(t *testing.T) {
	password := "testPassword"
	hash, err := HashPassword(password)
	require.NoError(t, err)

	testCases := []string{
		"wrongPassword1",
		"wrongPassword2",
		"a",
		strings.Repeat("x", 1000),
	}

	for _, wrongPassword := range testCases {
		valid, err := VerifyPassword(wrongPassword, hash)
		require.NoError(t, err)
		assert.False(t, valid)
	}
}

func TestIsArgon2Hash(t *testing.T) {
	validHash, err := HashPassword("testPassword")
	require.NoError(t, err)

	tests := []struct {
		name     string
		hash     string
		expected bool
	}{
		{
			name:     "valid argon2id hash",
			hash:     validHash,
			expected: true,
		},
		{
			name:     "empty string",
			hash:     "",
			expected: false,
		},
		{
			name:     "bcrypt hash",
			hash:     "$2b$10$N9qo8uLOickgx2ZMRZoMye1234567890abcdefghijklmnop",
			expected: false,
		},
		{
			name:     "invalid format - too few parts",
			hash:     "$argon2id$v=19$m=65536",
			expected: false,
		},
		{
			name:     "invalid format - wrong algorithm",
			hash:     "$scrypt$v=19$m=65536,t=3,p=2$salt$hash",
			expected: false,
		},
		{
			name:     "plain text",
			hash:     "plainPassword",
			expected: false,
		},
		{
			name:     "argon2i hash (not argon2id)",
			hash:     "$argon2i$v=19$m=65536,t=3,p=2$salt$hash",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsArgon2Hash(tt.hash)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestArgon2Constants(t *testing.T) {
	assert.Equal(t, uint32(64*1024), uint32(Argon2Memory), "memory should be 64MB")
	assert.Equal(t, uint32(3), uint32(Argon2Iterations), "iterations should be 3")
	assert.Equal(t, uint8(2), uint8(Argon2Parallelism), "parallelism should be 2")
	assert.Equal(t, 16, Argon2SaltLength, "salt length should be 16 bytes")
	assert.Equal(t, 32, Argon2KeyLength, "key length should be 32 bytes")
}

func BenchmarkHashPassword(b *testing.B) {
	password := "benchmarkPassword123!"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := HashPassword(password)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkVerifyPassword(b *testing.B) {
	password := "benchmarkPassword123!"
	hash, err := HashPassword(password)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		valid, err := VerifyPassword(password, hash)
		if err != nil {
			b.Fatal(err)
		}
		if !valid {
			b.Fatal("password should be valid")
		}
	}
}

func BenchmarkVerifyPassword_Wrong(b *testing.B) {
	password := "benchmarkPassword123!"
	wrongPassword := "wrongPassword123!"
	hash, err := HashPassword(password)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		valid, err := VerifyPassword(wrongPassword, hash)
		if err != nil {
			b.Fatal(err)
		}
		if valid {
			b.Fatal("wrong password should not be valid")
		}
	}
}
