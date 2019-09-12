package main

import (
	"database/sql"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/jaytaylor/html2text"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	//"github.com/gohugoio/hugo/parser"
	"github.com/gohugoio/hugo/hugolib"
	"github.com/gohugoio/hugo/resources/resource"
	//"github.com/spf13/cast"
	//"github.com/spf13/afero"
	"github.com/gohugoio/hugo/deps"
	"github.com/gohugoio/hugo/hugofs"
)

const date_fmt = "[02/Jan/2006:15:04:05 -0700]"
const iso_8601 = "2006-01-02 15:04:05"

var (
	verbose *bool
)

func main() {
	// command-line options
	verbose = flag.Bool("v", false, "Verbose error reporting")
	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to file")
	dsn := flag.String("db", "search.db", "SQLite DB to use for the search index")
	port := flag.String("p", "localhost:8086", "host address and port to bind to")
	tmpl_fn := flag.String("template", "", "Go template for the search page")
	do_html := flag.Bool("html", false, "Index HTML files")
	do_hugo := flag.Bool("hugo", false, "Index Hugo markdown files")
	flag.Parse()
	var err error
	var f *os.File
	var db *sql.DB
	// Profiler
	if *cpuprofile != "" {
		f, err = os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	var updated time.Time
	if *dsn != "" {
		stat, err := os.Stat(*dsn)
		if err == nil {
			updated = stat.ModTime()
		}

		db, err = sql.Open("sqlite3", *dsn)
		if err != nil {
			log.Fatalf("ERROR: opening SQLite DB %q, error: %s", *dsn, err)
		}

		row := db.QueryRow("SELECT count(sql) FROM sqlite_master WHERE name='search'")
		var count int32
		err = row.Scan(&count)
		if err != nil {
			log.Fatal("Could not check table status ")
		}
		if count == 0 {
			_, err = db.Exec("CREATE VIRTUAL TABLE search USING fts5(path UNINDEXED, title, text, summary UNINDEXED)")
			if err != nil {
				log.Fatal("Could not create search table", err)
			}
		}
	} else {
		log.Fatal("no SQLite index filename supplied")
	}

	if *do_html {
		before := time.Now()
		log.Println("indexing HTML...")
		index_html(db, updated)
		log.Println("done in", time.Now().Sub(before))
	}
	if *do_hugo {
		before := time.Now()
		log.Println("indexing Hugo...")
		index_hugo(db, updated)
		log.Println("done in", time.Now().Sub(before))
	}

	if *tmpl_fn != "" {
		tmpl, err := template.ParseFiles(*tmpl_fn)
		if err != nil {
			log.Fatal("Could not load template:", *tmpl_fn, ":", err)
		}
		serve(db, *port, tmpl)
	}
	db.Close()
}

// recurse through the parsed HTML tree to extract the title
func extract_title(n *html.Node) string {
	if n.Type == html.ElementNode && n.DataAtom == atom.Title {
		return n.FirstChild.Data
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		result := extract_title(c)
		if result != "" {
			return result
		}
	}
	return ""
}

func index_html(db *sql.DB, updated time.Time) {
	stmt, err := db.Prepare("INSERT OR REPLACE INTO search (path, title, text) VALUES (?, ?, ?)")
	if err != nil {
		log.Fatal("Could not prepare insert statement", err)
	}

	// walk the current directory looking for HTML files
	err = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		fn := strings.ToLower(path)
		if !(strings.HasSuffix(fn, ".html") || strings.HasSuffix(fn, ".htm")) {
			return nil
		}
		stat, err := os.Stat(path)
		if stat.ModTime().Before(updated) {
			return nil
		}
		if *verbose {
			fmt.Println(path)
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		html_doc, err := html.Parse(f)
		if err != nil {
			return err
		}
		f.Close()
		title := extract_title(html_doc)
		text, err := html2text.FromHTMLNode(html_doc)
		if err != nil {
			return err
		}
		if *verbose {
			fmt.Println("\t", title)
		}
		// if *verbose {
		// 	fmt.Println(text);
		// }
		_, err = stmt.Exec(path, title, text)
		return err
	})
	stmt.Close()
}

