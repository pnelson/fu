package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/pnelson/fu"
	"github.com/pnelson/fu/api"
	"github.com/pnelson/fu/http"
)

var (
	// General flags.
	help = flag.Bool("h", false, "show this usage information")

	addr  = flag.String("addr", os.Getenv("FU_ADDR"), "http server address")
	token = flag.String("token", os.Getenv("FU_TOKEN"), "secret token required to upload")

	// Server flags.
	s             = flag.Bool("s", false, "start a fu http server on addr")
	dbPath        = flag.String("db-path", "store.db", "path to database file")
	uploadDir     = flag.String("upload-dir", "uploads", "path to upload directory")
	maxUploadSize = flag.Int64("max-upload-size", 32<<20, "max upload file size in bytes")

	// Client flags.
	d = flag.Duration("d", time.Hour, "duration to keep file")
)

func init() {
	log.SetFlags(0)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTION]... [FILE]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "When FILE is -, read from stdin.\n\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  fu main.go\n")
		fmt.Fprintf(os.Stderr, "  echo 'Hello, world.' | fu -\n")
	}
}

func main() {
	flag.Parse()
	if *help {
		flag.Usage()
		return
	}
	if *s {
		if *token == "" {
			log.Println("fu: running insecurely without token")
		}
		config := api.Config{
			Addr:           *addr,
			Token:          []byte(*token),
			DataSourceName: *dbPath,
			UploadDir:      *uploadDir,
			MaxUploadSize:  *maxUploadSize,
		}
		err := http.Serve(config)
		if err != nil {
			log.Fatal(err)
		}
		return
	}
	var f *os.File
	c, err := fu.NewClient(*addr, *token)
	if err != nil {
		log.Fatal(err)
	}
	name := flag.Arg(0)
	if flag.NArg() == 1 {
		if name == "-" {
			f = os.Stdin
		} else {
			f, err = os.Open(name)
			if err != nil {
				log.Fatal(err)
			}
			defer f.Close()
		}
	} else {
		flag.Usage()
		os.Exit(2)
		return
	}
	file, err := c.Upload(f, name, *d)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(c.URL(file))
}
