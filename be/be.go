package main

import (
	"fmt"
	"log"
	"net/http"
)

func printInfo(resp http.ResponseWriter, req *http.Request) {

	fmt.Fprintf(resp, "BACKEND Response")

}

func main() {
	http.HandleFunc("/", printInfo)
	log.Fatal(http.ListenAndServe(":5000", nil))
}
