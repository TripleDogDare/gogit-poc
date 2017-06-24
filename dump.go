/*
	The is a POC of dumping a particular commit to a directory
*/
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/src-d/go-billy.v3/osfs"
	sdgit "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/filemode"
	"gopkg.in/src-d/go-git.v4/plumbing/format/packfile"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/protocol/packp"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/client"
	"gopkg.in/src-d/go-git.v4/storage"
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
	Main(remoteURL, ref.Hash(), workingDirectory, 0)
}

func Main(remoteURL string, commitHash plumbing.Hash, workingDirectory string, recursionDepth int) {
	indent := strings.Repeat("\t", recursionDepth+1)
	fmt.Printf("%sCloning %s from %s to %s\n", strings.Repeat("\t", recursionDepth), commitHash, remoteURL, workingDirectory)

	gitStore := memory.NewStorage() // store git objects
	fs := osfs.New(workingDirectory)
	repository, err := sdgit.Init(gitStore, fs)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%sCreate pack request: %v\n", indent, commitHash)
	uploadRequest := packp.NewUploadPackRequest()
	uploadRequest.Wants = []plumbing.Hash{commitHash}

	fmt.Printf("%sFetch: %s\n", indent, remoteURL)
	response := fetch(remoteURL, uploadRequest)

	fmt.Printf("%sUpdate store", indent)
	err = packfile.UpdateObjectStorage(gitStore, response)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%sPlacing: %s\n", indent, workingDirectory)
	worktree := checkout(repository, commitHash)

	fmt.Printf("%sList submoudles...", indent)
	subs := listSubmodules(gitStore, worktree, commitHash)
	fmt.Printf("%sFound submodules: %d\n", indent, len(subs))

	for cfg, entry := range subs {
		fmt.Printf("%sSubmodule: %s\n", indent, cfg.Path)
		Main(cfg.URL, entry.Hash, filepath.Join(workingDirectory, cfg.Path), recursionDepth+1)
	}
}

func checkout(repository *sdgit.Repository, commitHash plumbing.Hash) *sdgit.Worktree {
	worktree, err := repository.Worktree()
	if err != nil {
		log.Fatal(err)
	}
	err = worktree.Checkout(&sdgit.CheckoutOptions{Hash: commitHash})
	if err != nil {
		log.Fatal(err)
	}
	return worktree
}

func listSubmodules(gitStore storage.Storer, worktree *sdgit.Worktree, commitHash plumbing.Hash) map[*config.Submodule]*object.TreeEntry {
	commit, err := object.GetCommit(gitStore, commitHash)
	if err != nil {
		log.Fatal(err)
	}
	tree, err := commit.Tree()
	if err != nil {
		log.Fatal(err)
	}
	subs, err := worktree.Submodules()
	if err != nil {
		log.Fatal(err)
	}
	result := map[*config.Submodule]*object.TreeEntry{}
	for _, submodule := range subs {
		cfg := submodule.Config()
		entry, err := tree.FindEntry(cfg.Path)
		if err != nil {
			log.Fatal(err)
		}
		isSubmodule := entry.Mode == filemode.Submodule
		if !isSubmodule {
			log.Fatalf("Entry is not a submodule: %+v", entry)
		}
		result[cfg] = entry
	}
	return result
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
