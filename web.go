package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/elazarl/go-bindata-assetfs"
	"github.com/gorilla/mux"
)

var (
	webConfiguration *Configuration
)

type WebProvider struct {
	Address           string
	CertFile, KeyFile string
}

type Page struct {
	Configuration Configuration
}

func (provider *WebProvider) Provide(configurationChan chan<- configMessage) {
	systemRouter := mux.NewRouter()
	systemRouter.Methods("GET").Path("/").Handler(http.HandlerFunc(GetHTMLConfigHandler))
	systemRouter.Methods("GET").Path("/health").Handler(http.HandlerFunc(GetHealthHandler))
	systemRouter.Methods("GET").Path("/api").Handler(http.HandlerFunc(GetConfigHandler))
	systemRouter.Methods("PUT").Path("/api").Handler(http.HandlerFunc(
		func(rw http.ResponseWriter, r *http.Request) {
			configuration := new(Configuration)
			b, _ := ioutil.ReadAll(r.Body)
			err := json.Unmarshal(b, configuration)
			if err == nil {
				configurationChan <- configMessage{"web", configuration}
				GetConfigHandler(rw, r)
			} else {
				log.Errorf("Error parsing configuration %+v", err)
				http.Error(rw, fmt.Sprintf("%+v", err), http.StatusBadRequest)
			}
		}))
	systemRouter.Methods("GET").Path("/api/backends").Handler(http.HandlerFunc(GetBackendsHandler))
	systemRouter.Methods("GET").Path("/api/backends/{backend}").Handler(http.HandlerFunc(GetBackendHandler))
	systemRouter.Methods("GET").Path("/api/backends/{backend}/servers").Handler(http.HandlerFunc(GetServersHandler))
	systemRouter.Methods("GET").Path("/api/backends/{backend}/servers/{server}").Handler(http.HandlerFunc(GetServerHandler))
	systemRouter.Methods("GET").Path("/api/frontends").Handler(http.HandlerFunc(GetFrontendsHandler))
	systemRouter.Methods("GET").Path("/api/frontends/{frontend}").Handler(http.HandlerFunc(GetFrontendHandler))
	systemRouter.Methods("GET").PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(&assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, Prefix: "static"})))

	go func() {
		if len(provider.CertFile) > 0 && len(provider.KeyFile) > 0 {
			err := http.ListenAndServeTLS(provider.Address, provider.CertFile, provider.KeyFile, systemRouter)
			if err != nil {
				log.Fatal("Error creating server: ", err)
			}
		} else {
			err := http.ListenAndServe(provider.Address, systemRouter)
			if err != nil {
				log.Fatal("Error creating server: ", err)
			}
		}
	}()
}

func GetConfigHandler(rw http.ResponseWriter, r *http.Request) {
	templatesRenderer.JSON(rw, http.StatusOK, webConfiguration)
}

func GetHTMLConfigHandler(response http.ResponseWriter, request *http.Request) {
	var cfg Configuration
	cfg.Backends = make(map[string]*Backend)
	cfg.Frontends = make(map[string]*Frontend)

	// Quick and dirty merge of config for display
	for _, config := range currentConfigurations {
		for name, config := range config.Backends {
			cfg.Backends[name] = config
		}

		for name, config := range config.Frontends {
			cfg.Frontends[name] = config
		}
	}

	templatesRenderer.HTML(response, http.StatusOK, "configuration", Page{Configuration: cfg})
}

func GetHealthHandler(rw http.ResponseWriter, r *http.Request) {
	templatesRenderer.JSON(rw, http.StatusOK, metrics.Data())
}

func GetBackendsHandler(rw http.ResponseWriter, r *http.Request) {
	templatesRenderer.JSON(rw, http.StatusOK, webConfiguration.Backends)
}

func GetBackendHandler(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["backend"]
	if backend, ok := webConfiguration.Backends[id]; ok {
		templatesRenderer.JSON(rw, http.StatusOK, backend)
	} else {
		http.NotFound(rw, r)
	}
}

func GetFrontendsHandler(rw http.ResponseWriter, r *http.Request) {
	templatesRenderer.JSON(rw, http.StatusOK, webConfiguration.Frontends)
}

func GetFrontendHandler(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["frontend"]
	if frontend, ok := webConfiguration.Frontends[id]; ok {
		templatesRenderer.JSON(rw, http.StatusOK, frontend)
	} else {
		http.NotFound(rw, r)
	}
}

func GetServersHandler(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	backend := vars["backend"]
	if backend, ok := webConfiguration.Backends[backend]; ok {
		templatesRenderer.JSON(rw, http.StatusOK, backend.Servers)
	} else {
		http.NotFound(rw, r)
	}
}

func GetServerHandler(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	backend := vars["backend"]
	server := vars["server"]
	if backend, ok := webConfiguration.Backends[backend]; ok {
		if server, ok := backend.Servers[server]; ok {
			templatesRenderer.JSON(rw, http.StatusOK, server)
		} else {
			http.NotFound(rw, r)
		}
	} else {
		http.NotFound(rw, r)
	}
}
