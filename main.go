package main

import (
	"flag"
	"fmt"
	"github.com/cinode/golib/blobstore"
	"html"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"regexp"
)

////////////////////////////////////////////////////////////////////
// Flags

var storagePath string
var initialBid string
var initialKey string

func initFlags() {

	flag.StringVar(&storagePath, "storage", "", "Storage path")
	flag.StringVar(&storagePath, "s", "", "Storage path")

	var initialBlob string
	flag.StringVar(&initialBlob, "initialblob", "", "Initial blob")
	flag.StringVar(&initialBlob, "ib", "", "Initial blob")

	flag.Parse()

	if initialBlob != "" {
		matches := regexp.MustCompile(`^([a-zA-Z0-9]+):([a-zA-Z0-9]+)$`).FindStringSubmatch(initialBlob)
		if matches == nil {
			panic("Invalid initial blob parameter, make sure it's of form: <BID>:<KEY>")
		}
		initialBid = matches[1]
		initialKey = matches[2]
	}

}

////////////////////////////////////////////////////////////////////
// Blob handling

var blobUrlRegex = regexp.MustCompile(`^/blob/([0-9A-Fa-f]+)/([0-9A-Fa-f]+)$`)
var blobStorage blobstore.BlobStorage

// Blob request handler
func blobHandler(w http.ResponseWriter, r *http.Request) {

	// The url must be in a form /blob/<BID>/<KEY>
	matches := blobUrlRegex.FindStringSubmatch(r.URL.Path)
	if matches == nil {
		http.NotFound(w, r)
		return
	}
	bid := matches[1]
	key := matches[2]

	if !handleDirectory(w, r, bid, key, false) {
		handleFile(w, r, bid, key, "")
	}
}

func handleDirectory(w http.ResponseWriter, r *http.Request, bid, key string, prettyPath bool) bool {

	// Open the directory
	reader := blobstore.NewDirBlobReader(blobStorage)
	err := reader.Open(bid, key)
	if err != nil {
		// Let's assume it's a file blob then
		return false
	}

	// Prepare html layout
	fmt.Fprint(w, "<html><head><title>Directory listing</title></head><body><h1>Directory listing:</h1><pre>")
	defer fmt.Fprint(w, "</pre></body></html>")

	for reader.IsNextEntry() {
		entry, err := reader.NextEntry()
		if err != nil {
			fmt.Fprintf(w, "Error happened: %s", err)
			break
		}

		if prettyPath {
			fmt.Fprintf(w, "<a href=\"%s\">%s</a>\n",
				html.EscapeString(entry.Name),
				html.EscapeString(entry.Name),
			)
		} else {
			fmt.Fprintf(w, "<a href=\"/blob/%s/%s\">%s</a>\n",
				html.EscapeString(entry.Bid),
				html.EscapeString(entry.Key),
				html.EscapeString(entry.Name),
			)
		}
	}

	return true
}

func handleFile(w http.ResponseWriter, r *http.Request, bid, key string, mime string) {
	// Open the blob
	blobFileReader := blobstore.NewFileBlobReader(blobStorage)
	err := blobFileReader.Open(bid, key)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Read up to 512 first bytes in order to detect the mime type
	if mime == "" {

		buff := make([]byte, 512)
		n, err := blobFileReader.Read(buff)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		buff = buff[:n]

		mime := http.DetectContentType(buff)
		w.Header().Add("Content-type", mime)

		w.Write(buff)

	} else {

		w.Header().Add("Content-type", mime)

	}

	io.Copy(w, blobFileReader)
}

////////////////////////////////////////////////////////////////////
// Path handling

var pathPartMatch = regexp.MustCompile(`^/?([^/]+)`)

func pathHandler(w http.ResponseWriter, r *http.Request) {

	// Make sure the initial blob has been configured
	if initialBid == "" {
		http.NotFound(w, r)
		return
	}

	path := r.URL.Path
	bid, key := initialBid, initialKey
	name := ""

	for {

		// The path ends with '/' character -
		// must be interpreted as a directory
		if path == "/" {
			if !handleDirectory(w, r, bid, key, true) {
				http.NotFound(w, r)
			}
			return
		}

		if path == "" {

			// If that's the directory blob, redirect it so that it
			// contains the trailing '/'
			if blobstore.NewDirBlobReader(blobStorage).Open(bid,key) == nil {
				http.Redirect( w, r, r.URL.Path + "/", http.StatusMovedPermanently )
				return
			}

			handleFile(w, r, bid, key,
				mime.TypeByExtension(filepath.Ext(name)))
			return
		}

		matches := pathPartMatch.FindStringSubmatch(path)
		if matches == nil {
			// Malformed path
			http.NotFound(w, r)
			return
		}
		// Cut off this part of the path
		path = path[len(matches[0]):]
		name = matches[1]

		// Search for entry in the current blob dir
		reader := blobstore.NewDirBlobReader(blobStorage)
		if reader.Open(bid, key) != nil {
			http.NotFound(w, r)
			return
		}

		// TODO: Use some kind of optimized search
		found := false
		for reader.IsNextEntry() {
			entry, err := reader.NextEntry()
			if err != nil {
				http.NotFound(w, r)
				return
			}
			if entry.Name == name {
				bid, key, found = entry.Bid, entry.Key, true
			}
		}

		if !found {
			http.NotFound(w, r)
			return
		}
	}

}

////////////////////////////////////////////////////////////////////
// Main code

func initStorage() bool {

	if storagePath == "" {
		return false
	}

	blobStorage = blobstore.NewFileBlobStorage(storagePath)
	return true
}

func usage() {
	fmt.Printf("Ussage: goclient -s <storage path>\n")
}

// Main function
func main() {

	initFlags()

	if !initStorage() {
		usage()
		return
	}

	http.HandleFunc("/blob/", blobHandler)
	http.HandleFunc("/", pathHandler)
	http.ListenAndServe(":8080", nil)
}
