package main

import (
	"crypto/subtle"
	"fmt"
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

	authorizedKeysFileBytes, err := os.ReadFile(SFTP_AUTHORIZED_KEYS_FILE)
	if err != nil {
		log.Fatalf("Failed to load authorized keys file: %s", err)
	}

	// Store keys as a slice of byte slices for constant-time iteration.
	var authorizedKeys [][]byte

	for len(authorizedKeysFileBytes) > 0 {
		pubKey, _, options, rest, err := ssh.ParseAuthorizedKey(authorizedKeysFileBytes)
		if err != nil {
			if err.Error() == "ssh: no key found" {
				break
			}

			log.Printf("Warning: Failed to parse an authorized key line: %v", err)
			authorizedKeysFileBytes = rest
			continue
		}

		if len(options) > 0 {
			log.Printf("Warning: Key options %v detected but ignored for key %s. Restrictions will not be enforced.", options, pubKey.Type())
		}

		authorizedKeys = append(authorizedKeys, pubKey.Marshal())
		authorizedKeysFileBytes = rest
	}

	config.PublicKeyCallback = func(conn ssh.ConnMetadata, auth ssh.PublicKey) (*ssh.Permissions, error) {
		authBytes := auth.Marshal()
		matched := 0

		// Iterate over ALL authorized keys.
		// Do not return early, or you leak information about which key matched (or didn't) via timing.
		for _, ak := range authorizedKeys {
			// ConstantTimeCompare returns 1 if the two slices are equal, 0 otherwise.
			// It takes time proportional to the length of the slices.
			if subtle.ConstantTimeCompare(ak, authBytes) == 1 {
				matched = 1
			}
		}

		if matched == 1 {
			return &ssh.Permissions{
				// Record the public key used for authentication.
				Extensions: map[string]string{
					"pubkey-fp": ssh.FingerprintSHA256(auth),
				},
			}, nil
		}

		return nil, fmt.Errorf("unknown public key for %q", conn.User())
	}
}
