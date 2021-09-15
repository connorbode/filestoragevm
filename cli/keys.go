package main

import (
	"io/ioutil"
	"strings"
	"os"
	"fmt"
	"github.com/ava-labs/avalanchego/utils/crypto"
	"github.com/ava-labs/avalanchego/utils/formatting"
)

func createAddress() {
	factory := crypto.FactorySECP256K1R{}
        skIntf, _ := factory.NewPrivateKey()
        sk := skIntf.(*crypto.PrivateKeySECP256K1R)
        privKeyStr, _ := formatting.EncodeWithChecksum(formatting.CB58, sk.Bytes())
	fmt.Println(privKeyStr)
}

func signMessage() {
	var privKeyCB58 string
	var messageCB58 string
	var messageBytes []byte
	var privKeyBytes []byte
	var privKey crypto.PrivateKey
	var sig []byte;
	bytes, _ := ioutil.ReadFile(os.Args[2])
	file_content := string(bytes)
	lines := strings.Split(file_content, "\n")
	for i, line := range lines {
		if i == 0 {
			privKeyCB58 = line
		} else {
			messageCB58 = line
		}
	}
	factory := crypto.FactorySECP256K1R{}
	messageBytes, _ = formatting.Decode(formatting.CB58, messageCB58)
	privKeyBytes, _ = formatting.Decode(formatting.CB58, privKeyCB58)
	privKey, _ = factory.ToPrivateKey(privKeyBytes)
	sig, _ = privKey.Sign(messageBytes)
	encoded, _ := formatting.EncodeWithChecksum(formatting.CB58, sig)
	write_message := []byte(encoded)
	_ = ioutil.WriteFile(os.Args[2], write_message, 0644)
}

func main() {
	var cmd string
	cmd = os.Args[1]
	if cmd == "create_address" {
		createAddress()
	} else if cmd == "sign" {
		signMessage()
	}
}
