// Package auth resolves SSH/HTTPS credentials for a given remote URL the
// same way the native `git` CLI does, so users of this tool don't need to
// configure anything beyond what they already have set up for git itself.
//
// SSH resolution order (matches `ssh`/`git` behavior):
//  1. ~/.ssh/config — Host alias, User override, IdentityFile
//  2. Default identity files: id_ed25519, id_ecdsa, id_rsa, id_dsa
//  3. Running ssh-agent (SSH_AUTH_SOCK)
//
// HTTPS resolution: shells out to `git credential fill`, which delegates to
// whatever credential helper the user already has configured (osxkeychain,
// manager-core, libsecret, cache, store, ...). This avoids reimplementing
// credential storage and stays in sync with the user's existing git setup.
package auth

import (
	"bufio"
	"bytes"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5/plumbing/transport"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/kevinburke/ssh_config"
	gossh "golang.org/x/crypto/ssh"
)

// Manager resolves transport.AuthMethod values for git remote URLs.
type Manager struct {
	getSSHConfig func(host, key string) string
	homeDir      func() (string, error)
	readFile     func(string) ([]byte, error)
}

// New creates a Manager wired to real OS and SSH-config implementations.
func New() *Manager {
	return &Manager{
		getSSHConfig: sshConfigGet,
		homeDir:      os.UserHomeDir,
		readFile:     os.ReadFile,
	}
}

// Resolve picks the right auth method for a remote URL, following the same
// precedence native git uses for that protocol.
func (m *Manager) Resolve(remoteURL string) (transport.AuthMethod, error) {
	scheme, host, user, path := parseRemote(remoteURL)

	switch scheme {
	case "http", "https":
		return m.resolveHTTPAuth(scheme, host, user, path)
	case "ssh", "":
		return m.resolveSSHAuth(host, user)
	default:
		return nil, fmt.Errorf("unsupported remote scheme %q in %q", scheme, remoteURL)
	}
}

// parseRemote understands the three URL shapes git accepts:
//
//	https://host/path
//	ssh://user@host/path
//	user@host:path        (scp-like shorthand, e.g. git@github.com:org/repo.git)
func parseRemote(remote string) (scheme, host, user, path string) {
	if strings.HasPrefix(remote, "http://") || strings.HasPrefix(remote, "https://") {
		u, err := url.Parse(remote)
		if err != nil {
			return "", "", "", ""
		}
		user = ""
		if u.User != nil {
			user = u.User.Username()
		}
		return u.Scheme, u.Hostname(), user, u.Path
	}
	if strings.HasPrefix(remote, "ssh://") {
		u, err := url.Parse(remote)
		if err != nil {
			return "", "", "", ""
		}
		return "ssh", u.Hostname(), u.User.Username(), u.Path
	}
	if at := strings.Index(remote, "@"); at >= 0 && strings.Contains(remote[at:], ":") {
		user = remote[:at]
		rest := remote[at+1:]
		colon := strings.Index(rest, ":")
		host = rest[:colon]
		path = rest[colon+1:]
		return "ssh", host, user, path
	}
	return "", remote, "", ""
}

