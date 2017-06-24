package main

import (
	"fmt"
	"log"
	"os"

	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/storer"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/client"
	"gopkg.in/src-d/go-git.v4/storage/memory"
)

func gitLsRemote(url string) memory.ReferenceStorage {
	endpoint, err := transport.NewEndpoint(url)
	if err != nil {
		log.Fatal(err)
	}
	gitClient, err := client.NewClient(endpoint)
	if err != nil {
		log.Fatal(err)
	}
	gitSession, err := gitClient.NewUploadPackSession(endpoint, nil)
	if err != nil {
		log.Fatal(err)
	}
	advertisedRefs, err := gitSession.AdvertisedReferences()
	if err != nil {
		log.Fatal(err)
	}
	refs, err := advertisedRefs.AllReferences()
	if err != nil {
		log.Fatal(err)
	}
	err = gitSession.Close()
	if err != nil {
		log.Fatal(err)
	}
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
