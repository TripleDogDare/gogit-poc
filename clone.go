package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"gopkg.in/src-d/go-billy.v3/osfs"
	sdgit "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/storage/memory"
)

func gitClone(remoteURL string, dataHash string, workingDirectory string, progress io.Writer) {
	if workingDirectory == "" {
		panic(meep.Meep(
			&ErrInternal{Msg: "Unexpected empty working directory path"},
		))
	}
	options := sdgit.CloneOptions{
		URL:           remoteURL,
		ReferenceName: plumbing.ReferenceName(dataHash),
		Depth:         1,
		Progress:      progress,
	}
	storer := memory.NewStorage() // store git objects
	fs := osfs.New(workingDirectory)
	_, err := sdgit.Clone(storer, fs, &options)
	if err != nil {
		panic(meep.Meep(
			&ErrInternal{Msg: "git clone failed"},
			meep.Cause(err),
		))
	}
	return
}

func main() {
	if len(os.Args) != 4 {
		fmt.Println("<remote> <ref> <target dir>")
		os.Exit(1)
	}
	log.SetFlags(log.Lshortfile)

	remoteURL := os.Args[1]
	ref := os.Args[2]
	workingDirectory := os.Args[3]
	fmt.Printf("remote: %s\n", remoteURL)
	fmt.Printf("ref: %s\n", ref)
	fmt.Printf("dir: %s\n", workingDirectory)
	gitClone(remoteURL, ref, workingDirectory, os.Stderr)
}
