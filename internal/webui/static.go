package webui

import (
	"io/fs"
	"net/http"
)

// Handler returns an http.Handler serving the embedded frontend build as
// a single-page app: a request for any path that isn't a real file under
// dist/ falls back to index.html, so client-side routing (react-router)
// owns the URL space. It returns (nil, nil) if the frontend hasn't been
// built yet (dist/ has no index.html) — callers should treat that as
// "run API-only", not an error.
func Handler() (http.Handler, error) {
	return spaHandler(DistFS)
}

// spaHandler builds the handler against any fs.FS so tests can exercise
// the fallback logic without a real frontend build embedded.
func spaHandler(distFS fs.FS) (http.Handler, error) {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return nil, err
	}
	if _, err := fs.Stat(sub, "index.html"); err != nil {
		return nil, nil
	}

	fileServer := http.FileServerFS(sub)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if len(path) > 0 && path[0] == '/' {
			path = path[1:]
		}
		if path != "" {
			if info, err := fs.Stat(sub, path); err != nil || info.IsDir() {
				r = r.Clone(r.Context())
				r.URL.Path = "/"
			}
		}
		fileServer.ServeHTTP(w, r)
	}), nil
}
