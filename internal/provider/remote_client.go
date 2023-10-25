package provider

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

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

type RemoteClient struct {
	sshClient *ssh.Client
	sudo      bool
}

func (c *RemoteClient) WriteFile(content string, path string, sudo bool) error {
	return c.WriteFileShell(content, path, sudo)
}

func (c *RemoteClient) WriteFileShell(content string, path string, sudo bool) error {
	sshClient := c.GetSSHClient()

	session, err := sshClient.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

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
	return run(session, cmd)
}

func (c *RemoteClient) ChmodFile(path string, permissions string, sudo bool) error {
	sshClient := c.GetSSHClient()

	session, err := sshClient.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	cmd := fmt.Sprintf("chmod %s %s", permissions, path)
	if c.sudo {
		cmd = fmt.Sprintf("sudo %s", cmd)
	}
	return run(session, cmd)
}

func (c *RemoteClient) CreateDir(path string, sudo bool) error {
	sshClient := c.GetSSHClient()

	session, err := sshClient.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	cmd := fmt.Sprintf("mkdir -p %s", path)
	if c.sudo {
		cmd = fmt.Sprintf("sudo %s", cmd)
	}
	return run(session, cmd)
}

func (c *RemoteClient) ChgrpFile(path string, group string, sudo bool) error {
	sshClient := c.GetSSHClient()

	session, err := sshClient.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	cmd := fmt.Sprintf("chgrp %s %s", group, path)
	if c.sudo {
		cmd = fmt.Sprintf("sudo %s", cmd)
	}

	return run(session, cmd)
}

func (c *RemoteClient) ChownFile(path string, owner string, sudo bool) error {
	sshClient := c.GetSSHClient()

	session, err := sshClient.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	cmd := fmt.Sprintf("chown %s %s", owner, path)
	if c.sudo {
		cmd = fmt.Sprintf("sudo %s", cmd)
	}
	return run(session, cmd)
}

func (c *RemoteClient) FileExists(path string, sudo bool) (bool, error) {
	sshClient := c.GetSSHClient()

	session, err := sshClient.NewSession()
	if err != nil {
		return false, err
	}
	defer session.Close()

	cmd := fmt.Sprintf("test -f %s", path)
	if c.sudo {
		cmd = fmt.Sprintf("sudo %s", cmd)
	}
	err = run(session, cmd)

	if err != nil {
		session2, err := sshClient.NewSession()
		if err != nil {
			return false, err
		}
		defer session2.Close()

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
	sshClient := c.GetSSHClient()

	session, err := sshClient.NewSession()
	if err != nil {
		return false, err
	}
	defer session.Close()

	cmd := fmt.Sprintf("[ -d \"%s\" ] && exit 0 || exit 1 ", path)
	_, err = session.CombinedOutput(cmd)
	if err != nil {
		return false, nil
	}

	return true, nil
}

func (c *RemoteClient) ReadFileShell(path string, sudo bool) (string, bool, error) {
	sshClient := c.GetSSHClient()

	session, err := sshClient.NewSession()
	if err != nil {
		return "", false, err
	}
	defer session.Close()

	cmd := fmt.Sprintf("cat %s", path)
	if c.sudo {
		cmd = fmt.Sprintf("sudo %s", cmd)
	}
	content, err := session.CombinedOutput(cmd)
	if err != nil {
		// TODO find a more reliable way to catch file not found
		if bytes.Contains(content, []byte("No such file or directory")) {
			return "", false, nil
		}
		return "", false, errors.New(string(content))
	}

	return string(content), true, nil
}

func (c *RemoteClient) ReadFilePermissions(path string, sudo bool) (string, error) {
	sshClient := c.GetSSHClient()

	session, err := sshClient.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

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
	sshClient := c.GetSSHClient()

	session, err := sshClient.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

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
	sshClient := c.GetSSHClient()

	session, err := sshClient.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

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
	sshClient := c.GetSSHClient()

	session, err := sshClient.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	cmd := fmt.Sprintf("rm %s", path)
	if c.sudo {
		cmd = fmt.Sprintf("sudo %s", cmd)
	}
	return run(session, cmd)
}

func NewRemoteClient(host string, clientConfig *ssh.ClientConfig, sudo bool) (*RemoteClient, error) {
	client, err := ssh.Dial("tcp", host, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("couldn't establish a connection to the remote server: %s", err.Error())
	}

	return &RemoteClient{
		sshClient: client,
		sudo:      sudo,
	}, nil
}

func (c *RemoteClient) Close() error {
	return c.sshClient.Close()
}

func (c *RemoteClient) GetSSHClient() *ssh.Client {
	return c.sshClient
}
