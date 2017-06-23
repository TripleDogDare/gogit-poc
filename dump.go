/*
	The is a POC of dumping a particular commit to a directory
*/
package main

import (
	"fmt"
	"log"
	"os"

	"gopkg.in/src-d/go-billy.v3/osfs"
	sdgit "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/format/packfile"
	"gopkg.in/src-d/go-git.v4/plumbing/protocol/packp"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/client"
	"gopkg.in/src-d/go-git.v4/storage/memory"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Println("<remote> <ref> <target dir>")
		os.Exit(1)
	}
	log.SetFlags(log.Lshortfile)
	remoteURL := os.Args[1]
	hash := os.Args[2]
	workingDirectory := os.Args[3]
	fmt.Printf("remote: %s\n", remoteURL)
	fmt.Printf("hash: %s\n", hash)
	fmt.Printf("dir: %s\n", workingDirectory)
	ref := plumbing.NewReferenceFromStrings("", hash)
	Main(remoteURL, ref.Hash(), workingDirectory)
}

func Main(remoteURL string, commitHash plumbing.Hash, workingDirectory string) {
	gitStore := memory.NewStorage() // store git objects
	fs := osfs.New(workingDirectory)
	repository, err := sdgit.Init(gitStore, fs)
	if err != nil {
		log.Fatal(err)
	}

	uploadRequest := packp.NewUploadPackRequest()
	uploadRequest.Wants = []plumbing.Hash{commitHash}

	response := fetch(remoteURL, uploadRequest)
	err = packfile.UpdateObjectStorage(gitStore, response)
	if err != nil {
		log.Fatal(err)
	}
	checkout(repository, commitHash)
}

func checkout(repository *sdgit.Repository, commitHash plumbing.Hash) {
	worktree, err := repository.Worktree()
	if err != nil {
		log.Fatal(err)
	}
	err = worktree.Checkout(&sdgit.CheckoutOptions{Hash: commitHash})
	if err != nil {
		log.Fatal(err)
	}
}

func fetch(url string, uploadRequest *packp.UploadPackRequest) *packp.UploadPackResponse {
	endpoint, err := transport.NewEndpoint(url)
	if err != nil {
		log.Fatal(err)
	}
	gitClient, err := client.NewClient(endpoint)
	if err != nil {
		log.Fatal(err)
	}
	session, err := gitClient.NewUploadPackSession(endpoint, nil)
	if err != nil {
		log.Fatal(err)
	}
	response, err := session.UploadPack(uploadRequest)
	if err != nil {
		log.Fatal(err)
	}
	return response
}
