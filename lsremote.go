package main

import (
	"fmt"
	"log"
	"os"

	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/storer"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/client"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"
	"gopkg.in/src-d/go-git.v4/storage/memory"
)

func Challenge(user, instruction string, questions []string, echos []bool) (answers []string, err error) {
	fmt.Println(user)
	fmt.Println(instruction)
	fmt.Println(questions)
	fmt.Println(echos)
	answers, err = []string{}, nil
	return
}

func gitLsRemote(url string) memory.ReferenceStorage {
	endpoint, err := transport.NewEndpoint(url)
	if err != nil {
		log.Fatal(err)
	}
	log.Println(endpoint.Protocol())
	log.Println(endpoint.User())
	gitClient, err := client.NewClient(endpoint)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("created client")
	// authMethod := &ssh.KeyboardInteractive{
	// 	Challenge: Challenge,
	// }
	authMethod, err := ssh.NewSSHAgentAuth(endpoint.User())
	if err != nil {
		log.Fatal(err)
	}
	gitSession, err := gitClient.NewUploadPackSession(endpoint, authMethod)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("created session")
	advertisedRefs, err := gitSession.AdvertisedReferences()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("received adv refs")
	refs, err := advertisedRefs.AllReferences()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("received all refs")
	err = gitSession.Close()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("closed session")
	return refs
}

func PrintRefs(refs storer.ReferenceStorer) {
	iter, _ := refs.IterReferences()
	iter.ForEach(func(ref *plumbing.Reference) error {
		if ref.Type() == plumbing.HashReference {
			fmt.Println(ref)
		} else if ref.Type() == plumbing.SymbolicReference {
			target, _ := refs.Reference(ref.Target())
			fmt.Printf("%s %s\n", target.Hash(), ref.Name())
		}
		return nil
	})

}

func main() {
	remoteURL := os.Args[1]
	refs := gitLsRemote(remoteURL)
	PrintRefs(refs)
}
