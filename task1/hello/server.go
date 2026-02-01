package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Port int `yaml:"port" env:"HELLO_PORT" env-default:"8888"`
}

func loadConfig() (Config, error) {

	var config Config
	var configPath string

	flag.StringVar(&configPath, "config", "", "Path to config file (yaml)")
	flag.Parse()

	// Если указан -config=..., читаем его
	if configPath != "" {
		if err := cleanenv.ReadConfig(configPath, &config); err != nil {
			return config, fmt.Errorf("read config from %s: %w", configPath, err)
		}
		return config, nil
	}

	// Флаг не задан. Делаем через config.yaml
	if _, err := os.Stat("/config.yaml"); err == nil {
		if err := cleanenv.ReadConfig("/config.yaml", &config); err != nil {
			return config, fmt.Errorf("read config from /config.yaml: %w", err)
		}
		return config, nil
	}

	// Ни флага, ни файла
	if err := cleanenv.ReadEnv(&config); err != nil {
		return config, fmt.Errorf("read config from env: %w", err)
	}

	return config, nil

}

func pingHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("pong\n"))

}

func helloHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// /hello?name=Misha
	v := r.URL.Query() // map key : value
	name := v.Get("name")

	if name == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("empty name\n"))
		return
	}

	w.WriteHeader(http.StatusOK)

	if _, err := io.WriteString(w, "Hello, "+name+"!\n"); err != nil {
		log.Printf("write error: %v", err)
	}
}

func main() {

	config, err := loadConfig()

	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ping", pingHandler)
	mux.HandleFunc("/hello", helloHandler)

	// Example :8080
	addr := fmt.Sprintf(":%d", config.Port)
	log.Printf("listening on %s", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}

}
