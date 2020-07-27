package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// a representation of the location of a post
type Location struct {
	// `json:"lat"` is for the json parsing of the field
	// Otherwise, it will be defaulted to 'Lat'
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// a representation of a post
type Post struct {
	// the user of the post
	User string `json:"user"`
	// the message with the post
	Message string `json:"message"`
	// the location of the post
	Location Location `json:"location"`
	// the url of the image/video
	Url string `json:"url"`
	// the type of the post, image/video
	Type string `json:"type"`
	// for face recognition
	Face float32 `json:"face"`
}

func main() {
	fmt.Println("started-service")
	// tells the http package to handle all requests to the
	// web root with handler
	http.HandleFunc("/post", handlerPost)
	// specify the program should listen on port 8080
	// nil -> DefaultServeMux
	// log error with log.Fatal
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// handles post request sent by user
// construct a Post object accordingly
func handlerPost(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Receieved one post request")
	decoder := json.NewDecoder(r.Body)
	var p Post
	// if an error was encountered
	if err := decoder.Decode(&p); err != nil {
		panic(err)
	}
	fmt.Fprintf(w, "Post received: %s\n", p.Message)
}
