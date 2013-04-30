package main

import (
	"github.com/cinode/golib/blobstore"
	"io"
	"net/http"
	"regexp"
)

var blobUrlRegex = regexp.MustCompile(`^/blob/([0-9A-Fa-f]+)/([0-9A-Fa-f]+)$`)
var blobStorage = blobstore.NewFileBlobStorage("../../../storage")
var blobFileReder = blobstore.FileBlobReader{Storage: blobStorage}

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
	err := blobFileReder.Open(bid, key)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	
	io.Copy( w, &blobFileReder );
}

// Main function
func main() {

	http.HandleFunc("/blob/", blobHandler)
	http.ListenAndServe(":8080", nil)
}
