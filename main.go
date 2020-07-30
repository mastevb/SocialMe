package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"reflect"
	"strconv"

	"cloud.google.com/go/storage"
	"github.com/olivere/elastic"
	"github.com/pborman/uuid"
)

const (
	POST_INDEX  = "post"
	DISTANCE    = "200km" // the default distance for Elasticsearch
	ES_URL      = "http://10.128.0.2:9200"
	BUCKET_NAME = "socialme-bucket" // GCS bucket name
)

// media types
var (
	mediaTypes = map[string]string{
		".jpeg": "image",
		".jpg":  "image",
		".gif":  "image",
		".png":  "image",
		".mov":  "video",
		".mp4":  "video",
		".avi":  "video",
		".flv":  "video",
		".wmv":  "video",
	}
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
	// tell the http package to handle all requests to the
	// web root with handler
	http.HandleFunc("/post", handlerPost)
	http.HandleFunc("/search", handlerSearch)
	http.HandleFunc("/cluster", handlerCluster)
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
	fmt.Println("Received one request")
	w.Header().Set("Content-Type", "application/json")
	lat, _ := strconv.ParseFloat(r.FormValue("lat"), 64)
	lon, _ := strconv.ParseFloat(r.FormValue("lon"), 64)
	// construct post
	p := &Post{
		User:    r.FormValue("user"),
		Message: r.FormValue("message"),
		Location: Location{
			Lat: lat,
			Lon: lon,
		},
	}
	file, header, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "Image is not available", http.StatusBadRequest)
		fmt.Printf("Image is not available %v\n", err)
		return
	}
	// get media type
	suffix := filepath.Ext(header.Filename)
	if t, ok := mediaTypes[suffix]; ok {
		p.Type = t
	} else {
		p.Type = "unknown"
	}
	id := uuid.New()
	// save the media to GCS
	mediaLink, err := saveToGCS(file, id)
	if err != nil {
		http.Error(w, "Failed to save image to GCS", http.StatusInternalServerError)
		fmt.Printf("Failed to save image to GCS %v\n", err)
		return
	}
	p.Url = mediaLink
	if p.Type == "image" { // detect faces
		uri := fmt.Sprintf("gs://%s/%s", BUCKET_NAME, id)
		if score, err := annotate(uri); err != nil {
			http.Error(w, "Failed to annotate image", http.StatusInternalServerError)
			fmt.Printf("Failed to annotate the image %v\n", err)
			return
		} else {
			p.Face = score // assign confidence score
		}
	}
	err = saveToES(p, POST_INDEX, id)
	if err != nil {
		http.Error(w, "Failed to save post to Elasticsearch", http.StatusInternalServerError)
		fmt.Printf("Failed to save post to Elasticsearch %v\n", err)
		return
	}
}

// handle search request sent by user
// call helper functions to read and convert results
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

// handle cluster requests
// ex. /cluser?term=face -> images with faces
func handlerCluster(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Received one cluster request")
	w.Header().Set("Content-Type", "application/json")
	term := r.URL.Query().Get("term")
	query := elastic.NewRangeQuery(term).Gte(0.9) // confidence level
	searchResult, err := readFromES(query, POST_INDEX)
	if err != nil {
		http.Error(w, "Failed to read from Elasticsearch", http.StatusInternalServerError)
		return
	}
	posts := getPostFromSearchResult(searchResult)
	js, err := json.Marshal(posts)
	if err != nil {
		http.Error(w, "Failed to parse post object", http.StatusInternalServerError)
		fmt.Printf("Failed to parse post object %v\n", err)
		return
	}
	w.Write(js)
}

//////////////////////////////////////////////////////////////////////////////
// Helper Functions

// read the data from Elasticsearch
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

// convert the search result to post information
func getPostFromSearchResult(searchResult *elastic.SearchResult) []Post {
	var ptype Post
	var posts []Post
	// extract dynamic type information
	// access the underlying "Post" value
	for _, item := range searchResult.Each(reflect.TypeOf(ptype)) {
		p := item.(Post)
		posts = append(posts, p)
	}
	return posts
}

// save post images to GCS
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

// save a user post to Elasticsearch
func saveToES(post *Post, index string, id string) error {
	client, err := elastic.NewClient(elastic.SetURL(ES_URL))
	if err != nil {
		return err
	}
	_, err = client.Index().
		Index(index).
		Id(id).
		BodyJson(post).
		Do(context.Background())
	if err != nil {
		return err
	}
	fmt.Printf("Post is saved to index: %s\n", post.Message)
	return nil
}
