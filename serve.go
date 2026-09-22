package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/swaggest/openapi-go/openapi3"
	"github.com/swaggest/rest/web"
	"github.com/swaggest/usecase"
	"github.com/tkrajina/gpxgo/gpx"
	"github.com/vearutop/gpxt/static"
)

// posterSession is one uploaded GPX+image set, addressable by a short random ID and
// garbage collected sessionTTL after creation — there's no login, so the upload itself
// is the only thing standing in for an identity.
type posterSession struct {
	id    string
	dir   string
	files []string
	image string
	until time.Time
}

type sessionStore struct {
	mu       sync.Mutex
	sessions map[string]*posterSession
	rootDir  string
	ttl      time.Duration
}

func newSessionStore(rootDir string, ttl time.Duration) *sessionStore {
	return &sessionStore{sessions: make(map[string]*posterSession), rootDir: rootDir, ttl: ttl}
}

func (s *sessionStore) create() (*posterSession, error) {
	id, err := randomID()
	if err != nil {
		return nil, fmt.Errorf("generating session id: %w", err)
	}

	dir := filepath.Join(s.rootDir, id)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("creating session dir: %w", err)
	}

	sess := &posterSession{id: id, dir: dir, until: time.Now().Add(s.ttl)}

	s.mu.Lock()
	s.sessions[id] = sess
	s.mu.Unlock()

	return sess, nil
}

func (s *sessionStore) get(id string) (*posterSession, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sess, ok := s.sessions[id]
	if !ok || time.Now().After(sess.until) {
		return nil, false
	}

	return sess, true
}

// gc periodically deletes sessions past their TTL, along with their uploaded files.
func (s *sessionStore) gc(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.sweep()
		}
	}
}

func (s *sessionStore) sweep() {
	now := time.Now()

	s.mu.Lock()

	var expired []*posterSession

	for id, sess := range s.sessions {
		if now.After(sess.until) {
			expired = append(expired, sess)
			delete(s.sessions, id)
		}
	}

	s.mu.Unlock()

	for _, sess := range expired {
		if err := os.RemoveAll(sess.dir); err != nil {
			log.Println("gc: removing session dir:", err)
		}
	}
}

func randomID() (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}

// rateLimiter is a single global token bucket shared by every request. It's blunt (one
// heavy client can starve everyone else) but that's an accepted tradeoff for a public,
// unauthenticated service where the alternative is per-IP bookkeeping.
type rateLimiter struct {
	mu       sync.Mutex
	tokens   float64
	burst    float64
	perSec   float64
	lastFill time.Time
}

func newRateLimiter(perSecond float64, burst int) *rateLimiter {
	return &rateLimiter{tokens: float64(burst), burst: float64(burst), perSec: perSecond, lastFill: time.Now()}
}

func (rl *rateLimiter) allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	rl.tokens += now.Sub(rl.lastFill).Seconds() * rl.perSec
	rl.lastFill = now

	if rl.tokens > rl.burst {
		rl.tokens = rl.burst
	}

	if rl.tokens < 1 {
		return false
	}

	rl.tokens--

	return true
}

