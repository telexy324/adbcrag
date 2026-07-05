package client

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"ops-kb-rag/backend/internal/dto"

	"golang.org/x/crypto/ssh"
)

type SSHConfig struct {
	Host       string
	Port       int
	Username   string
	Password   string
	PrivateKey string
	Passphrase string
	AuthType   string
	Timeout    time.Duration
}

type SSHLogQuery struct {
	SSHConfig
	LogPath       string
	PathAllowlist []string
	Keyword       string
	LogLevel      string
	Limit         int
}

type SSHLogClient interface {
	Test(ctx context.Context, cfg SSHConfig) error
	ReadLogs(ctx context.Context, query SSHLogQuery) ([]dto.LogItem, error)
}

type SSHCommandLogClient struct{}

func NewSSHLogClient() SSHLogClient {
	return &SSHCommandLogClient{}
}

func (c *SSHCommandLogClient) Test(ctx context.Context, cfg SSHConfig) error {
	client, err := dialSSH(ctx, cfg)
	if err != nil {
		return err
	}
	return client.Close()
}

func (c *SSHCommandLogClient) ReadLogs(ctx context.Context, query SSHLogQuery) ([]dto.LogItem, error) {
	if query.Limit <= 0 {
		query.Limit = 100
	}
	if err := validateRemotePath(query.LogPath, query.PathAllowlist); err != nil {
		return nil, err
	}
	client, err := dialSSH(ctx, query.SSHConfig)
	if err != nil {
		return nil, err
	}
	defer client.Close()
	session, err := client.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()
	command := fmt.Sprintf("tail -n %d -- %s", query.Limit, shellQuote(query.LogPath))
	output, err := session.Output(command)
	if err != nil {
		return nil, err
	}
	return parseLogLines(string(output), query.Keyword, query.LogLevel, query.LogPath), nil
}

func dialSSH(ctx context.Context, cfg SSHConfig) (*ssh.Client, error) {
	if cfg.Port == 0 {
		cfg.Port = 22
	}
	auth, err := sshAuthMethod(cfg)
	if err != nil {
		return nil, err
	}
	sshConfig := &ssh.ClientConfig{
		User:            cfg.Username,
		Auth:            []ssh.AuthMethod{auth},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         cfg.Timeout,
	}
	address := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, err
	}
	c, chans, reqs, err := ssh.NewClientConn(conn, address, sshConfig)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	return ssh.NewClient(c, chans, reqs), nil
}

func sshAuthMethod(cfg SSHConfig) (ssh.AuthMethod, error) {
	if cfg.AuthType == "private_key" {
		var signer ssh.Signer
		var err error
		if cfg.Passphrase != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(cfg.PrivateKey), []byte(cfg.Passphrase))
		} else {
			signer, err = ssh.ParsePrivateKey([]byte(cfg.PrivateKey))
		}
		if err != nil {
			return nil, err
		}
		return ssh.PublicKeys(signer), nil
	}
	return ssh.Password(cfg.Password), nil
}

func validateRemotePath(path string, allowlist []string) error {
	if path == "" || !strings.HasPrefix(path, "/") {
		return fmt.Errorf("logPath must be an absolute path")
	}
	clean := filepath.Clean(path)
	if clean != path || strings.Contains(path, "\x00") {
		return fmt.Errorf("invalid logPath")
	}
	for _, forbidden := range []string{"/etc", "/root", "/home"} {
		if path == forbidden || strings.HasPrefix(path, forbidden+"/") {
			return fmt.Errorf("logPath is in a forbidden directory")
		}
	}
	for _, prefix := range allowlist {
		prefix = filepath.Clean(prefix)
		if prefix != "/" && (path == prefix || strings.HasPrefix(path, strings.TrimRight(prefix, "/")+"/")) {
			return nil
		}
	}
	return fmt.Errorf("logPath is outside pathAllowlist")
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func parseLogLines(output, keyword, level, source string) []dto.LogItem {
	items := []dto.LogItem{}
	scanner := bufio.NewScanner(bytes.NewBufferString(output))
	for scanner.Scan() {
		line := scanner.Text()
		if keyword != "" && !strings.Contains(strings.ToLower(line), strings.ToLower(keyword)) {
			continue
		}
		if level != "" && !strings.Contains(strings.ToLower(line), strings.ToLower(level)) {
			continue
		}
		items = append(items, dto.LogItem{Level: detectLevel(line), Message: line, Source: source, Raw: line})
	}
	return items
}

func detectLevel(line string) string {
	upper := strings.ToUpper(line)
	for _, level := range []string{"ERROR", "WARN", "INFO", "DEBUG", "TRACE", "FATAL"} {
		if strings.Contains(upper, level) {
			return level
		}
	}
	return ""
}
