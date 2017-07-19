GO=	env GOPATH=`pwd` go

all: fts5index

DEPS=	src/github.com/jaytaylor/html2text \
	src/github.com/mattn/go-sqlite3

fts5index: $(DEPS) fts5index.go
	$(GO) build  --tags fts5 fts5index.go

src/github.com/mattn/go-sqlite3:
	$(GO) get -f -t -u -v --tags fts5 github.com/mattn/go-sqlite3

src/github.com/jaytaylor/html2text:
	$(GO) get -f -t -u -v --tags fts5 github.com/jaytaylor/html2text

clean:
	-rm -rf src pkg fts5index *~ core search.db
