package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/skip2/go-qrcode"
	"golang.org/x/crypto/bcrypt"
)

type Config struct {
	StorageDir     string
	Port           int
	Password       string
	Verbose        bool
	NgrokURL       string
	SessionTimeout time.Duration
}

type Session struct {
	ID        string
	Users     map[string]bool
	CreatedAt time.Time
	LastUsed  time.Time
}

var (
	config        Config
	sessions      = &sync.Map{}
	logger        *log.Logger
	friendlyWords = []string{"Friend", "Share", "Group", "Team", "Mate", "Buddy", "Pal", "Crew"}
)

func main() {
	// Seed random number generator
	rand.Seed(time.Now().UnixNano())

	// Setup logging first
	setupLogging()

	// Parse flags
	parseFlags()

	// Ensure storage directory exists
	if err := os.MkdirAll(config.StorageDir, 0755); err != nil {
		logger.Fatalf("Failed to create storage directory: %v", err)
	}

	// Setup cleanup routine
	go sessionCleanupRoutine()

	// Start server
	startServer()
}

func setupLogging() {
	logFlags := log.LstdFlags
	if config.Verbose {
		logFlags |= log.Lshortfile
	}
	logger = log.New(os.Stdout, "gshare: ", logFlags)
}

func parseFlags() {
	flag.StringVar(&config.StorageDir, "dir", "gshare-cache", "Directory to store shared files")
	flag.IntVar(&config.Port, "port", 8080, "Port to run the server on")
	flag.StringVar(&config.Password, "password", "", "Password to protect file access")
	flag.StringVar(&config.NgrokURL, "ngrok-url", "", "Ngrok public URL (optional)")
	flag.BoolVar(&config.Verbose, "verbose", false, "Enable verbose logging")
	flag.DurationVar(&config.SessionTimeout, "session-timeout", 1*time.Hour, "Session inactivity timeout")
	flag.Parse()
}

func sessionCleanupRoutine() {
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		cleanupInactiveSessions()
	}
}

func cleanupInactiveSessions() {
	now := time.Now()
	sessions.Range(func(key, value interface{}) bool {
		session := value.(*Session)
		if now.Sub(session.LastUsed) > config.SessionTimeout {
			sessionID := key.(string)
			if err := os.RemoveAll(filepath.Join(config.StorageDir, sessionID)); err != nil {
				logger.Printf("Failed to delete inactive session directory %s: %v", sessionID, err)
			} else if config.Verbose {
				logger.Printf("Cleaned up inactive session %s", sessionID)
			}
			sessions.Delete(sessionID)
		}
		return true
	})
}

func startServer() {
	http.Handle("/", http.FileServer(http.Dir("./filesharefrontend/public/")))
	http.HandleFunc("/api/session/create", withLogging(createSession))
	http.HandleFunc("/api/session/join", withLogging(joinSession))
	http.HandleFunc("/api/files", withLogging(handleFiles))
	http.HandleFunc("/api/session/qr", withLogging(serveQRCode))

	// Detect Ngrok URL
	if ngrokURL := getNgrokURL(); ngrokURL != "" {
		config.NgrokURL = ngrokURL
		if config.Verbose {
			logger.Printf("Ngrok URL detected: %s", config.NgrokURL)
		}
	}

	logger.Printf("Starting server on port %d", config.Port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", config.Port), nil); err != nil {
		logger.Fatalf("Server failed to start: %v", err)
	}
}

func withLogging(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if config.Verbose {
			logger.Printf("Request: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		}
		handler(w, r)
	}
}

func getIP(remoteAddr string) string {
	if idx := strings.LastIndex(remoteAddr, ":"); idx != -1 {
		return remoteAddr[:idx]
	}
	return remoteAddr
}

func generateFriendlyID() string {
	word := friendlyWords[rand.Intn(len(friendlyWords))]
	number := rand.Intn(100)
	return fmt.Sprintf("%s%d", word, number)
}

func createSession(w http.ResponseWriter, r *http.Request) {
	// Generate friendly session ID instead of UUID
	friendlyID := generateFriendlyID()
	now := time.Now()
	session := &Session{
		ID:        friendlyID,
		Users:     map[string]bool{getIP(r.RemoteAddr): true},
		CreatedAt: now,
		LastUsed:  now,
	}
	sessions.Store(friendlyID, session)

	sessionDir := filepath.Join(config.StorageDir, friendlyID)
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		http.Error(w, "Failed to create session directory", http.StatusInternalServerError)
		return
	}

	// Generate base URL
	baseURL := config.NgrokURL
	if baseURL == "" {
		baseURL = fmt.Sprintf("http://localhost:%d", config.Port)
	}
	sessionURL := fmt.Sprintf("%s/api/session/join?sessionID=%s", baseURL, friendlyID)

	// Generate QR code
	qrFile := filepath.Join(sessionDir, "qr.png")
	err := qrcode.WriteFile(sessionURL, qrcode.Medium, 256, qrFile)
	if err != nil {
		logger.Printf("Failed to generate QR code: %v", err)
	}

	if config.Verbose {
		logger.Printf("Session created: %s by %s", friendlyID, r.RemoteAddr)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"sessionID":    friendlyID,
		"sessionURL":   sessionURL,
		"qrCodeExists": err == nil,
	})
}