func index_hugo(db *sql.DB, updated time.Time) {
	stmt, err := db.Prepare("INSERT OR REPLACE INTO search (path, title, text, summary) VALUES (?, ?, ?, ?)")
	if err != nil {
		log.Fatal("Could not prepare insert statement", err)
	}

	osFs := hugofs.Os
	cfg, err := hugolib.LoadConfigDefault(osFs)
	if err != nil {
		log.Fatal("Could not load Hugo config.toml", err)
	}
	fs := hugofs.NewFrom(osFs, cfg)
	sites, err := hugolib.NewHugoSites(deps.DepsCfg{Fs: fs, Cfg: cfg})
	if err != nil {
		log.Fatal("Could not load Hugo site(s)", err)
	}
	err = sites.Build(hugolib.BuildCfg{SkipRender: true})
	if err != nil {
		log.Fatal("Could not run render", err)
	}
	for _, p := range sites.Pages() {
		if p.Draft() || resource.IsFuture(p) || resource.IsExpired(p) {
			continue
		}
		title := p.Title()
		path := p.Permalink()
		text := p.Plain()
		if *verbose {
			fmt.Println(path)
			fmt.Println("\t", title)
			//fmt.Println("\t", p.Summary);
			fmt.Println()
		}
		_, err = stmt.Exec(path, title, text, p.Summary)
		if err != nil {
			log.Println("path = ", path, "title = ", title, "text = ", text, "summary = ", p.Summary)
			log.Fatal("Could not write page to DB", err)
		}
	}
	stmt.Close()
}

type Result struct {
	Path    string
	Title   string
	Summary template.HTML
	Text    string
}

func do_error(w http.ResponseWriter, msg string, query string) {
	if query != "" {
		log.Println("query error:", query, msg)
	}
	io.WriteString(w, fmt.Sprintf(`<!doctype html>
<html>
  <head>
    <title>Error</title>
  </head>
  <body>
    <h1>Error</h1>
    <p>%s</p>
  </body>
</html>
`, msg))
}

func min(a int, b int) int {
	if a < b {
		return a
	} else {
		return b
	}
}

func serve(db *sql.DB, port string, tmpl *template.Template) {
	SearchHandler := func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			do_error(w, "Could not parse search form.", "")
			return
		}
		raw_term := r.Form.Get("q")

		if raw_term == "" {
			do_error(w, "Please enter search terms.", "")
			return
		}
		term, err := fts5_term(raw_term)
		if err != nil {
			do_error(w, "Malformed search terms.", "")
			return
		}
		query := "SELECT path, title, summary, text FROM search WHERE search=?"
		if *verbose {
			log.Println("Query: ", query, term)
		}
		rows, err := db.Query(query, term)
		if err != nil {
			do_error(w, "Query error: "+err.Error(), term)
			return
		}
		results := make([]Result, 0)
		for rows.Next() {
			var path, title, summary, text string
			err = rows.Scan(&path, &title, &summary, &text)
			if err != nil {
				do_error(w, "Row error: "+err.Error(), term)
				return
			}
			results = append(results, Result{path, title, template.HTML(summary), text})
			log.Printf("query %q result\n\t%s\n\t%s\n", term, path, text[:min(len(text), 72)])
		}
		data := make(map[string]interface{}, 1)
		data["Results"] = results
		err = tmpl.Execute(w, data)
		if err != nil {
			do_error(w, "Template error: "+err.Error(), query)
			return
		}
	}
	http.HandleFunc("/search", SearchHandler)
	if *verbose {
		log.Println("starting web server on ", port)
	}
	log.Fatal(http.ListenAndServe(port, nil))
}
