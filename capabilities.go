package main

import (
	"fmt"
	"log"
	"os"

	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/client"
)

func gitCapabilities(url string) {
	log.Println("new endpoint")
	endpoint, err := transport.NewEndpoint(url)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("new client")
	gitClient, err := client.NewClient(endpoint)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("new upload-pack-session")
	gitSession, err := gitClient.NewUploadPackSession(endpoint, nil)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("get advertised refs")
	advertisedRefs, err := gitSession.AdvertisedReferences()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("do stuff")
	foo := make(chan struct{})
	go func() {
		for _, item := range advertisedRefs.Capabilities.All() {
			fmt.Println(item)
		}
		foo <- struct{}{}
	}()
	go func() {
		err = gitSession.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()
	<-foo
}

func main() {
	remoteURL := os.Args[1]
	gitCapabilities(remoteURL)
}
