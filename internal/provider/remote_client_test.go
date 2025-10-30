package provider

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

func _client(user string) *RemoteClient {
	clientConfig := ssh.ClientConfig{
		User:            user,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	clientConfig.Auth = append(clientConfig.Auth, ssh.Password("password"))

	client, _ := NewRemoteClient("localhost:8022", &clientConfig, false, 1)

	return client
}

func TestSshRootPasswordAuth(t *testing.T) {
	clientConfig := ssh.ClientConfig{
		User:            "root",
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	clientConfig.Auth = append(clientConfig.Auth, ssh.Password("password"))

	_, err := NewRemoteClient("localhost:8022", &clientConfig, false, 1)
	if err != nil {
		t.Errorf("Couldn't connect to root@localhost:8022. Error: %s", err)
	}
}

func TestWriteFile(t *testing.T) {
	err := _client("root").WriteFile("blabetiblou", "/tmp/test", true, false)

	if err != nil {
		t.Errorf("unable to create remote file: %s", err)
	}
}

func TestWriteFileEnsureDir(t *testing.T) {
	err := _client("root").WriteFile("blabetiblou", "/tmp/blabetiblou/test", true, true)

	if err != nil {
		t.Errorf("unable to create remote file: %s", err)
	}
}

func TestWriteFileEnsureDirFail(t *testing.T) {
	// "randomize" the path to make sure it doesn't exist yet
	path := fmt.Sprintf("/etc/doesnt-exists-%d/file", time.Now().UnixMilli())
	err := _client("root").WriteFile(
		"blabetiblou",
		path,
		true, false,
	)

	if err == nil {
		t.Errorf("Didn't fail as expected")
	}

	localError, ok := err.(Error)
	if !ok {
		t.Errorf("Unexpected error: %s", err)
	}
	if ok && !bytes.Equal(localError.stderr, []byte(fmt.Sprintf("tee: %s: No such file or directory\n", path))) {
		t.Errorf("Didn't fail with the expected message. %s", localError.stderr)
	}
}

func TestWriteFileNonSudoFail(t *testing.T) {
	err := _client("raphaeljoie").WriteFile("blabetiblou", "/home/file", false, false)

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
