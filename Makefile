GO=	env GOPATH=`pwd` go

all: fts5index

DEPS=	src/github.com/jaytaylor/html2text \
	src/github.com/mattn/go-sqlite3 \
	src/github.com/gohugoio/hugo

fts5index: $(DEPS) fts5index.go fts5query.go
	$(GO) build  --tags fts5 fts5index.go fts5query.go

src/github.com/mattn/go-sqlite3:
	$(GO) get -f -t -u -v --tags fts5 github.com/mattn/go-sqlite3

src/github.com/jaytaylor/html2text:
	$(GO) get -f -t -u -v --tags fts5 github.com/jaytaylor/html2text

src/github.com/gohugoio/hugo:
	$(GO) get -f -t -u -v --tags fts5 github.com/gohugoio/hugo

test:
	$(GO) test

clean:
	-rm -rf src pkg fts5index *~ core search.db
