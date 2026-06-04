package static

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
)

//go:embed dist
var dist embed.FS

// Handler serves the embedded frontend static files.
// It strips the "dist" prefix and supports SPA routing by falling back to index.html.
func Handler() http.Handler {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := path.Clean(r.URL.Path)
		if p == "/" {
			p = "/index.html"
		}

		// Try to serve the requested file
		data, err := fs.ReadFile(sub, p[1:]) // strip leading /
		if err != nil {
			// If file not found, serve index.html for SPA routing
			data, err = fs.ReadFile(sub, "index.html")
			if err != nil {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(data)
			return
		}

		// Set content type based on extension
		ext := path.Ext(p)
		switch ext {
		case ".html":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		case ".js":
			w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		case ".css":
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
		case ".svg":
			w.Header().Set("Content-Type", "image/svg+xml")
		case ".json":
			w.Header().Set("Content-Type", "application/json")
		}
		w.Write(data)
	})
}
