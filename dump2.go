/*
	The is a POC of dumping a particular commit to a directory
*/
package main

import (
	"fmt"
	"io"
	stdioutil "io/ioutil"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/src-d/go-billy.v3"
	"gopkg.in/src-d/go-billy.v3/osfs"
	// sdgit "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/filemode"
	"gopkg.in/src-d/go-git.v4/plumbing/format/packfile"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/protocol/packp"
	"gopkg.in/src-d/go-git.v4/plumbing/storer"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/client"
	"gopkg.in/src-d/go-git.v4/storage"
	"gopkg.in/src-d/go-git.v4/storage/filesystem"
	"gopkg.in/src-d/go-git.v4/utils/ioutil"
)

const MaxRecusionDepth = 1

/*
	Return a string that's safe to use as a dir name.

	Uses URL query escaping so it remains roughly readable.
	Does not attempt any URL normalization.
*/
func slugifyRemote(remoteURL string) string {
	return url.QueryEscape(remoteURL)
}

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
	cacheDir := ".cache"
	Main(remoteURL, ref.Hash(), cacheDir, workingDirectory, 0)
}

func Main(remoteURL string, commitHash plumbing.Hash, cacheDir string, workingDirectory string, recursionDepth int) {
	indent := strings.Repeat("\t", recursionDepth+1)
	fmt.Printf("%sCloning %s from %s to %s\n", strings.Repeat("\t", recursionDepth), commitHash, remoteURL, workingDirectory)

	if commitHash.IsZero() {
		log.Fatal("super wat")
	}

	// cache of the .git files
	cacheFS := osfs.New(filepath.Join(cacheDir, slugifyRemote(remoteURL), commitHash.String()))
	gitStore, err := filesystem.NewStorage(cacheFS) // store git objects
	if err != nil {
		log.Fatal(err)
	}

	// where the repository files will go
	fs := osfs.New(workingDirectory)

	fmt.Printf("%sCreate pack request: %v\n", indent, commitHash)
	uploadRequest := packp.NewUploadPackRequest()
	uploadRequest.Wants = []plumbing.Hash{commitHash}
	if uploadRequest.IsEmpty() {
		log.Fatal("wat")
	}
	fmt.Printf("%sFetch: %s\n", indent, remoteURL)
	response := fetch(remoteURL, uploadRequest)

	fmt.Printf("%sUpdate store\n", indent)
	err = packfile.UpdateObjectStorage(gitStore, response)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%sPlacing: %s\n", indent, workingDirectory)
	checkout(gitStore, fs, commitHash)

	if recursionDepth >= MaxRecusionDepth {
		fmt.Printf("%sReached recursion depth limit: %d \n", indent, MaxRecusionDepth)
		return
	}

	fmt.Printf("%sList submoudles...\n", indent)
	subs := listSubmodules(gitStore, fs, commitHash)
	fmt.Printf("%sFound submodules: %d\n", indent, len(subs))

	for cfg, entry := range subs {
		fmt.Printf("%sSubmodule: %s\n", indent, cfg.Path)
		Main(cfg.URL, entry.Hash, cacheDir, filepath.Join(workingDirectory, cfg.Path), recursionDepth+1)
	}
}

type FilePlacer interface {
	Place(*object.File) error
}

type GitFilePlacer struct {
	billy.Filesystem
}

func (g *GitFilePlacer) Place(f *object.File) error {
	return checkoutFile(f, g)
}

func NewFilePlacer(fs billy.Filesystem) FilePlacer {
	return &GitFilePlacer{fs}
}

func PlaceTree(tree *object.Tree, fp FilePlacer) {
	fileIterator := tree.Files()
	err := fileIterator.ForEach(fp.Place)
	if err != nil {
		log.Fatal(err)
	}
}

func checkout(store storer.Storer, fs billy.Filesystem, commitHash plumbing.Hash) {
	commit, err := object.GetCommit(store, commitHash)
	if err != nil {
		log.Fatal(err)
	}
	tree, err := commit.Tree()
	if err != nil {
		log.Fatal(err)
	}
	PlaceTree(tree, NewFilePlacer(fs))
}

func listSubmodules(gitStore storage.Storer, fs billy.Filesystem, commitHash plumbing.Hash) map[*config.Submodule]*object.TreeEntry {
	commit, err := object.GetCommit(gitStore, commitHash)
	if err != nil {
		log.Fatal(err)
	}
	tree, err := commit.Tree()
	if err != nil {
		log.Fatal(err)
	}

	cfgModules, err := readGitmodulesFile(fs)
	if err != nil {
		log.Fatal(err)
	}

	result := map[*config.Submodule]*object.TreeEntry{}
	if cfgModules != nil {
		for _, submodule := range cfgModules.Submodules {
			if submodule == nil {
				log.Fatal("nil submodule")
			}
			entry, err := tree.FindEntry(submodule.Path)
			if err != nil {
				log.Fatal(err)
			}
			isSubmodule := entry.Mode == filemode.Submodule
			if !isSubmodule {
				log.Fatalf("Entry is not a submodule: %+v", entry)
			}
			result[submodule] = entry
		}
	}
	return result
}

func fetch(rawUrl string, uploadRequest *packp.UploadPackRequest) *packp.UploadPackResponse {
	parsedUrl, err := url.Parse(rawUrl)
	if err != nil {
		log.Fatal(err)
	}
	// force https submodules on github.com
	if strings.EqualFold(parsedUrl.Hostname(), "github.com") {
		parsedUrl.Scheme = "https"
		log.Println("Rewriting URL: ", parsedUrl.String())
	}
	endpoint, err := transport.NewEndpoint(parsedUrl.String())
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
	err = session.Close()
	if err != nil {
		log.Fatal(err)
	}

	return response
}

// !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
// THE FOLLOWING IS COPIED FROM go-git:Worktree FOR _REASONS_
// !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
const gitmodulesFile = ".gitmodules"

// Copied from go-git/worktree with minor changes
func checkoutFile(f *object.File, fs billy.Filesystem) (err error) {
	mode, err := f.Mode.ToOSFileMode()
	if err != nil {
		return
	}

	if mode&os.ModeSymlink != 0 {
		return checkoutFileSymlink(f, fs)
	}

	from, err := f.Reader()
	if err != nil {
		return
	}

	defer ioutil.CheckClose(from, &err)

	to, err := fs.OpenFile(f.Name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode.Perm())
	if err != nil {
		return
	}

	defer ioutil.CheckClose(to, &err)

	_, err = io.Copy(to, from)
	return
}

// Copied from go-git/worktree with minor changes
func checkoutFileSymlink(f *object.File, fs billy.Filesystem) (err error) {
	from, err := f.Reader()
	if err != nil {
		return
	}

	defer ioutil.CheckClose(from, &err)

	bytes, err := stdioutil.ReadAll(from)
	if err != nil {
		return
	}

	err = fs.Symlink(string(bytes), f.Name)
	return
}

// Copied from go-git/worktree with minor changes
func readGitmodulesFile(fs billy.Filesystem) (*config.Modules, error) {
	f, err := fs.Open(gitmodulesFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, err
	}

	input, err := stdioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	m := config.NewModules()
	return m, m.Unmarshal(input)
}
