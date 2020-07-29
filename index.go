package main

import (
	"context"
	"fmt"

	"github.com/olivere/elastic"
)

const (
	POST_INDEX = "post"
	ES_URL     = "http://10.128.0.2:9200" // GCE internal IP
)

// create indexes in Elasticsearch
func main() {
	// create a new client to Elasticsearch
	client, err := elastic.NewClient(elastic.SetURL(ES_URL))
	if err != nil {
		panic(err)
	}
	// check if the index exists
	exists, err := client.IndexExists(POST_INDEX).Do(context.Background())
	if err != nil {
		panic(err)
	}
	if !exists {
		mapping := `{
			"mappings": {
				"properties": {
					"user":     { "type": "keyword", "index": false },
					"message":  { "type": "keyword", "index": false },
					"location": { "type": "geo_point" },
					"url":      { "type": "keyword", "index": false },
					"type":     { "type": "keyword", "index": false },
					"face":     { "type": "float" }
 				}
			}
		}`
	}
	// create the new index
	_, err := client.CreateIndex(POST_INDEX).Body(mapping).Do(context.Background())
	if err != nil {
		panic(err)
	}
	fmt.Println("Post index is created")
}
