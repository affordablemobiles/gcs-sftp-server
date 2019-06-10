package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"golang.org/x/crypto/ssh"
)

func mustGetenv(k string) string {
	v := os.Getenv(k)
	if v == "" {
		log.Fatalf("%s environment variable not set.", k)
	}
	return v
}

func processPublicKeyAuth(config *ssh.ServerConfig) {
	if SFTP_AUTHORIZED_KEYS_FILE == "" {
		return
	}

	authorizedKeysBytes, err := ioutil.ReadFile(SFTP_AUTHORIZED_KEYS_FILE)
	if err != nil {
		log.Fatalf("Failed to load authorized keys file: %s", err)
	}

	authorizedKeysArray := []ssh.PublicKey{}
	for {
		out, _, _, rest, err := ssh.ParseAuthorizedKey(authorizedKeysBytes)
		if err != nil {
			if err.Error() == "ssh: no key found" {
				break
			} else {
				log.Fatalf("Failed to parse authorized key: %s", err)
			}
		} else {
			authorizedKeysArray = append(authorizedKeysArray, out)
		}
		authorizedKeysBytes = rest
	}

	config.PublicKeyCallback = func(conn ssh.ConnMetadata, auth ssh.PublicKey) (*ssh.Permissions, error) {
		for _, pubKey := range authorizedKeysArray {
			if string(pubKey.Marshal()) == string(auth.Marshal()) {
				return &ssh.Permissions{
					// Record the public key used for authentication.
					Extensions: map[string]string{
						"pubkey-fp": ssh.FingerprintSHA256(auth),
					},
				}, nil
			}
		}

		return nil, fmt.Errorf("unknown public key for %q", conn.User())
	}
}
