// internal/server/server.go
package server

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"nibl/internal/builder"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

func Run(port int, buildFunc func(builder.BuildOptions) error, opts builder.BuildOptions) error {
	opts.CleanDestination = true
	if err := buildFunc(opts); err != nil {
		return fmt.Errorf("initial build failed: %w", err)
	}

	hub := newHub()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("could not create file watcher: %w", err)
	}
	defer watcher.Close()

	// Use a map to track watched directories and avoid duplicates.
	watchedDirs := make(map[string]bool)

	// Helper function to add a directory to the watcher if it hasn't been added already.
	addWatch := func(dir string) {
		// Clean the path to have a consistent map key.
		dir = filepath.Clean(dir)
		if !watchedDirs[dir] {
			if err := watcher.Add(dir); err != nil {
				log.Printf("Error adding watch on %s: %v", dir, err)
			} else {
				fmt.Printf("Watching directory: %s\n", dir)
				watchedDirs[dir] = true
			}
		}
	}

	pathsToWatch := []string{"content", "templates", "static", "site.yaml", "site.biff"}
	for _, path := range pathsToWatch {
		info, err := os.Stat(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return fmt.Errorf("could not stat path %s: %w", path, err)
		}

		if info.IsDir() {
			// If it's a directory, walk it and add all subdirectories.
			if err := filepath.Walk(path, func(walkPath string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if info.IsDir() {
					addWatch(walkPath)
				}
				return nil
			}); err != nil {
				return fmt.Errorf("failed to watch directory %s: %w", path, err)
			}
		} else {
			// For files, watch their PARENT directory. This handles Vim's save-swap behavior.
			addWatch(filepath.Dir(path))
		}
	}

	opts.CleanDestination = false
	go watchForChanges(watcher, hub, buildFunc, opts)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})

	fileServer := http.FileServer(http.Dir("public"))
	mux.Handle("/", liveReloadWrapper(fileServer))

	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("Serving site on http://localhost%s\n", addr)
	fmt.Println("Press Ctrl+C to stop")
	return http.ListenAndServe(addr, mux)
}

func watchForChanges(watcher *fsnotify.Watcher, hub *Hub, buildFunc func(builder.BuildOptions) error, opts builder.BuildOptions) {
	var lastBuildTime time.Time
	const debounceDuration = 500 * time.Millisecond

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			// We now get notifications for create, write, remove, and rename
			// to robustly handle all editor save strategies.
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
				if time.Since(lastBuildTime) > debounceDuration {
					time.Sleep(100 * time.Millisecond)

					log.Printf("Change detected in %s, rebuilding...", event.Name)
					if err := buildFunc(opts); err != nil {
						log.Printf("Error rebuilding site: %v", err)
					} else {
						log.Println("Site rebuilt successfully. Triggering reload...")
						hub.broadcastMessage([]byte("reload"))
					}
					lastBuildTime = time.Now()
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %v", err)
		}
	}
}

func liveReloadWrapper(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")

		isHTML := strings.HasSuffix(r.URL.Path, ".html") || strings.HasSuffix(r.URL.Path, "/")

		if !isHTML {
			next.ServeHTTP(w, r)
			return
		}

		iw := newInterceptingWriter(w)
		next.ServeHTTP(iw, r)

		for key, values := range iw.Header() {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}

		bodyBytes := iw.body.Bytes()

		if iw.statusCode != http.StatusOK {
			w.WriteHeader(iw.statusCode)
			w.Write(bodyBytes)
			return
		}

		injectedBody := bytes.Replace(bodyBytes, []byte("</body>"), []byte(liveReloadScript+"</body>"), 1)
		w.Header().Set("Content-Length", fmt.Sprint(len(injectedBody)))
		w.WriteHeader(iw.statusCode)
		w.Write(injectedBody)
	})
}

type interceptingWriter struct {
	http.ResponseWriter
	body       *bytes.Buffer
	statusCode int
	header     http.Header
}

func newInterceptingWriter(w http.ResponseWriter) *interceptingWriter {
	return &interceptingWriter{
		ResponseWriter: w,
		body:           new(bytes.Buffer),
		header:         make(http.Header),
		statusCode:     http.StatusOK,
	}
}

func (iw *interceptingWriter) Header() http.Header {
	return iw.header
}

func (iw *interceptingWriter) Write(b []byte) (int, error) {
	return iw.body.Write(b)
}

func (iw *interceptingWriter) WriteHeader(statusCode int) {
	iw.statusCode = statusCode
}

const liveReloadScript = `
<script>
  (function() {
    let socket = new WebSocket("ws://" + window.location.host + "/ws");
    socket.onmessage = function(event) {
      if (event.data === "reload") {
        console.log("Reloading page...");
        window.location.reload();
      }
    };
    socket.onclose = function() {
      // Don't log on normal close, it's just noise.
    };
    socket.onerror = function(error) {
      console.error("Live reload connection error. Please restart 'nibl serve'.");
    };
  })();
</script>
`

