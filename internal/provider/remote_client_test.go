package provider

import (
	"bytes"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestSshRootPasswordAuth(t *testing.T) {
	clientConfig := ssh.ClientConfig{
		User:            "root",
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	clientConfig.Auth = append(clientConfig.Auth, ssh.Password("password"))

	_, err := NewRemoteClient("localhost:8022", &clientConfig, false)
	if err != nil {
		t.Errorf("Couldn't connect to root@localhost:8022. Error: %s", err)
	}
}

func TestWriteFile(t *testing.T) {
	clientConfig := ssh.ClientConfig{
		User:            "root",
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	clientConfig.Auth = append(clientConfig.Auth, ssh.Password("password"))

	client, _ := NewRemoteClient("localhost:8022", &clientConfig, false)

	err := client.WriteFile("blabetiblou", "/tmp/test", true)

	if err != nil {
		t.Errorf("unable to create remote file: %s", err)
	}
}

func TestWriteFileNonSudoFail(t *testing.T) {
	clientConfig := ssh.ClientConfig{
		User:            "raphaeljoie",
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	clientConfig.Auth = append(clientConfig.Auth, ssh.Password("password"))

	client, _ := NewRemoteClient("localhost:8022", &clientConfig, false)
	err := client.WriteFile("blabetiblou", "/home/file", false)

	if err == nil {
		t.Errorf("Didn't fail as expected")
	}

	localError, ok := err.(Error)
	if !ok {
		t.Errorf("Unexpected error: %s", err)
	}
	if ok && !bytes.HasSuffix(localError.stderr, []byte("Permission denied\n")) {
		t.Errorf("Didn't fail as expected %s", localError.stderr)
	}
}