func rateLimitMiddleware(rl *rateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !rl.allow() {
				http.Error(w, "rate limit exceeded, try again shortly", http.StatusTooManyRequests)

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func serveCmd() {
	var (
		addr               string
		styleURL           string
		elevationThreshold float64
		sessionTTL         time.Duration
		maxUploadMB        int
		ratePerSecond      float64
		rateBurst          int
		tmpDir             string
	)

	cmd := kingpin.Command("serve", "Run gpxt as a public web service (currently: GPX poster)")
	cmd.Flag("addr", "Address to listen on.").Default(":8080").StringVar(&addr)
	cmd.Flag("style", "MapLibre style URL.").
		Default("https://tiles.openfreemap.org/styles/liberty").
		Envar("MAPLIBRE_STYLE").
		StringVar(&styleURL)
	cmd.Flag("elevation-threshold", "Minimum elevation change (m) counted as gain/loss, filters GPS/barometric noise.").
		Default("8").FloatVar(&elevationThreshold)
	cmd.Flag("session-ttl", "How long an uploaded session's files are kept before garbage collection.").
		Default("30m").DurationVar(&sessionTTL)
	cmd.Flag("max-upload-mb", "Max total upload size per session, megabytes.").
		Default("50").IntVar(&maxUploadMB)
	cmd.Flag("rate", "Global rate limit, requests per second across all clients — deliberately "+
		"coarse: a public unauthenticated service, so it protects the box, not individual users.").
		Default("5").Float64Var(&ratePerSecond)
	cmd.Flag("rate-burst", "Burst size for the global rate limit.").
		Default("15").IntVar(&rateBurst)
	cmd.Flag("tmp-dir", "Directory for uploaded session files.").
		Default(filepath.Join(os.TempDir(), "gpxt-serve")).StringVar(&tmpDir)

	cmd.Action(func(_ *kingpin.ParseContext) error {
		if err := os.MkdirAll(tmpDir, 0o700); err != nil {
			return fmt.Errorf("creating tmp dir: %w", err)
		}

		store := newSessionStore(tmpDir, sessionTTL)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go store.gc(ctx, time.Minute)

		s := web.NewService(openapi3.NewReflector())
		s.Use(rateLimitMiddleware(newRateLimiter(ratePerSecond, rateBurst)))
		s.Mount("/static/", http.StripPrefix("/static", Static))

		styleEndpoint := styleURL
		if !strings.HasPrefix(styleURL, "http://") && !strings.HasPrefix(styleURL, "https://") {
			styleData, err := os.ReadFile(styleURL)
			if err != nil {
				return fmt.Errorf("reading style file %s: %w", styleURL, err)
			}

			s.Get("/style.json", posterServeRaw(styleData, "application/json"))
			styleEndpoint = "/style.json"
		}

		s.Get("/", serveIndexPage())
		s.Wrapper.Get("/camera", serveCameraPage())

		maxUploadBytes := int64(maxUploadMB) << 20

		s.Wrapper.Get("/poster", serveUploadForm(sessionTTL))
		s.Wrapper.Post("/poster", serveUploadHandler(store, maxUploadBytes, sessionTTL))

		s.Get("/poster/{session}", servePosterPage(store, styleEndpoint))
		s.Get("/poster/{session}/image", servePosterImage(store))
		s.Get("/poster/{session}/stats.json", servePosterStats(store, elevationThreshold))
		s.Get("/poster/{session}/profiles.json", servePosterProfiles(store))
		s.Get("/poster/{session}/track/{id}.geojson", servePosterGeoJSON(store))

		log.Println("Serving gpxt on", addr)

		return http.ListenAndServe(addr, s) //nolint:gosec // Timeouts are the reverse proxy's job here.
	})
}

func serveIndexPage() usecase.Interactor {
	tmpl, err := static.Template("index.html")
	if err != nil {
		panic(err)
	}

	return usecase.NewInteractor(func(_ context.Context, _ struct{}, out *page) error {
		return out.Render(tmpl, nil)
	})
}

type uploadPageData struct {
	Error      string
	TTLMinutes int
}

func serveUploadForm(ttl time.Duration) http.HandlerFunc {
	tmpl, err := static.Template("upload.html")
	if err != nil {
		panic(err)
	}

	return func(w http.ResponseWriter, _ *http.Request) {
		renderUploadPage(w, tmpl, uploadPageData{TTLMinutes: int(ttl.Minutes())})
	}
}

func renderUploadPage(w http.ResponseWriter, tmpl *template.Template, data uploadPageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := tmpl.Execute(w, data); err != nil {
		log.Println("rendering upload page:", err)
	}
}

// serveUploadHandler accepts one background image and one or more GPX files, stashes
// them in a fresh session directory under fixed, safe names (no path traversal risk from
// user-supplied filenames), and redirects to the resulting poster page.
func serveUploadHandler(store *sessionStore, maxUploadBytes int64, ttl time.Duration) http.HandlerFunc {
	tmpl, err := static.Template("upload.html")
	if err != nil {
		panic(err)
	}

	fail := func(w http.ResponseWriter, status int, msg string) {
		w.WriteHeader(status)
		renderUploadPage(w, tmpl, uploadPageData{Error: msg, TTLMinutes: int(ttl.Minutes())})
	}

	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)

		if err := r.ParseMultipartForm(32 << 20); err != nil {
			fail(w, http.StatusRequestEntityTooLarge, "Upload too large or malformed: "+err.Error())

			return
		}

		imageHeaders := r.MultipartForm.File["image"]
		if len(imageHeaders) != 1 {
			fail(w, http.StatusBadRequest, "Exactly one background image is required.")

			return
		}

		gpxHeaders := r.MultipartForm.File["gpx"]
		if len(gpxHeaders) == 0 {
			fail(w, http.StatusBadRequest, "At least one GPX file is required.")

			return
		}

		sess, err := store.create()
		if err != nil {
			log.Println("creating session:", err)
			fail(w, http.StatusInternalServerError, "Could not create session, try again.")

			return
		}

		if err := saveUploadedImage(sess, imageHeaders[0]); err != nil {
			_ = os.RemoveAll(sess.dir)
			fail(w, http.StatusBadRequest, err.Error())

			return
		}

		for i, fh := range gpxHeaders {
			if err := saveUploadedGPX(sess, i, fh); err != nil {
				_ = os.RemoveAll(sess.dir)
				fail(w, http.StatusBadRequest, err.Error())

				return
			}
		}

		http.Redirect(w, r, "/poster/"+sess.id, http.StatusSeeOther)
	}
}

