package identity

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
)

var (
	userKeySize = 512
)

type PrivateUser struct {
	Id         string
	Address    string
	privateKey *rsa.PrivateKey
}

type PublicUser struct {
	Id        string
	Address   string
	publicKey *rsa.PublicKey
}

type privateUserFile struct {
	Id         string `json:"id"`
	Address    string `json:"address"`
	PrivateKey []byte `json:"privatekey"`
}

func WriteUser(user PrivateUser, path string) error {
	fi, err := os.Create(path)
	if err != nil {
		return err
	}

	data, err := json.Marshal(privateUserFile{Id: user.Id, Address: user.Address, PrivateKey: x509.MarshalPKCS1PrivateKey(user.privateKey)})
	if err != nil {
		return err
	}
	_, err = fi.Write(data)
	if err != nil {
		return err
	}

	err = fi.Close()
	if err != nil {
		log.Printf("Error closing output file: %v", err)
	}
	return nil
}

func ReadUser(path string) (PrivateUser, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return PrivateUser{}, err
	}
	var userFile privateUserFile
	err = json.Unmarshal(data, &userFile)
	if err != nil {
		return PrivateUser{}, err
	}

	key, err := x509.ParsePKCS1PrivateKey(userFile.PrivateKey)
	if err != nil {
		return PrivateUser{}, err
	}

	return PrivateUser{Id: userFile.Id, Address: userFile.Address, privateKey: key}, nil
}

// Generates user with specified address and private key. If no private key was provided generates a new one.
func GenUser(address string) (PrivateUser, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, userKeySize)
	if err != nil {
		return PrivateUser{}, err
	}

	pubString := base64.StdEncoding.EncodeToString(x509.MarshalPKCS1PublicKey(&privateKey.PublicKey))

	unsignedId := fmt.Sprintf("%s@%s", address, pubString)

	signature, err := sign(privateKey, []byte(unsignedId))
	if err != nil {
		return PrivateUser{}, err
	}

	fullId := fmt.Sprintf("%s#%s", unsignedId, base64.StdEncoding.EncodeToString(signature))

	return PrivateUser{Id: fullId, Address: address, privateKey: privateKey}, nil
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
	err = verify(publicKey, []byte(unsignedId), signature)
	if err != nil {
		return PublicUser{}, err
	}

	return PublicUser{Id: id, Address: address, publicKey: publicKey}, nil
}
