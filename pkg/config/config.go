package config

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"path/filepath"
	"sync"
)

type Config struct {
	WorkDir       string
	EnvType       string
	TogglAPIHost  string
	PipesAPIHost  string
	CorsWhitelist []string

	urls envUrls
	mx   sync.RWMutex
}

type envUrls struct {
	ReturnURL     map[string]string   `json:"return_url"`
	TogglAPIHost  map[string]string   `json:"toggl_api_host"`
	PipesAPIHost  map[string]string   `json:"pipes_api_host"`
	CorsWhitelist map[string][]string `json:"cors_whitelist"`
}

func Load(f *Flags) *Config {
	svc := &Config{
		urls: envUrls{
			ReturnURL:     map[string]string{},
			TogglAPIHost:  map[string]string{},
			PipesAPIHost:  map[string]string{},
			CorsWhitelist: map[string][]string{},
		},
	}

	b, err := ioutil.ReadFile(filepath.Join(f.WorkDir, "config", "urls.json"))
	if err != nil {
		log.Fatalf("Could not read urls.json, reason: %v", err)
	}
	if err := json.Unmarshal(b, &svc.urls); err != nil {
		log.Fatalf("Could not parse urls.json, reason: %v", err)
	}

	svc.EnvType = f.Environment
	svc.WorkDir = f.WorkDir
	svc.TogglAPIHost = svc.urls.TogglAPIHost[svc.EnvType]
	svc.PipesAPIHost = svc.urls.PipesAPIHost[svc.EnvType]
	svc.CorsWhitelist = svc.urls.CorsWhitelist[svc.EnvType]
	return svc
}
