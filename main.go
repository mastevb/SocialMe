package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"reflect"
	"strconv"

	"cloud.google.com/go/storage"
	"github.com/olivere/elastic"
)

const (
	POST_INDEX  = "post"
	DISTANCE    = "200km" // the default distance for Elasticsearch
	ES_URL      = "http://10.128.0.2:9200"
	BUCKET_NAME = "socialme-bucket" // GCS bucket name
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
	http.HandleFunc("/search", handlerSearch)
	// specify the program should listen on port 8080
	// nil -> DefaultServeMux
	// log error with log.Fatal
	log.Fatal(http.ListenAndServe(":8080", nil))
}

//////////////////////////////////////////////////////////////////////////////
// RPC Handlers

// handles post request sent by user
// constructs a Post object accordingly
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

// handles search request sent by user
// calls helper functions to read and convert results
func handlerSearch(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Received one request for search")
	w.Header().Set("Content-Type", "application/json")
	// get geo-location information
	// convert from string -> float
	lat, _ := strconv.ParseFloat(r.URL.Query().Get("lat"), 64)
	lon, _ := strconv.ParseFloat(r.URL.Query().Get("lon"), 64)
	// get range of search, set to default if not specified
	ran := DISTANCE
	if val := r.URL.Query().Get("range"); val != "" {
		ran = val + "km"
	}
	fmt.Println("range is ", ran)
	// set query
	query := elastic.NewGeoDistanceQuery("location")
	// set geo-location information
	query = query.Distance(ran).Lat(lat).Lon(lon)
	// get search result from helper function
	searchResult, err := readFromES(query, POST_INDEX)
	if err != nil {
		// handle HTTP error
		http.Error(w, "Failed to read post from Elasticsearch", http.StatusInternalServerError)
		fmt.Printf("Failed to read post from Elasticsearch %v.\n", err)
		return
	}
	// get Posts from helper function
	posts := getPostFromSearchResult(searchResult)
	// parse to JSON
	js, err := json.Marshal(posts)
	if err != nil {
		http.Error(w, "Failed to parse posts into JSON format", http.StatusInternalServerError)
		fmt.Printf("Failed to parse posts into JSON format %v.\n", err)
		return
	}
	// write to response
	w.Write(js)
}

//////////////////////////////////////////////////////////////////////////////
// Helper Functions

// reads the data from Elasticsearch
func readFromES(query elastic.Query, index string) (*elastic.SearchResult, error) {
	client, err := elastic.NewClient(elastic.SetURL(ES_URL))
	if err != nil {
		return nil, err
	}
	// get the search result from client
	searchResult, err := client.Search().
		Index(index).
		Query(query).
		Pretty(true).
		Do(context.Background())
	if err != nil {
		return nil, err
	}
	return searchResult, nil
}

// converts the search result to post information
func getPostFromSearchResult(searchResult *elastic.SearchResult) []Post {
	var ptype Post
	var posts []Post
	// extracts dynamic type information
	// access the underlying "Post" value
	for _, item := range searchResult.Each(reflect.TypeOf(ptype)) {
		p := item.(Post)
		posts = append(posts, p)
	}
	return posts
}

// saves post images to GCS
func saveToGCS(r io.Reader, objectName string) (string, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return "", err
	}
	bucket := client.Bucket(BUCKET_NAME)
	if _, err := bucket.Attrs(ctx); err != nil {
		return "", err
	}
	// start upload file
	object := bucket.Object(objectName)
	wc := object.NewWriter(ctx)
	if _, err := io.Copy(wc, r); err != nil {
		return "", err
	}
	if err := wc.Close(); err != nil {
		return "", err
	}
	// end upload file
	// set access control
	if err := object.ACL().Set(ctx, storage.AllUsers, storage.RoleReader); err != nil {
		return "", err
	}
	attrs, err := object.Attrs(ctx)
	if err != nil {
		return "", err
	}
	fmt.Printf("Image is saved to GCS: %s\n", attrs.MediaLink)
	return attrs.MediaLink, nil
}
