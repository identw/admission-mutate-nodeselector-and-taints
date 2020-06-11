package main

import (
	"fmt"
	"os"
	"html"
	"io/ioutil"
	"log"
	"net/http"
	"time"
	"crypto/tls"
	"encoding/json"
	mutate "github.com/identw/admission-mutate-nodeselector-and-taints/pkg/mutate"
)

var m mutate.Mutate

func handleRoot(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "hello %q", html.EscapeString(r.URL.Path))
}

func handleMutate(w http.ResponseWriter, r *http.Request) {

	// read the body / request
	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%s", err)
	}

	// mutate the request
	mutated, err := m.Mutate(body, true)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%s", err)
	}

	// and write it back
	w.WriteHeader(http.StatusOK)
	w.Write(mutated)
}

func main() {

	PORT := os.Getenv("PORT")
	if PORT == "" {
		PORT = "8443"
	}
	tlsCert := os.Getenv("TLS_CERT")
	tlsKey := os.Getenv("TLS_KEY")

	mutateOptions := mutate.Mutate{}
	if err := json.Unmarshal([]byte(os.Getenv("MUTATE_OPTIONS")), &mutateOptions); err != nil {
		fmt.Fprintf(os.Stderr, "MUTATE_OPTIONS parse error: %s\n", err)
		os.Exit(1)
	}
	m.NodeSelector = mutateOptions.NodeSelector
	m.Tolerations = mutateOptions.Tolerations
	m.RemoveNodeAffinity = mutateOptions.RemoveNodeAffinity

	mux := http.NewServeMux()

	mux.HandleFunc("/", handleRoot)
	mux.HandleFunc("/mutate", handleMutate)

	cert, err := tls.X509KeyPair([]byte(tlsCert), []byte(tlsKey))
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	  }

	s := &http.Server{
		Addr:           ":" + PORT,
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1048576
		TLSConfig: tlsConfig,

	}

	log.Fatal(s.ListenAndServeTLS("", ""))

}