func (m *Manager) resolveSSHAuth(host, user string) (transport.AuthMethod, error) {
	if user == "" {
		user = "git"
	}

	home, err := m.homeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve home directory: %w", err)
	}

	// 1. ~/.ssh/config: Host alias may rename the user and point at a
	//    specific IdentityFile.
	if cfgUser := m.getSSHConfig(host, "User"); cfgUser != "" {
		user = cfgUser
	}

	// 2. Build the full list of candidate key paths. When ssh config names an
	//    explicit IdentityFile we use only that; otherwise we probe all the
	//    default names in the same priority order native ssh uses.
	var candidates []string
	if keyPath := m.getSSHConfig(host, "IdentityFile"); keyPath != "" && keyPath != "~/.ssh/identity" {
		candidates = append(candidates, expandHome(keyPath, home))
	} else {
		for _, name := range []string{"id_ed25519", "id_ecdsa", "id_rsa", "id_dsa"} {
			candidates = append(candidates, filepath.Join(home, ".ssh", name))
		}
	}

	// 3. Parse every readable key file into a signer. We collect them all so
	//    the SSH handshake can offer each one to the server — mirroring what
	//    native ssh does rather than stopping at the first parseable key.
	var fileSigners []gossh.Signer
	for _, path := range candidates {
		keyBytes, err := m.readFile(path)
		if err != nil {
			continue
		}
		pk, err := gitssh.NewPublicKeys(user, keyBytes, "")
		if err != nil {
			continue
		}
		fileSigners = append(fileSigners, pk.Signer)
	}

	// 4. Capture the agent callback lazily so it is evaluated at handshake
	//    time, not at resolve time, matching how native ssh uses the agent.
	var agentCallback func() ([]gossh.Signer, error)
	if os.Getenv("SSH_AUTH_SOCK") != "" {
		if agentAuth, err := gitssh.NewSSHAgentAuth(user); err == nil {
			agentCallback = agentAuth.Callback
		}
	}

	if len(fileSigners) == 0 && agentCallback == nil {
		return nil, fmt.Errorf(
			"no usable SSH key or agent found for host %q (checked ~/.ssh/config, default key files in ~/.ssh, and ssh-agent)",
			host,
		)
	}

	return &gitssh.PublicKeysCallback{
		User: user,
		Callback: func() ([]gossh.Signer, error) {
			all := make([]gossh.Signer, len(fileSigners))
			copy(all, fileSigners)
			if agentCallback != nil {
				if agentSigners, err := agentCallback(); err == nil {
					all = append(all, agentSigners...)
				}
			}
			return all, nil
		},
	}, nil
}

// resolveHTTPAuth shells out to `git credential fill`, so any credential
// helper already configured via `git config credential.helper` is honored
// without this tool needing to know how that helper stores secrets.
//
// Returns nil auth when no credentials are found, allowing anonymous public
// fetches to proceed (matching native git behavior).
func (m *Manager) resolveHTTPAuth(scheme, host, user, path string) (transport.AuthMethod, error) {
	// Build credential protocol input matching what native git sends.
	// Include path and username so credential helpers configured with
	// credential.useHttpPath or serving multiple credentials per host
	// can select the right credential.
	var input strings.Builder
	fmt.Fprintf(&input, "protocol=%s\n", scheme)
	fmt.Fprintf(&input, "host=%s\n", host)
	if path != "" {
		fmt.Fprintf(&input, "path=%s\n", path)
	}
	if user != "" {
		fmt.Fprintf(&input, "username=%s\n", user)
	}
	input.WriteString("\n")

	cmd := exec.Command("git", "credential", "fill")
	cmd.Stdin = strings.NewReader(input.String())
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	// When no credential helper has credentials for a public HTTPS remote,
	// git credential fill exits with error. Allow this case to return nil
	// so go-git can fetch anonymously, matching native git behavior.
	if err := cmd.Run(); err != nil {
		// If git credential fill fails, assume it's a public repo and allow
		// anonymous access by returning nil auth
		return nil, nil
	}

	creds := map[string]string{}
	scanner := bufio.NewScanner(&out)
	for scanner.Scan() {
		line := scanner.Text()
		if eq := strings.Index(line, "="); eq > 0 {
			creds[line[:eq]] = line[eq+1:]
		}
	}

	// No credentials returned means public repo - allow anonymous access
	if creds["username"] == "" {
		return nil, nil
	}

	return &githttp.BasicAuth{
		Username: creds["username"],
		Password: creds["password"],
	}, nil
}

func sshConfigGet(host, key string) string {
	if val, err := ssh_config.GetStrict(host, key); err == nil && val != "" {
		return val
	}
	return ssh_config.Get(host, key)
}

func expandHome(p, home string) string {
	if strings.HasPrefix(p, "~/") {
		return filepath.Join(home, p[2:])
	}
	return p
}
