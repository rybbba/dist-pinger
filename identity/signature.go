package identity

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
)

var (
	signatureHash = crypto.SHA256
)

func sign(privateKey *rsa.PrivateKey, message []byte) ([]byte, error) {
	hasher := signatureHash.New()
	hasher.Write(message)
	signature, err := rsa.SignPSS(rand.Reader, privateKey, signatureHash, hasher.Sum(nil), nil)
	return signature, err
}

func verify(publicKey *rsa.PublicKey, message []byte, signature []byte) error {
	hasher := signatureHash.New()
	hasher.Write(message)
	err := rsa.VerifyPSS(publicKey, signatureHash, hasher.Sum(nil), signature, nil)
	return err
}

func SignMessage(user PrivateUser, message []byte) ([]byte, error) {
	return sign(user.privateKey, message)
}

func VerifyMessage(user PublicUser, message []byte, signature []byte) error {
	return verify(user.publicKey, message, signature)
}
