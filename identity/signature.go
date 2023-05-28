package identity

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"

	"google.golang.org/protobuf/proto"
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

func SignProto(user PrivateUser, message proto.Message) ([]byte, error) {
	serialized, err := proto.Marshal(message)
	if err != nil {
		return nil, err
	}
	return sign(user.privateKey, serialized)
}

func VerifyProto(user PublicUser, message proto.Message, signature []byte) error {
	serialized, err := proto.Marshal(message)
	if err != nil {
		return err
	}
	return verify(user.publicKey, serialized, signature)
}