var imageContentTypeByExt = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".webp": "image/webp",
}

func saveUploadedImage(sess *posterSession, fh *multipart.FileHeader) error {
	ext := strings.ToLower(filepath.Ext(fh.Filename))
	if _, ok := imageContentTypeByExt[ext]; !ok {
		return errors.New("background image must be one of: jpg, jpeg, png, gif, webp")
	}

	data, err := readUploadedFile(fh)
	if err != nil {
		return fmt.Errorf("reading uploaded image: %w", err)
	}

	sess.image = filepath.Join(sess.dir, "image"+ext)

	return os.WriteFile(sess.image, data, 0o600)
}

func saveUploadedGPX(sess *posterSession, i int, fh *multipart.FileHeader) error {
	if !strings.HasSuffix(strings.ToLower(fh.Filename), ".gpx") {
		return fmt.Errorf("%s: not a .gpx file", fh.Filename)
	}

	data, err := readUploadedFile(fh)
	if err != nil {
		return fmt.Errorf("reading uploaded %s: %w", fh.Filename, err)
	}

	if _, err := gpx.ParseBytes(data); err != nil {
		return fmt.Errorf("%s: not a valid GPX file: %w", fh.Filename, err)
	}

	path := filepath.Join(sess.dir, fmt.Sprintf("track-%d.gpx", i))
	sess.files = append(sess.files, path)

	return os.WriteFile(path, data, 0o600)
}

func readUploadedFile(fh *multipart.FileHeader) ([]byte, error) {
	f, err := fh.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return io.ReadAll(f)
}

// posterSessionInput is the shared path-param shape for every /poster/{session}/... route.
type posterSessionInput struct {
	Session string `path:"session"`
}

// sessionNotFoundError reports as 404, not the default 500 — an expired/unknown session
// is a client-side condition (stale bookmark, GC'd upload), not a server failure.
type sessionNotFoundError string

func (e sessionNotFoundError) Error() string {
	return "session " + string(e) + " not found or expired"
}

func (e sessionNotFoundError) HTTPStatus() int {
	return http.StatusNotFound
}

