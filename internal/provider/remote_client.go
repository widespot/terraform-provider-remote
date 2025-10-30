package provider

import (
	"bytes"
	"fmt"
	"strings"
	"sync"

	"golang.org/x/crypto/ssh"
)

type Error struct {
	cmd    string
	err    error
	stderr []byte
}

func (e Error) Error() string {
	stderr := strings.TrimRight(string(e.stderr), "\n")
	return fmt.Sprintf("`%s`\n  %s\n  %s", e.cmd, e.err, stderr)
}

func run(s *ssh.Session, cmd string) error {
	var b bytes.Buffer
	s.Stderr = &b
	err := s.Run(cmd)

	if err != nil {
		return Error{
			cmd:    cmd,
			err:    err,
			stderr: b.Bytes(),
		}
	}
	return nil
}

// SessionPool manages a pool of SSH sessions with a maximum concurrency limit
type SessionPool struct {
	sshClient *ssh.Client
	semaphore chan struct{} // Used as semaphore to limit concurrent sessions
	mu        sync.Mutex
	closed    bool
}

// NewSessionPool creates a new session pool
func NewSessionPool(client *ssh.Client, maxSize int) *SessionPool {
	if maxSize <= 0 {
		maxSize = 10 // Default to SSHD's default MaxSessions
	}
	return &SessionPool{
		sshClient: client,
		semaphore: make(chan struct{}, maxSize),
		closed:    false,
	}
}

// Get retrieves a session from the pool or creates a new one
// This will block if maxSize sessions are already active
func (p *SessionPool) Get() (*ssh.Session, error) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil, fmt.Errorf("session pool is closed")
	}
	p.mu.Unlock()

	// Acquire a slot (will block if pool is full)
	p.semaphore <- struct{}{}

	// Create a new session
	session, err := p.sshClient.NewSession()
	if err != nil {
		// Release the slot if session creation failed
		<-p.semaphore
		return nil, err
	}

	return session, nil
}

// Put closes the session and releases a slot in the pool
func (p *SessionPool) Put(session *ssh.Session) {
	if session == nil {
		return
	}

	// Close the session (sessions cannot be reused)
	session.Close()

	// Release the slot
	select {
	case <-p.semaphore:
		// Slot released
	default:
		// This shouldn't happen, but handle gracefully
	}
}

// Close closes all sessions in the pool
func (p *SessionPool) Close() {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	p.mu.Unlock()

	// No need to close the semaphore channel or drain it
	// Any blocked Get() calls will be handled by the closed check
}

type RemoteClient struct {
	sshClient   *ssh.Client
	sessionPool *SessionPool
	sudo        bool
}

// NewSession gets a session from the pool
func (c *RemoteClient) NewSession() (*ssh.Session, error) {
	return c.sessionPool.Get()
}

// ReleaseSession returns a session to the pool
func (c *RemoteClient) ReleaseSession(session *ssh.Session) {
	c.sessionPool.Put(session)
}

func (c *RemoteClient) WriteFile(content string, path string, sudo bool, ensureDir bool) error {
	return c.WriteFileShell(content, path, sudo, ensureDir)
}

func (c *RemoteClient) WriteFileShell(content string, path string, sudo bool, ensureDir bool) error {
	session, err := c.NewSession()
	if err != nil {
		return err
	}
	defer c.ReleaseSession(session)

	stdin, err := session.StdinPipe()
	if err != nil {
		return err
	}

	go func() {
		stdin.Write([]byte(content))
		stdin.Close()
	}()

	cmd := fmt.Sprintf("tee %s", path)
	if c.sudo {
		cmd = fmt.Sprintf("sudo %s", cmd)
	}
	cmd = fmt.Sprintf("cat /dev/stdin | %s", cmd)
	if ensureDir {
		dirPathElements := strings.Split(path, "/")
		dirPathElements = dirPathElements[:len(dirPathElements)-1]
		dirPath := strings.Join(dirPathElements, "/")
		cmd = fmt.Sprintf("mkdir -p %s && %s", dirPath, cmd)
	}
	return run(session, cmd)
}

func (c *RemoteClient) ChmodFile(path string, permissions string, sudo bool) error {
	session, err := c.NewSession()
	if err != nil {
		return err
	}
	defer c.ReleaseSession(session)

	cmd := fmt.Sprintf("chmod %s %s", permissions, path)
	if c.sudo {
		cmd = fmt.Sprintf("sudo %s", cmd)
	}
	return run(session, cmd)
}

func (c *RemoteClient) CreateDir(path string, sudo bool) error {
	session, err := c.NewSession()
	if err != nil {
		return err
	}
	defer c.ReleaseSession(session)

	cmd := fmt.Sprintf("mkdir -p %s", path)
	if c.sudo {
		cmd = fmt.Sprintf("sudo %s", cmd)
	}
	return run(session, cmd)
}

func (c *RemoteClient) ChgrpFile(path string, group string, sudo bool) error {
	session, err := c.NewSession()
	if err != nil {
		return err
	}
	defer c.ReleaseSession(session)

	cmd := fmt.Sprintf("chgrp %s %s", group, path)
	if c.sudo {
		cmd = fmt.Sprintf("sudo %s", cmd)
	}

	return run(session, cmd)
}

func (c *RemoteClient) ChownFile(path string, owner string, sudo bool) error {
	session, err := c.NewSession()
	if err != nil {
		return err
	}
	defer c.ReleaseSession(session)

	cmd := fmt.Sprintf("chown %s %s", owner, path)
	if c.sudo {
		cmd = fmt.Sprintf("sudo %s", cmd)
	}
	return run(session, cmd)
}

