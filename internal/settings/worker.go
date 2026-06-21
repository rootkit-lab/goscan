package settings

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
)

// RemoteWorker describes a remote scan node reachable via SSH.
type RemoteWorker struct {
	ID            string `yaml:"id"`
	Name          string `yaml:"name"`
	Host          string `yaml:"host"`
	Port          int    `yaml:"port,omitempty"`
	User          string `yaml:"user"`
	AuthType      string `yaml:"auth_type"` // password | key | ppk
	Password      string `yaml:"password,omitempty"`
	KeyPath       string `yaml:"key_path,omitempty"`
	KeyPassphrase string `yaml:"key_passphrase,omitempty"`
	ExecMode      string `yaml:"exec_mode"` // ssh | http
	APIPort       int    `yaml:"api_port,omitempty"`
	APIToken      string `yaml:"api_token,omitempty"`
	Enabled       bool   `yaml:"enabled,omitempty"`
}

const (
	AuthPassword = "password"
	AuthKey      = "key"
	AuthPPK      = "ppk"
	ExecSSH      = "ssh"
	ExecHTTP     = "http"
)

func NewWorkerID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return "worker-" + hex.EncodeToString(b)
}

func (w RemoteWorker) Normalized() RemoteWorker {
	w.ID = strings.TrimSpace(w.ID)
	w.Name = strings.TrimSpace(w.Name)
	w.Host = strings.TrimSpace(w.Host)
	w.User = strings.TrimSpace(w.User)
	w.AuthType = strings.TrimSpace(w.AuthType)
	w.KeyPath = strings.TrimSpace(w.KeyPath)
	if w.Port <= 0 {
		w.Port = 22
	}
	if w.AuthType == "" {
		w.AuthType = AuthPassword
	}
	if w.ExecMode == "" {
		w.ExecMode = ExecSSH
	}
	if w.APIPort <= 0 {
		w.APIPort = 9090
	}
	if w.ID == "" {
		w.ID = NewWorkerID()
	}
	if w.Name == "" {
		w.Name = w.Host
	}
	return w
}

func (w RemoteWorker) Address() string {
	w = w.Normalized()
	return w.Host
}
