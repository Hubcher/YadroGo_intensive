package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Port int    `yaml:"port" env:"FILESERVER_PORT" env-default:"8080"`
	Dir  string `yaml:"dir"  env:"FILESERVER_DIR"  env-default:"/data"`
}

func loadConfig() (Config, error) {
	var config Config
	var configPath string

	flag.StringVar(&configPath, "config", "", "path to config file (yaml)")
	flag.Parse()

	if configPath != "" {
		if err := cleanenv.ReadConfig(configPath, &config); err != nil {
			return config, fmt.Errorf("read config from %s: %w", configPath, err)
		}
		return config, nil
	}
	if _, err := os.Stat("/config.yaml"); err == nil {
		if err := cleanenv.ReadConfig("/config.yaml", &config); err != nil {
			return config, fmt.Errorf("read config from /config.yaml: %w", err)
		}
		return config, nil
	}
	if err := cleanenv.ReadEnv(&config); err != nil {
		return config, fmt.Errorf("read env: %w", err)
	}
	return config, nil
}

func ensureDir(dir string) error {
	return os.MkdirAll(dir, 0o755)
}

func sanitizeName(name string) (string, error) {

	name = strings.TrimSpace(name)

	if name == "" {
		return "", fmt.Errorf("empty filename")
	}

	// Например: file/name.txt
	if strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return "", fmt.Errorf("invalid filename: %s", name)
	}

	// исходное имя не сходится с проверкой base и clean
	cleanName := filepath.Clean(name)
	baseName := filepath.Base(cleanName)

	if name != baseName {
		return "", fmt.Errorf("invalid filename: %s", name)
	}

	return name, nil

}

func writeFile(dstPath string, r io.Reader, perm os.FileMode) error {
	// Best practice запись через временный файл
	tmp := dstPath + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}

	_, copyErr := io.Copy(f, r)
	closeErr := f.Close()

	if copyErr != nil {
		_ = os.Remove(tmp)
		return copyErr
	}

	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}

	return os.Rename(tmp, dstPath)

}

func readMultipartFile(r *http.Request) (file multipart.File, header *multipart.FileHeader, err error) {

	ct := r.Header.Get("Content-Type")
	mediaType, _, ctErr := mime.ParseMediaType(ct)

	if ctErr != nil || !strings.HasPrefix(mediaType, "multipart/") {
		return nil, nil, fmt.Errorf("invalid Content-Type: %s", ct)
	}
	f, h, err := r.FormFile("file")
	return f, h, err

}

type server struct {
	root string
}

func (s *server) handleCollection(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case http.MethodGet:

		entries, err := os.ReadDir(s.root)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		names := make([]string, 0, len(entries))

		for _, entry := range entries {
			if entry.Type().IsRegular() { // true на файлы, исключая directory
				names = append(names, entry.Name())
			}
		}

		sort.Strings(names)

		for _, name := range names {
			_, _ = io.WriteString(w, name+"\n")
		}

	case http.MethodPost:
		f, header, err := readMultipartFile(r)
		if err != nil {
			http.Error(w, "bad multipart", http.StatusBadRequest)
			return
		}
		defer func() {
			if err := f.Close(); err != nil {
				log.Printf("close error: %v", err)
			}
		}()

		name, err := sanitizeName(header.Filename)
		if err != nil {
			http.Error(w, "bad filename", http.StatusBadRequest)
			return
		}

		dst := filepath.Join(s.root, name)
		if _, err := os.Stat(dst); err == nil {
			// Уже есть, возвращаем по заданию
			w.WriteHeader(http.StatusConflict)
			return
		}

		if err := writeFile(dst, f, 0o644); err != nil {
			http.Error(w, "write error", http.StatusInternalServerError)
			return
		}

		// По заданию
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, name)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *server) handleItem(w http.ResponseWriter, r *http.Request) {
	// путь вида /files/filename — выковыриваем имя после "/files/"
	// ServeMux с префиксом "/files/" даст нам полный r.URL.Path, берём base
	name := strings.TrimPrefix(r.URL.Path, "/files/")
	name = path.Clean(name)

	if name == "" || name == "." || name == "/" {
		http.Error(w, "bad filename", http.StatusBadRequest)
		return
	}

	name, err := sanitizeName(name)
	if err != nil {
		http.Error(w, "bad filename", http.StatusBadRequest)
		return
	}

	full := filepath.Join(s.root, name)

	switch r.Method {
	case http.MethodGet:
		f, err := os.Open(full)
		if err != nil {
			if os.IsNotExist(err) {
				// По заданию
				http.NotFound(w, r)
				return
			}
			http.Error(w, "open error", http.StatusInternalServerError)
			return
		}
		defer func() {
			if err := f.Close(); err != nil {
				log.Printf("close error: %v", err)
			}
		}()

		if _, err := io.Copy(w, f); err != nil {
			return
		}

	case http.MethodPut:
		// Файл должен существовать
		if _, err := os.Stat(full); err != nil {
			if os.IsNotExist(err) {
				http.NotFound(w, r)
				return
			}
			http.Error(w, "stat error", http.StatusInternalServerError)
			return
		}
		fup, _, err := readMultipartFile(r)
		if err != nil {
			http.Error(w, "bad multipart", http.StatusBadRequest)
			return
		}

		defer func() {
			if err := fup.Close(); err != nil {
				log.Printf("close error: %v", err)
			}
		}()

		if err := writeFile(full, fup, 0o644); err != nil {
			http.Error(w, "write error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	case http.MethodDelete:
		// идемпотентно: всегда 200
		if err := os.Remove(full); err != nil && !os.IsNotExist(err) {
			http.Error(w, "delete error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)

	}

}

func main() {
	config, err := loadConfig()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	if err := ensureDir(config.Dir); err != nil {
		log.Fatalf("cannot create directory %s: %v", config.Dir, err)
	}

	s := &server{root: config.Dir}

	mux := http.NewServeMux()
	mux.HandleFunc("/files", s.handleCollection) // GET, POST
	mux.HandleFunc("/files/", s.handleItem)      // GET, PUT, DELETE

	addr := fmt.Sprintf(":%d", config.Port)
	log.Printf("fileserver listening on %s, dir=%s", addr, config.Dir)
	log.Fatal(http.ListenAndServe(addr, mux))
}
