package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
)

var SecretGreeting string

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("got / request from %s\n", r.RemoteAddr)
		io.WriteString(w, fmt.Sprintf("%s %s", greeting(), SecretGreeting))
	})

	err := http.ListenAndServe(":8080", nil)
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("server closed\n")
	} else if err != nil {
		fmt.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}
}

func greeting() string {
	return "Greetings"
}
