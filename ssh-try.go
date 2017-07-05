/*
Attempt to achieve a simmilar effect as the following:
```
 $ ssh -T tripledogdare.github.com
Hi TripleDogDare! You've successfully authenticated, but GitHub does not provide shell access.
```

*/
package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/kevinburke/ssh_config"
	"github.com/xanzy/ssh-agent"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

const (
	CFG_IDENTITY_FILE = "IdentityFile"
	CFG_HOSTNAME      = "Hostname"
	CFG_PORT          = "Port"
)

func SSHAgent() ssh.AuthMethod {
	sshAgent, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
	if err != nil {
		log.Fatalf("error creating SSH agent: %q", err)
	}
	a := agent.NewClient(sshAgent)
	signers, _ := a.Signers()
	log.Printf("signers: %d\n", len(signers))
	return ssh.PublicKeysCallback(a.Signers)
}

func XanzySSHAgent() ssh.AuthMethod {
	a, _, err := sshagent.New()
	if err != nil {
		log.Fatalf("error creating SSH agent: %q", err)
	}
	signers, _ := a.Signers()
	log.Printf("signers: %d\n", len(signers))
	return ssh.PublicKeysCallback(a.Signers)
}

func NewSshCfg() *ssh_config.Config {
	f, err := os.Open(filepath.Join(os.Getenv("HOME"), ".ssh", "config"))
	if err != nil {
		log.Fatalf("error opening ssh config file: %q", err)
	}
	cfg, err := ssh_config.Decode(f)
	if err != nil {
		log.Fatalf("error decoding ssh config file: %q", err)
	}
	return cfg
}

func PrintSshConfig(cfg *ssh_config.Config) {
	for _, host := range cfg.Hosts {
		fmt.Println("patterns:", host.Patterns)
		for _, node := range host.Nodes {
			// Manipulate the nodes as you see fit, or use a type switch to
			// distinguish between Empty, KV, and Include nodes.
			fmt.Println(node.String())
		}
	}
}

func SSHConfigAuth(cfg *ssh_config.Config, target *url.URL) ssh.AuthMethod {
	idFilePath, err := cfg.Get(target.Hostname(), CFG_IDENTITY_FILE)
	if err != nil {
		log.Fatalf("Unable to retrieve identity file: %q", err)
	}
	log.Printf("%s: %s\n", CFG_IDENTITY_FILE, idFilePath)
	// Create the Signer for this private key.
	key, err := ioutil.ReadFile(idFilePath)
	if err != nil {
		log.Fatalf("Unable to retrieve identity key: %q", err)
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		log.Fatalf("unable to parse private key: %v", err)
	}
	return ssh.PublicKeys(signer)
}

func Translate(cfg *ssh_config.Config, target *url.URL) string {
	host, err := cfg.Get(target.Hostname(), CFG_HOSTNAME)
	if err != nil {
		log.Fatalf("Unable to retrieve hostname from config: %q", err)
	}
	port, err := cfg.Get(target.Hostname(), CFG_PORT)
	if err != nil {
		log.Fatalf("Unable to retrieve hostname from config: %q", err)
	}
	if host == "" {
		host = target.Hostname()
	}
	if port == "" {
		port = target.Port()
		if port == "" {
			port = "22"
		}
	}
	return fmt.Sprintf("%s:%s", host, port)
}

func main() {
	log.SetFlags(log.Lshortfile)
	target, err := url.Parse(os.Args[1])
	if err != nil {
		log.Fatalf("bad url: %q", err)
	}

	cfg := NewSshCfg()
	config := ssh.ClientConfig{
		User: "git",
		Auth: []ssh.AuthMethod{
			SSHConfigAuth(cfg, target),
			// XanzySSHAgent(),
		},
		// Auth: []ssh.AuthMethod{ssh.}
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         time.Second * 5,
	}
	remote := Translate(cfg, target)
	log.Println(remote)
	client, err := ssh.Dial("tcp", remote, &config)
	if err != nil {
		log.Fatalf("error creating SSH client: %q", err)
	}
	session, err := client.NewSession()
	if err != nil {
		log.Fatalf("error creating SSH session: %q", err)
	}
	defer session.Close()

	// // Start remote shell
	stdout, err := session.StdoutPipe()
	if err != nil {
		log.Fatalf("failed to get stdout pipe: %q", err)
	}

	stderr, err := session.StdoutPipe()
	if err != nil {
		log.Fatalf("failed to get stderr pipe: %q", err)
	}

	if err := session.Shell(); err != nil {
		log.Fatalf("failed to start shell: %q", err)
	}

	log.Println("waiting...")
	err = session.Wait()
	io.Copy(os.Stdout, stdout)
	io.Copy(os.Stdout, stderr)
	os.Stdout.Sync()
	if err != nil {
		log.Fatalf("error: %q", err)
	}
}
