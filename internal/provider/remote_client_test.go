package provider

import (
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestSshRootPasswordAuth(t *testing.T) {
	clientConfig := ssh.ClientConfig{
		User:            "root",
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	clientConfig.Auth = append(clientConfig.Auth, ssh.Password("password"))

	_, err := NewRemoteClient("localhost:8022", &clientConfig)
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

	client, _ := NewRemoteClient("localhost:8022", &clientConfig)

	err := client.WriteFile("blabetiblou", "/tmp/test", "permissions", true)

	if err != nil {
		t.Errorf("unable to create remote file: %s", err)
	}
}
