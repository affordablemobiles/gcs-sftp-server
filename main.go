package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"

	_ "github.com/GoogleCloudPlatform/berglas/pkg/auto"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"google.golang.org/api/option"

	gsftp "github.com/a1comms/gcs-sftp-server/handler"
)

var (
	SFTP_USERNAME             string = os.Getenv("SFTP_USERNAME")
	SFTP_PASSWORD             string = os.Getenv("SFTP_PASSWORD")
	SFTP_PORT                 string = mustGetenv("SFTP_PORT")
	SFTP_SERVER_KEY_PATH      string = mustGetenv("SFTP_SERVER_KEY_PATH")
	SFTP_AUTHORIZED_KEYS_FILE string = os.Getenv("SFTP_AUTHORIZED_KEYS_FILE")
	GCS_CREDENTIALS_FILE      string = os.Getenv("GCS_CREDENTIALS_FILE")
	GCS_BUCKET                string = mustGetenv("GCS_BUCKET")
	SFTP_TEMP_DIR             string = os.Getenv("SFTP_TEMP_DIR")
)

func main() {
	// An SSH server is represented by a ServerConfig, which holds
	// certificate details and handles authentication of ServerConns.
	config := &ssh.ServerConfig{
		NoClientAuth:  false,
		ServerVersion: "SSH-2.0-GCS-SFTP",
		AuthLogCallback: func(conn ssh.ConnMetadata, method string, err error) {
			if err != nil {
				log.Printf("Failed %s for user %s from %s ssh2", method, conn.User(), conn.RemoteAddr())
			} else {
				log.Printf("Accepted %s for user %s from %s ssh2", method, conn.User(), conn.RemoteAddr())
			}
		},
	}

	privateBytes, err := os.ReadFile(SFTP_SERVER_KEY_PATH)
	if err != nil {
		log.Fatalf("Failed to load private key: %s", err)
	}

	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		log.Fatalf("Failed to parse private key: %s", err)
	}

	if SFTP_USERNAME != "" && SFTP_PASSWORD != "" {
		config.PasswordCallback = func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			if c.User() == SFTP_USERNAME && string(pass) == SFTP_PASSWORD {
				return nil, nil
			}
			return nil, fmt.Errorf("password rejected for %q", c.User())
		}
	}

	processPublicKeyAuth(config)

	config.AddHostKey(private)

	// Once a ServerConfig has been configured, connections can be
	// accepted.
	listener, err := net.Listen("tcp", "0.0.0.0:"+SFTP_PORT)
	if err != nil {
		log.Fatalf("failed to listen for connection: %s", err)
	}
	log.Printf("Listening on %v\n", listener.Addr())

	for {
		nConn, err := listener.Accept()
		if err != nil {
			log.Printf("ERROR: failed to accept incoming connection", err)
		}

		go HandleConn(nConn, config)
	}
}

func HandleConn(nConn net.Conn, config *ssh.ServerConfig) {
	// Before use, a handshake must be performed on the incoming net.Conn.
	sconn, chans, reqs, err := ssh.NewServerConn(nConn, config)
	if err != nil {
		log.Printf("failed to handshake: %s", err)
		return
	}
	log.Printf("login detected: %s", sconn.User())

	// The incoming Request channel must be serviced.
	go ssh.DiscardRequests(reqs)

	// Service the incoming Channel channel.
	for newChannel := range chans {
		// Channels have a type, depending on the application level
		// protocol intended. In the case of an SFTP session, this is "subsystem"
		// with a payload string of "<length=4>sftp"
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}
		channel, requests, err := newChannel.Accept()
		if err != nil {
			log.Printf("could not accept channel: %s", err)
			return
		}

		// Sessions have out-of-band requests such as "shell",
		// "pty-req" and "env".  Here we handle only the
		// "subsystem" request.
		go func(in <-chan *ssh.Request) {
			for req := range in {
				ok := false
				switch req.Type {
				case "subsystem":
					if string(req.Payload[4:]) == "sftp" {
						ok = true
					}
				}
				req.Reply(ok, nil)
			}
		}(requests)

		ctx := context.Background()

		opts := []option.ClientOption{}
		if GCS_CREDENTIALS_FILE != "" {
			opts = append(opts, option.WithCredentialsFile(GCS_CREDENTIALS_FILE))
		}

		root, err := gsftp.GoogleCloudStorageHandler(ctx, GCS_BUCKET, SFTP_TEMP_DIR, opts...)
		if err != nil {
			log.Fatalf("GCS Init Failed: %s", err)
		}

		server := sftp.NewRequestServer(channel, *root)
		if err := server.Serve(); err == io.EOF {
			server.Close()

			log.Printf("sftp client exited session.")
		} else if err != nil {
			log.Printf("sftp server completed with error: %s", err)
		}
	}
}
