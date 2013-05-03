package main

import (
	"flag"
	"fmt"
	"github.com/cinode/golib/blobstore"
	"io"
	"net/http"
	"regexp"
)

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

	// Open the blob
	blobFileReader := blobstore.NewFileBlobReader(blobStorage)
	err := blobFileReader.Open(bid, key)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Read up to 512 first bytes in order to detect the mime type
	buff := make([]byte, 512)
	n, err := blobFileReader.Read(buff)
	buff = buff[:n]

	contentType := http.DetectContentType(buff)
	w.Header().Add("Content-type", contentType)

	w.Write(buff)

	io.Copy(w, blobFileReader)
}

func initStorage() bool {

	var storagePath string
	flag.StringVar(&storagePath, "storage", "", "Storage path")
	flag.StringVar(&storagePath, "s", "", "Storage path")

	flag.Parse()

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

	if !initStorage() {
		usage()
		return
	}

	http.HandleFunc("/blob/", blobHandler)
	http.ListenAndServe(":8080", nil)
}
