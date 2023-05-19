package identity

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
)

var (
	keySize = 512
	hash    = crypto.SHA256
)

type PrivateUser struct {
	Id         string
	privateKey *rsa.PrivateKey
}

type PublicUser struct {
	Id        string
	Address   string
	publicKey *rsa.PublicKey
}

func GenUser(address string) (PrivateUser, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, keySize)
	if err != nil {
		return PrivateUser{}, err
	}

	pubString := base64.StdEncoding.EncodeToString(x509.MarshalPKCS1PublicKey(&privateKey.PublicKey))
	if err != nil {
		return PrivateUser{}, err
	}

	unsignedId := fmt.Sprintf("%s@%s", address, pubString)

	hasher := hash.New()
	hasher.Write([]byte(unsignedId))
	signature, err := rsa.SignPSS(rand.Reader, privateKey, hash, hasher.Sum(nil), nil)
	if err != nil {
		return PrivateUser{}, err
	}

	fullId := fmt.Sprintf("%s#%s", unsignedId, base64.StdEncoding.EncodeToString(signature))

	return PrivateUser{Id: fullId, privateKey: privateKey}, nil
}

var (
	userIdPattern        = regexp.MustCompile(`(\S+)@(\S+)#(\S+)`)
	errUserParseIdFormat = errors.New("Bad user id format")
)

func ParseUser(id string) (PublicUser, error) {
	matched := userIdPattern.FindStringSubmatch(id)
	if matched == nil {
		return PublicUser{}, errUserParseIdFormat
	}
	address := matched[1]
	pubString := matched[2]
	signatureString := matched[3]

	pubMarsh, err := base64.StdEncoding.DecodeString(pubString)
	if err != nil {
		return PublicUser{}, err
	}

	publicKey, err := x509.ParsePKCS1PublicKey(pubMarsh)
	if err != nil {
		return PublicUser{}, err
	}

	signature, err := base64.StdEncoding.DecodeString(signatureString)
	if err != nil {
		return PublicUser{}, err
	}

	unsignedId := fmt.Sprintf("%s@%s", address, pubString)
	hasher := hash.New()
	hasher.Write([]byte(unsignedId))
	err = rsa.VerifyPSS(publicKey, hash, hasher.Sum(nil), signature, nil)
	if err != nil {
		return PublicUser{}, err
	}

	return PublicUser{Id: id, Address: address, publicKey: publicKey}, nil
}
