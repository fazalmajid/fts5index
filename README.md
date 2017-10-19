## SQLite FTS5 site search engine for Hugo

This is a simple search facility for Hugo generated sites. It uses SQLite's
[FTS5][1] full-text search engine, and a Go template that is itself generated
from your site's theme for seamless look and feel integration.

The SQLite file format is portable across OS and architectures. You could
build an index on a Mac and deploy to a Linux server.
https://www.sqlite.org/aff_short.html

### Installation

To build `fts5index` from source, run:

    env GOPATH=`pwd` go get -f -t -u -v github.com/fazalmajid/fts5index

copy `bin/fts5index` somewhere in your `$PATH`.

`fts5index` incorporates a subset of Hugo and you will need to rebuild it
whenever you rebuild your copy of Hugo.


### Indexing your Hugo site

From your Hugo main directory (i.e. where you would run `hugo` from):

    fts5index -hugo

This will generate a SQLite DB file (by default `search.db`).

### Running the search server

You will need to assign a port (by default port 8086 on `localhost`) for the
server to run on. Note: it is *not* recommended to bind `fts5index` to
anything other than `localhost`. While `fts5index` was coded with attention to
security best practices, it has not been hardened or independently audited for
potential security vulnerabilities.

The copy the file `search.md` somewhere in your `hugo/content` directory,
e.g. `hugo/content/search.md`. When Hugo is run, this will generate a file
`hugo/public/search/index.html` with your theme settings.

You will then start the search server using:

    fts5index -template public/search/index.html

(follow your operating system instructions for how to turn this into a
permanent service, using tools like Solaris SMF, Linux' systemd, OS X'
launchctl or DJB's daemontools).

Configure your web server so that `GET` requests to the page, e.g. `/search`
are proxied to `fts5index` instead of served from static files. On nginx, you would add the stanza:

    location /search {
      proxy_pass             http://127.0.0.1:8086/search;
    }

You can then add the search form to your templates:

    <form action="{{ .Site.BaseURL }}search">
      <input name="q">
      <input type="submit" value="Search">
    </form>

 [1]: https://www.sqlite.org/fts5.html
 