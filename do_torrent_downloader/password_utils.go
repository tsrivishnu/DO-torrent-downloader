package doTorrentDownloader

import (
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/pbkdf2"
)

func generateQbittorrentHash(password string) (string, error) {
	// 1. Generate a random salt (16 bytes)
	salt := make([]byte, 16)
	_, err := rand.Read(salt)
	if err != nil {
		return "", err
	}

	// 2. PBKDF2 parameters
	// Iterations: 100,000 (standard for qBittorrent 4.2+)
	// Key length: 64 bytes (512 bits)
	// Digest: SHA512
	iterations := 100000
	keyLen := 64
	dk := pbkdf2.Key([]byte(password), salt, iterations, keyLen, sha512.New)

	// 3. Base64 encode salt and hash
	saltB64 := base64.StdEncoding.EncodeToString(salt)
	hashB64 := base64.StdEncoding.EncodeToString(dk)

	// 4. Format: @ByteArray(salt:hash)
	return fmt.Sprintf("@ByteArray(%s:%s)", saltB64, hashB64), nil
}
