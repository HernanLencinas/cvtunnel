package ccrypto

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"

	"golang.org/x/crypto/ssh"
)

// GenerateKey generates a PEM key
func GenerateKey(seed string) ([]byte, error) {
	return Seed2PEM(seed)
}

// GenerateKeyFile generates an CvtKey
func GenerateKeyFile(keyFilePath, seed string) error {
	cvtKey, err := seed2CvtKey(seed)
	if err != nil {
		return err
	}

	if keyFilePath == "-" {
		fmt.Print(string(cvtKey))
		return nil
	}
	return os.WriteFile(keyFilePath, cvtKey, 0600)
}

// FingerprintKey calculates the SHA256 hash of an SSH public key
func FingerprintKey(k ssh.PublicKey) string {
	bytes := sha256.Sum256(k.Marshal())
	return base64.StdEncoding.EncodeToString(bytes[:])
}
