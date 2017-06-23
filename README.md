# gogit-poc
POC files using [src-d/go-git](https://github.com/src-d/go-git)

Needs src-d/go-git and related dependencies
```
go get gopkg.in/src-d/go-git.v4
```

`go run ./dump.go 'https://github.com/polydawn/repeatr.git' '615f57306c7bfbea934cf264a9230c25775a8115' ./repeatr`
Will dump the files from the given commit hash to `./repeatr`
