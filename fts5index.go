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

	_ "github.com/mattn/go-sqlite3"
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
	bufsize := flag.Int("buf", 1048576, "buffer size for reading HTML files")
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

	if *dsn != "" {
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
			_, err = db.Exec("CREATE VIRTUAL TABLE search USING fts5(path UNINDEXED, text)")
			if err != nil {
				log.Fatal("Could not create search table", err)
			}
		}
	} else {
		log.Fatal("no SQLite index filename supplied")
	}

	index(db, *bufsize);
}

func index(db *sql.DB, bufsize int) {
	stmt, err := db.Prepare("INSERT OR REPLACE INTO search (path, text) VALUES (?, ?)")
	if err != nil {
		log.Fatal("Could not prepare insert statement", err)
	}

	// walk the current directory looking for HTML files
	var buf []byte = make([]byte, bufsize)
	
	err = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		lp := strings.ToLower(path)
		if !strings.HasSuffix(lp, ".html") && !strings.HasSuffix(lp, ".htm") {
			return nil
		}
		if *verbose {
			fmt.Println(path);
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		len, err := f.Read(buf)
		if len == bufsize {
			log.Println("warning: filled buffer on", path)
		}
		if err != nil {
			return err
		}
		text, err := html2text.FromString(string(buf[0:len]))
		if err != nil {
			return err
		}
		// if *verbose {
		// 	fmt.Println(text);
		// }
		f.Close()
		_, err = stmt.Exec(path, text)
		return err
	})
	stmt.Close()
	db.Close()
}
