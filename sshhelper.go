/*

rtop - the remote system monitoring utility

Copyright (c) 2015-17 RapidLoop

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package main

import (
	"bufio"
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/terminal"
)

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

// ref golang.org/x/crypto/ssh/keys.go#ParseRawPrivateKey.
func ParsePemBlock(block *pem.Block) (interface{}, error) {
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

func addKeyAuth(auths []ssh.AuthMethod, keypath string) []ssh.AuthMethod {
	if len(keypath) == 0 {
		return auths
	}

	// read the file
	pemBytes, err := os.ReadFile(keypath)
	if err != nil {
		log.Print(err)
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
			log.Printf("failed to get passphrase: %v", err)
			return auths
		}

		// Try parsing the key with the passphrase
		signer, err = ssh.ParsePrivateKeyWithPassphrase(pemBytes, []byte(passphrase))
		if err != nil {
			log.Printf("failed to decrypt private key: %v", err)
			return auths
		}

		return append(auths, ssh.PublicKeys(signer))

	}

	log.Printf("invalid private key file: %v", err)
	return auths
}

func getAgentAuth() (bool, ssh.AuthMethod) {
	if sock := os.Getenv("SSH_AUTH_SOCK"); len(sock) > 0 {
		if agconn, err := net.Dial("unix", sock); err == nil {
			ag := agent.NewClient(agconn)
			return true, ssh.PublicKeysCallback(ag.Signers)
		}
	}
	return false, nil
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

func addDefaultKeys(auths []ssh.AuthMethod) []ssh.AuthMethod {
	homeDir := "~"
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Print(err)
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

func sshConnect(user, addr, keyPath string) (*ssh.Client, error) {
	Info("Establishing SSH connection to %s@%s", user, addr)
	auths := make([]ssh.AuthMethod, 0)

	// Try connecting via agent first
	Info("SSH Agent checking")
	client, err := tryAgentConnect(user, addr)
	if err != nil {
		Error("SSH connection with agent failed: %v", err)
		return nil, fmt.Errorf("filed to use agent: %w", err)
	}
	if client != nil {
		Info("SSH connection with agent established successfully")
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

	Info("SSH connection established successfully")
	return client, nil
}

func runCommand(client *ssh.Client, command string) (string, error) {
	Debug("Creating new SSH session")
	session, err := client.NewSession()
	if err != nil {
		Error("Failed to create SSH session: %v", err)
		return "", fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()
	Debug("SSH session created successfully")

	Debug("Executing command: %s", command)
	var buf bytes.Buffer
	session.Stdout = &buf
	if err := session.Run(command); err != nil {
		Error("Command execution failed: %s - %v", command, err)
		return "", fmt.Errorf("failed to run command '%s': %w", command, err)
	}

	output := buf.String()
	Debug("Command executed successfully, output length: %d bytes", len(output))
	return output, nil
}
