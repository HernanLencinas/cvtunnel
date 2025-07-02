package ccrypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"strings"
)

const CvtKeyPrefix = "ck-"

func Seed2PEM(seed string) ([]byte, error) {
	privateKey, err := seed2PrivateKey(seed)
	if err != nil {
		return nil, err
	}

	return privateKey2PEM(privateKey)
}

func seed2CvtKey(seed string) ([]byte, error) {
	privateKey, err := seed2PrivateKey(seed)
	if err != nil {
		return nil, err
	}

	return privateKey2CvtKey(privateKey)
}

func seed2PrivateKey(seed string) (*ecdsa.PrivateKey, error) {
	if seed == "" {
		return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	} else {
		return GenerateKeyGo119(elliptic.P256(), NewDetermRand([]byte(seed)))
	}
}

func privateKey2CvtKey(privateKey *ecdsa.PrivateKey) ([]byte, error) {
	b, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, err
	}

	encodedPrivateKey := make([]byte, base64.RawStdEncoding.EncodedLen(len(b)))
	base64.RawStdEncoding.Encode(encodedPrivateKey, b)

	return append([]byte(CvtKeyPrefix), encodedPrivateKey...), nil
}

func privateKey2PEM(privateKey *ecdsa.PrivateKey) ([]byte, error) {
	b, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, err
	}

	return pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: b}), nil
}

func cvtKey2PrivateKey(cvtKey []byte) (*ecdsa.PrivateKey, error) {
	rawCvtKey := cvtKey[len(CvtKeyPrefix):]

	decodedPrivateKey := make([]byte, base64.RawStdEncoding.DecodedLen(len(rawCvtKey)))
	_, err := base64.RawStdEncoding.Decode(decodedPrivateKey, rawCvtKey)
	if err != nil {
		return nil, err
	}

	return x509.ParseECPrivateKey(decodedPrivateKey)
}

func CvtKey2PEM(cvtKey []byte) ([]byte, error) {
	privateKey, err := cvtKey2PrivateKey(cvtKey)
	if err == nil {
		return privateKey2PEM(privateKey)
	}

	return nil, err
}

func IsCvtKey(cvtKey []byte) bool {
	return strings.HasPrefix(string(cvtKey), CvtKeyPrefix)
}
