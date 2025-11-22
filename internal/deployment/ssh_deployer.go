package deployment

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/ssh"
)

const (
	// SSHConnectionTimeout is the timeout for establishing SSH connections
	SSHConnectionTimeout = 30 * time.Second
)

// SSHDeployer handles deployment via SSH/SCP, managing secure file transfers
// and remote command execution for deploying JSON files to web servers.
type SSHDeployer struct {
	keyPath   string
	deployURL string
	client    *ssh.Client
	connected bool
}

// NewSSHDeployer creates a new SSH deployer
func NewSSHDeployer(deployURL string) *SSHDeployer {
	return &SSHDeployer{
		keyPath:   "deploy.pem",
		deployURL: deployURL,
	}
}

// parseDeployURL parses a deploy URL in format: user@host:path
func (d *SSHDeployer) parseDeployURL() (user, host, remotePath string, err error) {
	if d.deployURL == "" {
		return "", "", "", fmt.Errorf("deploy URL is empty")
	}

	// Split by @ to get user and host:path
	parts := strings.SplitN(d.deployURL, "@", 2)
	if len(parts) != 2 {
		return "", "", "", fmt.Errorf("invalid deploy URL format: expected user@host:path")
	}

	user = parts[0]
	hostPath := parts[1]

	// Split by : to get host and path
	hostParts := strings.SplitN(hostPath, ":", 2)
	if len(hostParts) != 2 {
		return "", "", "", fmt.Errorf("invalid deploy URL format: expected user@host:path")
	}

	host = hostParts[0]
	remotePath = hostParts[1]

	return user, host, remotePath, nil
}

// Connect establishes SSH connection
func (d *SSHDeployer) Connect() error {
	if d.connected {
		return nil
	}

	user, host, _, err := d.parseDeployURL()
	if err != nil {
		return fmt.Errorf("failed to parse deploy URL: %w", err)
	}

	// Read private key
	keyData, err := os.ReadFile(d.keyPath)
	if err != nil {
		return fmt.Errorf("failed to read SSH key file %s: %w", d.keyPath, err)
	}

	// Parse private key
	signer, err := ssh.ParsePrivateKey(keyData)
	if err != nil {
		return fmt.Errorf("failed to parse SSH private key: %w", err)
	}

	// Create SSH client config
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // In production, use proper host key verification
		Timeout:         SSHConnectionTimeout,
	}

	// Connect to SSH server
	d.client, err = ssh.Dial("tcp", net.JoinHostPort(host, "22"), config)
	if err != nil {
		return fmt.Errorf("failed to connect to SSH server %s: %w", host, err)
	}

	d.connected = true
	log.Info().
		Str("host", host).
		Str("user", user).
		Msg("Successfully connected to SSH server")

	return nil
}

// Disconnect closes SSH connection
func (d *SSHDeployer) Disconnect() error {
	if d.client != nil {
		err := d.client.Close()
		d.connected = false
		d.client = nil
		return err
	}
	return nil
}

// DeployData uploads data from an io.Reader via SCP
func (d *SSHDeployer) DeployData(data io.Reader, size int64, filename string) error {
	if !d.connected {
		if err := d.Connect(); err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
	}

	_, _, remotePath, err := d.parseDeployURL()
	if err != nil {
		return fmt.Errorf("failed to parse deploy URL: %w", err)
	}

	// Create SCP session
	session, err := d.client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	// Construct remote file path
	remoteFilePath := filepath.Join(remotePath, filename)

	// Create SCP command
	scpCmd := fmt.Sprintf("scp -t %s", remoteFilePath)

	// Get stdin for SCP
	stdin, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	// Start SCP session
	err = session.Start(scpCmd)
	if err != nil {
		return fmt.Errorf("failed to start SCP session: %w", err)
	}

	// Send file header
	header := fmt.Sprintf("C0644 %d %s\n", size, filename)
	_, err = stdin.Write([]byte(header))
	if err != nil {
		return fmt.Errorf("failed to write SCP header: %w", err)
	}

	// Copy data content
	_, err = io.Copy(stdin, data)
	if err != nil {
		return fmt.Errorf("failed to copy data content: %w", err)
	}

	// Send end marker
	_, err = stdin.Write([]byte{0})
	if err != nil {
		return fmt.Errorf("failed to write SCP end marker: %w", err)
	}

	// Close stdin and wait for completion
	stdin.Close()
	err = session.Wait()
	if err != nil {
		return fmt.Errorf("SCP session failed: %w", err)
	}

	log.Info().
		Str("remote_path", remoteFilePath).
		Int64("size", size).
		Msg("Successfully deployed data via SCP")

	return nil
}
