package main

import (
	"github.com/aybabtme/canlii"
	"io/ioutil"
	"fmt"
	"time"
	"encoding/json"
)

// STEPS TO PERFORM
var createSearchResultsFile = false
var createIntersectionFile = true

func main() {
	// Initialize client
	apiKey, err := ioutil.ReadFile("apiKey.key")
	check(err)
	canliiClient, err := canlii.NewClient(nil, "", string(apiKey))
	check(err)

	// Get search results
	searchResults := canlii.SearchResult{}
	searchResultsFilename := "/home/pbrink/Cloud/Documents/School Documents/2018.01-04/SOEN 498/Project/outputFromCanLii/searchResults.json"
	if createSearchResultsFile {
		keyword := "sentencing"
		totalResultsToGet := 1000
		fmt.Printf("Searching canlii for '%s', max results %d\n", keyword, totalResultsToGet)
		searchResults, err = searchByKeyword(canliiClient, keyword, totalResultsToGet, 0)
		if err != nil {
			fmt.Printf("Error: %s\n", err)
		}
		resultsJson, _ := json.Marshal(searchResults)
		fmt.Printf("Saving %d cases and %d legislations to %s\n", len(searchResults.Cases), len(searchResults.Legislations), searchResultsFilename)
		err = ioutil.WriteFile(searchResultsFilename, resultsJson, 0644)
	} else {
		fmt.Printf("Reading search results from file at %s\n", searchResultsFilename)
		jsonBlob, err := ioutil.ReadFile(searchResultsFilename)
		check(err)
		err = json.Unmarshal(jsonBlob, &searchResults)
		check(err)
		fmt.Printf("Successfully read %d cases and %d legislations\n", len(searchResults.Cases), len(searchResults.Legislations))
	}

	if createIntersectionFile {
		// Get the intersection of the search results and the case databases that are interesting
		databases := CaseDatabases{}
		databasesFilename := "/home/pbrink/Cloud/Documents/School Documents/2018.01-04/SOEN 498/Project/outputFromCanLii/databases.json"
		fmt.Printf("Reading databases from file at %s\n", databasesFilename)
		jsonBlob, err := ioutil.ReadFile(databasesFilename)
		check(err)
		err = json.Unmarshal(jsonBlob, &databases)
		check(err)
		fmt.Printf("Successfully read %d databases\n", len(databases.DBs))
	}

}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func searchByKeyword(client *canlii.Client, keyword string, totalToGet int, initialOffset int) (canlii.SearchResult, error) {
	var resultCount int
	var numLoops int
	offset := initialOffset
	if totalToGet > 100 {
		resultCount = 100
		numLoops = totalToGet/100
	} else {
		resultCount = totalToGet
		numLoops = 1
	}
	options := &canlii.SearchOptions{
		ResultCount:resultCount,
		Offset:offset,
	}

	var cumulativeResults canlii.SearchResult

	for i := 0; i < numLoops; i++ {
		results, _, err := client.Search.Search(keyword, options)
		if err != nil {
			return cumulativeResults, err
		} else {
			if i == 0 {
				if results.TotalResults < totalToGet {
					totalToGet = results.TotalResults
					numLoops = totalToGet/100
				}
				cumulativeResults.TotalResults = results.TotalResults
			}
			cumulativeResults.Cases = append(cumulativeResults.Cases, results.Cases...)
			cumulativeResults.Legislations = append(cumulativeResults.Legislations, results.Legislations...)
		}
		// don't overrun api limits
		time.Sleep(time.Millisecond * 500)
	}
	return cumulativeResults, nil
}

type CaseDatabases struct {
	DBs []canlii.CaseDatabase `json:"caseDatabases"`
}