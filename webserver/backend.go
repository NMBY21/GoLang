package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/", HelloServer)
	http.ListenAndServe(":8081", nil)
}

func HelloServer(w http.ResponseWriter, r *http.Request) {
	fmt.Println("got response....")
	http.Redirect(w, r, "https://go.dev/doc/", http.StatusSeeOther)
}