func lookupSession(store *sessionStore, id string) (*posterSession, error) {
	sess, ok := store.get(id)
	if !ok {
		return nil, sessionNotFoundError(id)
	}

	return sess, nil
}

func servePosterPage(store *sessionStore, styleURL string) usecase.Interactor {
	tmpl, err := static.Template("poster.html")
	if err != nil {
		panic(err)
	}

	return usecase.NewInteractor(func(_ context.Context, in posterSessionInput, out *page) error {
		sess, err := lookupSession(store, in.Session)
		if err != nil {
			return err
		}

		return out.Render(tmpl, posterPageData{
			Files:    sess.files,
			StyleURL: styleURL,
			BasePath: "/poster/" + sess.id,
		})
	})
}

func servePosterImage(store *sessionStore) usecase.Interactor {
	return usecase.NewInteractor(func(_ context.Context, in posterSessionInput, out *usecase.OutputWithEmbeddedWriter) error {
		rw, ok := out.Writer.(http.ResponseWriter)
		if !ok {
			return errors.New("missing http.ResponseWriter")
		}

		sess, err := lookupSession(store, in.Session)
		if err != nil {
			return err
		}

		data, err := os.ReadFile(sess.image)
		if err != nil {
			return err
		}

		ct := imageContentTypeByExt[strings.ToLower(filepath.Ext(sess.image))]
		if ct == "" {
			ct = "image/jpeg"
		}

		rw.Header().Set("Content-Type", ct)
		_, err = rw.Write(data)

		return err
	})
}

func servePosterStats(store *sessionStore, elevationThreshold float64) usecase.Interactor {
	return usecase.NewInteractor(func(_ context.Context, in posterSessionInput, out *usecase.OutputWithEmbeddedWriter) error {
		rw, ok := out.Writer.(http.ResponseWriter)
		if !ok {
			return errors.New("missing http.ResponseWriter")
		}

		sess, err := lookupSession(store, in.Session)
		if err != nil {
			return err
		}

		stats, err := computePosterStats(sess.files, elevationThreshold)
		if err != nil {
			return err
		}

		rw.Header().Set("Content-Type", "application/json")

		return json.NewEncoder(rw).Encode(stats)
	})
}

func servePosterProfiles(store *sessionStore) usecase.Interactor {
	return usecase.NewInteractor(func(_ context.Context, in posterSessionInput, out *usecase.OutputWithEmbeddedWriter) error {
		rw, ok := out.Writer.(http.ResponseWriter)
		if !ok {
			return errors.New("missing http.ResponseWriter")
		}

		sess, err := lookupSession(store, in.Session)
		if err != nil {
			return err
		}

		profiles, err := computePosterProfiles(sess.files)
		if err != nil {
			return err
		}

		rw.Header().Set("Content-Type", "application/json")

		return json.NewEncoder(rw).Encode(profiles)
	})
}

func servePosterGeoJSON(store *sessionStore) usecase.Interactor {
	type req struct {
		posterSessionInput
		ID uint `path:"id"`
	}

	return usecase.NewInteractor(func(_ context.Context, in req, out *usecase.OutputWithEmbeddedWriter) error {
		rw, ok := out.Writer.(http.ResponseWriter)
		if !ok {
			return errors.New("missing http.ResponseWriter")
		}

		sess, err := lookupSession(store, in.Session)
		if err != nil {
			return err
		}

		if in.ID >= uint(len(sess.files)) {
			return fmt.Errorf("unexpected id %d, max %d", in.ID, len(sess.files))
		}

		doc, err := gpx.ParseFile(sess.files[in.ID])
		if err != nil {
			return err
		}

		geojson, err := gpxToGeoJSON(doc, sess.files[in.ID])
		if err != nil {
			return err
		}

		rw.Header().Set("Content-Type", "application/geo+json")
		_, err = rw.Write([]byte(geojson))

		return err
	})
}