func (c *RemoteClient) FileExists(path string, sudo bool) (bool, error) {
	session, err := c.NewSession()
	if err != nil {
		return false, err
	}
	defer c.ReleaseSession(session)

	cmd := fmt.Sprintf("test -f %s", path)
	if c.sudo {
		cmd = fmt.Sprintf("sudo %s", cmd)
	}
	err = run(session, cmd)

	if err != nil {
		session2, err := c.NewSession()
		if err != nil {
			return false, err
		}
		defer c.ReleaseSession(session2)

		cmd := fmt.Sprintf("test ! -f %s", path)
		if c.sudo {
			cmd = fmt.Sprintf("sudo %s", cmd)
		}
		return false, session2.Run(cmd)
	}

	return true, nil
}

func (c *RemoteClient) ReadFile(path string, sudo bool) (string, bool, error) {
	return c.ReadFileShell(path, sudo)
}

func (c *RemoteClient) dirExists(path string) (bool, error) {
	session, err := c.NewSession()
	if err != nil {
		return false, err
	}
	defer c.ReleaseSession(session)

	cmd := fmt.Sprintf("[ -d \"%s\" ] && exit 0 || exit 1 ", path)
	_, err = session.CombinedOutput(cmd)
	if err != nil {
		return false, nil
	}

	return true, nil
}

func (c *RemoteClient) ReadFileShell(path string, sudo bool) (string, bool, error) {
	session, err := c.NewSession()
	if err != nil {
		return "", false, err
	}
	defer c.ReleaseSession(session)

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	cmd := fmt.Sprintf("cat %s", path)
	if c.sudo {
		cmd = fmt.Sprintf("sudo %s", cmd)
	}
	err = session.Run(cmd)
	if err != nil {
		if bytes.Contains(stderr.Bytes(), []byte("No such file or directory")) {
			return "", false, nil
		}
		return "", false, err
	}

	return stdout.String(), true, nil
}

func (c *RemoteClient) ReadFilePermissions(path string, sudo bool) (string, error) {
	session, err := c.NewSession()
	if err != nil {
		return "", err
	}
	defer c.ReleaseSession(session)

	cmd := fmt.Sprintf("stat -c %%a %s", path)
	if c.sudo {
		cmd = fmt.Sprintf("sudo %s", cmd)
	}
	output, err := session.Output(cmd)
	if err != nil {
		return "", err
	}

	permissions := strings.ReplaceAll(string(output), "\n", "")
	if len(permissions) > 0 && len(permissions) < 4 {
		permissions = fmt.Sprintf("0%s", permissions)
	}
	return permissions, nil
}

func (c *RemoteClient) ReadFileOwner(path string, sudo bool) (string, error) {
	return c.StatFile(path, "u", sudo)
}

func (c *RemoteClient) ReadFileGroup(path string, sudo bool) (string, error) {
	return c.StatFile(path, "g", sudo)
}

func (c *RemoteClient) ReadFileOwnerName(path string, sudo bool) (string, error) {
	return c.StatFile(path, "U", sudo)
}

func (c *RemoteClient) ReadFileGroupName(path string, sudo bool) (string, error) {
	return c.StatFile(path, "G", sudo)
}

func (c *RemoteClient) StatFile(path string, char string, sudo bool) (string, error) {
	session, err := c.NewSession()
	if err != nil {
		return "", err
	}
	defer c.ReleaseSession(session)

	cmd := fmt.Sprintf("stat -c %%%s %s", char, path)
	if c.sudo {
		cmd = fmt.Sprintf("sudo %s", cmd)
	}
	output, err := session.Output(cmd)
	if err != nil {
		return "", err
	}

	group := strings.ReplaceAll(string(output), "\n", "")
	return group, nil
}

func (c *RemoteClient) DeleteFolder(path string, sudo bool) error {
	session, err := c.NewSession()
	if err != nil {
		return err
	}
	defer c.ReleaseSession(session)

	cmd := fmt.Sprintf("rm -rf %s", path)
	if c.sudo {
		cmd = fmt.Sprintf("sudo %s", cmd)
	}
	return run(session, cmd)
}

func (c *RemoteClient) DeleteFile(path string, sudo bool) error {
	return c.DeleteFileShell(path, sudo)
}

func (c *RemoteClient) DeleteFileShell(path string, sudo bool) error {
	session, err := c.NewSession()
	if err != nil {
		return err
	}
	defer c.ReleaseSession(session)

	cmd := fmt.Sprintf("rm %s", path)
	if c.sudo {
		cmd = fmt.Sprintf("sudo %s", cmd)
	}
	return run(session, cmd)
}

func NewRemoteClient(host string, clientConfig *ssh.ClientConfig, sudo bool, maxSessions int) (*RemoteClient, error) {
	client, err := ssh.Dial("tcp", host, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("couldn't establish a connection to the remote server: %s", err.Error())
	}

	// Create session pool with max size of 8 (leave some buffer below SSHD's default of 10)
	sessionPool := NewSessionPool(client, maxSessions)

	return &RemoteClient{
		sshClient:   client,
		sessionPool: sessionPool,
		sudo:        sudo,
	}, nil
}

func (c *RemoteClient) Close() error {
	c.sessionPool.Close()
	return c.sshClient.Close()
}

func (c *RemoteClient) GetSSHClient() *ssh.Client {
	return c.sshClient
}
