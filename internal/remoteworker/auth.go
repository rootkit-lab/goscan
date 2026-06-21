package remoteworker

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"goscan/internal/settings"
	"golang.org/x/crypto/ssh"
)

func signerFor(w Config) (ssh.Signer, error) {
	switch w.AuthType {
	case settings.AuthPassword:
		return nil, nil
	case settings.AuthKey:
		return loadPEMKey(w.KeyPath, w.KeyPassphrase)
	case settings.AuthPPK:
		return loadPPK(w.KeyPath, w.KeyPassphrase)
	default:
		return nil, fmt.Errorf("tipo de auth desconhecido: %s", w.AuthType)
	}
}

func loadPEMKey(path, passphrase string) (ssh.Signer, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("caminho da chave em falta")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if passphrase != "" {
		key, err := ssh.ParsePrivateKeyWithPassphrase(raw, []byte(passphrase))
		if err != nil {
			return nil, err
		}
		return key, nil
	}
	key, err := ssh.ParsePrivateKey(raw)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func loadPPK(path, passphrase string) (ssh.Signer, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("caminho PPK em falta")
	}
	if _, err := exec.LookPath("puttygen"); err != nil {
		return nil, fmt.Errorf("PPK requer puttygen instalado (sudo apt install putty-tools)")
	}
	tmp, err := os.CreateTemp("", "goscan-ppk-*.pem")
	if err != nil {
		return nil, err
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()
	defer os.Remove(tmpPath)

	args := []string{path, "-O", "private-openssh", "-o", tmpPath}
	if passphrase != "" {
		args = append(args, "-passphrase", passphrase)
	}
	cmd := exec.Command("puttygen", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("puttygen: %v — %s", err, strings.TrimSpace(string(out)))
	}
	return loadPEMKey(tmpPath, "")
}

func sshClientConfig(w Config) (*ssh.ClientConfig, error) {
	w = Config{RemoteWorker: w.Normalized()}
	cfg := &ssh.ClientConfig{
		User:            w.User,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	if w.AuthType == settings.AuthPassword {
		if strings.TrimSpace(w.Password) == "" {
			return nil, fmt.Errorf("password em falta")
		}
		cfg.Auth = []ssh.AuthMethod{ssh.Password(w.Password)}
		return cfg, nil
	}
	signer, err := signerFor(w)
	if err != nil {
		return nil, err
	}
	if signer == nil {
		return nil, fmt.Errorf("chave inválida")
	}
	cfg.Auth = []ssh.AuthMethod{ssh.PublicKeys(signer)}
	return cfg, nil
}

func dial(w Config) (*ssh.Client, error) {
	cfg, err := sshClientConfig(w)
	if err != nil {
		return nil, err
	}
	addr := fmt.Sprintf("%s:%d", w.Host, w.Port)
	return ssh.Dial("tcp", addr, cfg)
}
