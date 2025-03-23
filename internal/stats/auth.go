package stats

import (
	"bufio"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/terminal"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/0x0BSoD/rtop/pkg/logger"
)

func parsePemBlock(block *pem.Block) (interface{}, error) {
	switch block.Type {
	case "RSA PRIVATE KEY":
		return x509.ParsePKCS1PrivateKey(block.Bytes)
	case "EC PRIVATE KEY":
		return x509.ParseECPrivateKey(block.Bytes)
	case "DSA PRIVATE KEY":
		return ssh.ParseDSAPrivateKey(block.Bytes)
	default:
		return nil, fmt.Errorf("unsupported key type: %q", block.Type)
	}
}

func addDefaultKeys(auths []ssh.AuthMethod) []ssh.AuthMethod {
	homeDir := "~"
	homeDir, err := os.UserHomeDir()
	if err != nil {
		logger.Warn("Failed to get home directory")
		// Try to use just ~
	}
	defaultKeys := []string{
		filepath.Join(homeDir, "/.ssh/id_rsa"),
		filepath.Join(homeDir, "/.ssh/id_ecdsa"),
		filepath.Join(homeDir, "/.ssh/id_ed25519"),
		filepath.Join(homeDir, "/.ssh/id_dsa"),
	}

	for _, key := range defaultKeys {
		if _, err := os.Stat(key); err == nil {
			auths = addKeyAuth(auths, key)
		}
	}

	return auths
}

// Auth by key
func addKeyAuth(auths []ssh.AuthMethod, keypath string) []ssh.AuthMethod {
	if len(keypath) == 0 {
		return auths
	}

	// read the file
	pemBytes, err := os.ReadFile(keypath)
	if err != nil {
		logger.Error("Failed to read pem file %s", keypath)
		os.Exit(1)
	}

	// Attempt to parse as an unencrypted private key
	signer, err := ssh.ParsePrivateKey(pemBytes)
	if err == nil {
		return append(auths, ssh.PublicKeys(signer))
	}

	// If parsing fails, assume the key is encrypted and request a passphrase
	var passphraseMissingError *ssh.PassphraseMissingError
	if errors.As(err, &passphraseMissingError) {
		prompt := fmt.Sprintf("Enter passphrase for key '%s': ", keypath)
		passphrase, err := getpass(prompt)
		if err != nil {
			logger.Error("failed to get passphrase: %v", err)
			return auths
		}

		// Try parsing the key with the passphrase
		signer, err = ssh.ParsePrivateKeyWithPassphrase(pemBytes, []byte(passphrase))
		if err != nil {
			logger.Error("failed to decrypt private key: %v", err)
			return auths
		}

		return append(auths, ssh.PublicKeys(signer))

	}

	logger.Warn("invalid private key file: %v", err)
	return auths
}

// SSH Agent Auth
func getAgentAuth() (bool, ssh.AuthMethod) {
	if sock := os.Getenv("SSH_AUTH_SOCK"); len(sock) > 0 {
		if agconn, err := net.Dial("unix", sock); err == nil {
			ag := agent.NewClient(agconn)
			return true, ssh.PublicKeysCallback(ag.Signers)
		}
	}
	return false, nil
}

func tryAgentConnect(user, addr string) (*ssh.Client, error) {
	ok, auth := getAgentAuth()
	if !ok {
		return nil, nil
	}

	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{auth},
		// dummy check
		HostKeyCallback: func(string, net.Addr, ssh.PublicKey) error {
			return nil
		},
	}

	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SSH: %w", err)
	}
	return client, nil
}

// Password auth
func getpass(prompt string) (string, error) {
	tstate, err := terminal.GetState(0)
	if err != nil {
		return "", fmt.Errorf("failed to get terminal state: %w", err)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		quit := false
		for range sig {
			quit = true
			break
		}
		terminal.Restore(0, tstate)
		if quit {
			fmt.Println()
			os.Exit(2)
		}
	}()
	defer func() {
		signal.Stop(sig)
		close(sig)
	}()

	f := bufio.NewWriter(os.Stdout)
	f.Write([]byte(prompt))
	f.Flush()

	passbytes, err := terminal.ReadPassword(0)

	f.Write([]byte("\n"))
	f.Flush()

	return string(passbytes), nil
}

func addPasswordAuth(user, addr string, auths []ssh.AuthMethod) []ssh.AuthMethod {
	if terminal.IsTerminal(0) == false {
		return auths
	}
	host := addr
	if i := strings.LastIndex(host, ":"); i != -1 {
		host = host[:i]
	}
	prompt := fmt.Sprintf("%s@%s's password: ", user, host)
	passwordCallback := func() (string, error) {
		return getpass(prompt)
	}
	return append(auths, ssh.PasswordCallback(passwordCallback))
}
