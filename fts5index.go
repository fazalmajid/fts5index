package main

import (
	"database/sql"
	"flag"
	"log"
	"os"
	"runtime/pprof"
	"path/filepath"
	"fmt"
	"strings"
	"net/http"
	"time"
	"io"
	"html/template"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"github.com/jaytaylor/html2text"
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
	port := flag.String("p", "localhost:8080", "host address and port to bind to")
	tmpl_fn := flag.String("template", "", "Go template for the search page")
	do_html := flag.Bool("html", false, "Index HTML files")
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
			_, err = db.Exec("CREATE VIRTUAL TABLE search USING fts5(path UNINDEXED, title, text)")
			if err != nil {
				log.Fatal("Could not create search table", err)
			}
		}
	} else {
		log.Fatal("no SQLite index filename supplied")
	}

	log.Println("indexing...")
	before := time.Now()
	if *do_html {
		index(db, updated,
			func(path string)bool {
				fn := strings.ToLower(path)
				return strings.HasSuffix(fn, ".html") || strings.HasSuffix(fn, ".htm")
			},
			func(path string) (string, string, error) {
				f, err := os.Open(path)
				if err != nil {
					return "", "", err
				}
				html_doc, err := html.Parse(f)
				if err != nil {
					return "", "", err
				}
				f.Close()
				title := extract_title(html_doc)
				text, err := html2text.FromHTMLNode(html_doc)
				if err != nil {
					return "", "", err
				}
				return title, text, nil
			})
	}
	log.Println("done in", time.Now().Sub(before))
	
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

func index(db *sql.DB, updated time.Time, fn_filter func(string)bool, fn_title_text func(string) (string, string, error)) {
	stmt, err := db.Prepare("INSERT OR REPLACE INTO search (path, title, text) VALUES (?, ?, ?)")
	if err != nil {
		log.Fatal("Could not prepare insert statement", err)
	}

	// walk the current directory looking for HTML files
	err = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if ! fn_filter(path) {
			return nil
		}
		stat, err := os.Stat(path)
		if stat.ModTime().Before(updated) {
			return nil
		}
		if *verbose {
			fmt.Println(path);
		}
		title, text, err := fn_title_text(path)
		if err != nil {
			return err
		}
		if *verbose {
		 	fmt.Println("\t", title);
		}
		// if *verbose {
		// 	fmt.Println(text);
		// }
		_, err = stmt.Exec(path, title, text)
		return err
	})
	stmt.Close()
}

type Result struct {
	Path string
	Summary string
	Text string
}

func do_error(w http.ResponseWriter, msg string) {
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

func serve(db *sql.DB, port string, tmpl *template.Template) {
	SearchHandler := func (w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			do_error(w, "Could not parse search form.")
			return
		}
		raw_term := r.Form.Get("q")
		
		if raw_term == "" {
			do_error(w, "Please enter search terms.")
			return
		}
		term, err := fts5_term(raw_term)
		if err != nil {
			do_error(w, "Malformed search terms.")
			return
		}
		rows, err := db.Query("SELECT path, title, text FROM search WHERE search=" + term)
		if err != nil {
			do_error(w, "Query error: " + err.Error())
			return
		}
		results := make([]Result, 10)
		for rows.Next() {
			var path, title, text string
			err = rows.Scan(&path, &title, &text)
			if err != nil {
				do_error(w, "Row error: " + err.Error())
				return
			}
			results = append(results, Result{path, title, text})
			log.Printf("query %q result\n\t%s\n\t%s\n", term, path, text[:72])
		}
		data := make(map[string]interface{}, 1)
		data["Results"] = results
		err = tmpl.Execute(w, data)
		if err != nil {
			do_error(w, "Template error: " + err.Error())
			return
		}
	}
	http.HandleFunc("/search", SearchHandler)
	log.Fatal(http.ListenAndServe(port, nil))
}