func joinSession(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("sessionID")
	if sessionID == "" {
		http.Error(w, "Session ID required", http.StatusBadRequest)
		return
	}

	value, ok := sessions.Load(sessionID)
	if !ok {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	session := value.(*Session)
	session.Users[getIP(r.RemoteAddr)] = true
	session.LastUsed = time.Now()

	if config.Verbose {
		logger.Printf("User %s joined session: %s", r.RemoteAddr, sessionID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"sessionID": sessionID,
	})
}

func serveQRCode(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("sessionID")
	if sessionID == "" {
		http.Error(w, "Session ID required", http.StatusBadRequest)
		return
	}

	value, ok := sessions.Load(sessionID)
	if !ok {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	session := value.(*Session)
	if !session.Users[getIP(r.RemoteAddr)] {
		http.Error(w, "Unauthorized access", http.StatusUnauthorized)
		return
	}

	session.LastUsed = time.Now()
	qrPath := filepath.Join(config.StorageDir, sessionID, "qr.png")

	if _, err := os.Stat(qrPath); os.IsNotExist(err) {
		http.Error(w, "QR code not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	http.ServeFile(w, r, qrPath)
}

func handleFiles(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("sessionID")
	if sessionID == "" {
		http.Error(w, "Session ID required", http.StatusBadRequest)
		return
	}

	value, ok := sessions.Load(sessionID)
	if !ok {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	session := value.(*Session)
	if !session.Users[getIP(r.RemoteAddr)] {
		http.Error(w, "Unauthorized access", http.StatusUnauthorized)
		return
	}

	// Update LastUsed
	session.LastUsed = time.Now()

	// Optional password protection
	if config.Password != "" {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "gshare" || !verifyPassword(pass, config.Password) {
			w.Header().Set("WWW-Authenticate", `Basic realm="Gshare"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	sessionDir := filepath.Join(config.StorageDir, sessionID)

	switch r.Method {
	case http.MethodGet:
		fileName := r.URL.Query().Get("file")
		if fileName == "" {
			listFiles(w, r, sessionDir)
		} else {
			downloadFile(w, r, sessionDir, fileName)
		}
	case http.MethodPost:
		uploadFile(w, r, sessionDir)
	case http.MethodDelete:
		deleteFile(w, r, sessionDir)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func listFiles(w http.ResponseWriter, r *http.Request, dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		http.Error(w, "Failed to read directory", http.StatusInternalServerError)
		return
	}

	files := []map[string]string{}
	for _, entry := range entries {
		if entry.Name() == "qr.png" { // Skip QR code file
			continue
		}
		info, _ := entry.Info()
		files = append(files, map[string]string{
			"name": entry.Name(),
			"size": formatSize(info.Size()),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(files)

	if config.Verbose {
		logger.Printf("Listed %d files for %s in session directory %s", len(files), r.RemoteAddr, dir)
	}
}

func downloadFile(w http.ResponseWriter, r *http.Request, dir, fileName string) {
	filePath := filepath.Join(dir, fileName)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, filePath)

	if config.Verbose {
		logger.Printf("File downloaded: %s from session directory %s by %s", fileName, dir, r.RemoteAddr)
	}
}

func uploadFile(w http.ResponseWriter, r *http.Request, dir string) {
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to get file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	destPath := filepath.Join(dir, header.Filename)
	out, err := os.Create(destPath)
	if err != nil {
		http.Error(w, "Failed to create file", http.StatusInternalServerError)
		return
	}
	defer out.Close()

	_, err = io.Copy(out, file)
	if err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	if config.Verbose {
		logger.Printf("File uploaded: %s to session directory %s by %s", header.Filename, dir, r.RemoteAddr)
	}

	w.Write([]byte("File uploaded successfully"))
}

func deleteFile(w http.ResponseWriter, r *http.Request, dir string) {
	fileName := r.URL.Query().Get("file")
	if fileName == "" {
		http.Error(w, "File name required", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join(dir, fileName)
	err := os.Remove(filePath)
	if err != nil {
		http.Error(w, "Failed to delete file", http.StatusInternalServerError)
		return
	}

	if config.Verbose {
		logger.Printf("File deleted: %s from session directory %s by %s", fileName, dir, r.RemoteAddr)
	}

	w.Write([]byte("File deleted successfully"))
}

func getNgrokURL() string {
	resp, err := http.Get("http://127.0.0.1:4040/api/tunnels")
	if err != nil {
		if config.Verbose {
			logger.Println("Could not fetch Ngrok URL:", err)
		}
		return ""
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		if config.Verbose {
			logger.Println("Failed to decode Ngrok tunnels:", err)
		}
		return ""
	}

	tunnels, ok := result["tunnels"].([]interface{})
	if !ok || len(tunnels) == 0 {
		if config.Verbose {
			logger.Println("No active Ngrok tunnels found")
		}
		return ""
	}

	tunnel, ok := tunnels[0].(map[string]interface{})
	if !ok {
		return ""
	}
	url, _ := tunnel["public_url"].(string)
	return url
}

func formatSize(size int64) string {
	const (
		B  = 1
		KB = 1024 * B
		MB = 1024 * KB
	)
	switch {
	case size < KB:
		return fmt.Sprintf("%d bytes", size)
	case size < MB:
		return fmt.Sprintf("%.2f KB", float64(size)/KB)
	default:
		return fmt.Sprintf("%.2f MB", float64(size)/MB)
	}
}

func verifyPassword(providedPassword, storedHash string) bool {
	if providedPassword == storedHash {
		return true
	}
	return bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(providedPassword)) == nil
}
