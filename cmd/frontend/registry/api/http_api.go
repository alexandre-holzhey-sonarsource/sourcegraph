package api

import (
	"net/http"
)

// HandleRegistry is called to handle HTTP requests for the extension registry.
var HandleRegistry = func(w http.ResponseWriter, r *http.Request) error {
	http.Error(w, "no local extension registry exists", http.StatusNotFound)
	return nil
}
