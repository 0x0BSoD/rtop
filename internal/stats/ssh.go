package stats

import (
	"bytes"
	"fmt"
	"github.com/0x0BSoD/rtop/pkg/logger"
	"golang.org/x/crypto/ssh"
	"log"
	"net"
	"os"
)

func SshConnect(user, addr, keyPath string) (*ssh.Client, error) {
	logger.Info("Establishing SSH connection to %s@%s", user, addr)
	auths := make([]ssh.AuthMethod, 0)

	// Try connecting via agent first
	logger.Info("SSH Agent checking")
	client, err := tryAgentConnect(user, addr)
	if err != nil {
		logger.Error("SSH connection with agent failed: %v", err)
		return nil, fmt.Errorf("filed to use agent: %w", err)
	}
	if client != nil {
		logger.Info("SSH connection with agent established successfully")
		return client, nil
	}

	// If that failed try with the key and password methods
	if len(keyPath) > 0 {
		auths = addKeyAuth(auths, keyPath) // User-specified key
	} else {
		auths = addDefaultKeys(auths) // Check ~/.ssh/id_* files
	}
	auths = addPasswordAuth(user, addr, auths)

	config := &ssh.ClientConfig{
		User: user,
		Auth: auths,
		HostKeyCallback: func(string, net.Addr, ssh.PublicKey) error {
			return nil
		},
	}

	client, err = ssh.Dial("tcp", addr, config)
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}

	logger.Info("SSH connection established successfully")
	return client, nil
}

func runCommand(client *ssh.Client, command string) (string, error) {
	logger.Debug("Creating new SSH session")
	session, err := client.NewSession()
	if err != nil {
		logger.Error("Failed to create SSH session: %v", err)
		return "", fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	logger.Debug("SSH session created successfully")

	logger.Debug("Executing command: %s", command)
	var buf bytes.Buffer
	session.Stdout = &buf
	if err := session.Run(command); err != nil {
		logger.Error("Command execution failed: %s - %v", command, err)
		return "", fmt.Errorf("failed to run command '%s': %w", command, err)
	}

	output := buf.String()
	logger.Debug("Command executed successfully, output length: %d bytes", len(output))
	return output, nil
}
