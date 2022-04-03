GO=	env CGO_CFLAGS=-D__EXTENSIONS__=1 go

all: fts5index

fts5index: fts5index.go fts5query.go
	$(GO) build -v --tags fts5 -o $@ fts5index.go fts5query.go

test:
	$(GO) test

solaris: clean
	env GOOS=illumos GOARCH=amd64 gmake fts5index

clean:
	-rm -rf bin src pkg fts5index *~ core search.db
