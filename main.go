package main

import (
	"github.com/aybabtme/canlii"
	"github.com/PuerkitoBio/goquery"
	"github.com/faiface/beep/wav"
	"io/ioutil"
	"fmt"
	"time"
	"encoding/json"
	"strings"
	"net/http"
	"regexp"
	"errors"
	"math/rand"
	"os"
	"github.com/faiface/beep/speaker"
)

// STEPS TO PERFORM
var createSearchResultsFile = false
var createIntersectionFile = false
var createCaseMetadataCollectionFile = false
var filterByKeyword = false
var downloadFullText = false
var generateLinkDocument = false
var generateFromDownloadedHtml = false


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
		totalResultsToGet := 20000
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

	// Read databases file
	databases := CaseDatabases{}
	databasesFilename := "/home/pbrink/Cloud/Documents/School Documents/2018.01-04/SOEN 498/Project/outputFromCanLii/databases.json"
	fmt.Printf("Reading databases from file at %s\n", databasesFilename)
	jsonBlob, err := ioutil.ReadFile(databasesFilename)
	check(err)
	err = json.Unmarshal(jsonBlob, &databases)
	check(err)
	fmt.Printf("Successfully read %d databases\n", len(databases.DBs))

	// Get the intersection of the search results and the case databases that are interesting
	interestingCases := canlii.SearchResult{}
	interestingCasesFilename := "/home/pbrink/Cloud/Documents/School Documents/2018.01-04/SOEN 498/Project/outputFromCanLii/interestingCases.json"
	if createIntersectionFile {
		// make a map of the databases
		databaseMap := make(map[string]string)
		for _, database := range databases.DBs {
			databaseMap[database.ID] = database.Name
		}
		// iterate through all cases in search results, and add any cases found that come from a desired database
		fmt.Println("Getting intersection of databases and search results")
		for _, caseRecord := range searchResults.Cases {
			if _, ok := databaseMap[caseRecord.DatabaseID]; ok {
				if caseRecord.ID.EN != "" {
					interestingCases.Cases = append(interestingCases.Cases, caseRecord)
				}
			}
		}
		resultsJson, _ := json.Marshal(interestingCases)
		fmt.Printf("Saving %d cases of interest to %s\n", len(interestingCases.Cases), interestingCasesFilename)
		err = ioutil.WriteFile(interestingCasesFilename, resultsJson, 0644)
	} else {
		fmt.Printf("Reading cases of interest from file at %s\n", interestingCasesFilename)
		jsonBlob, err := ioutil.ReadFile(interestingCasesFilename)
		check(err)
		err = json.Unmarshal(jsonBlob, &interestingCases)
		check(err)
		fmt.Printf("Successfully read %d cases of interest\n", len(interestingCases.Cases))
	}

	// Build the case metadata file
	caseMetadataCollection := CaseMetadataCollection{}
	caseMetadataCollectionFilename := "/home/pbrink/Cloud/Documents/School Documents/2018.01-04/SOEN 498/Project/outputFromCanLii/caseMetadataCollection.json"
	frenchCases := canlii.SearchResult{}
	frenchCasesFilename := "/home/pbrink/Cloud/Documents/School Documents/2018.01-04/SOEN 498/Project/outputFromCanLii/frenchCases.json"
	if createCaseMetadataCollectionFile {
		fmt.Printf("Getting case metadata for %d cases\n", len(interestingCases.Cases))
		for _, c := range interestingCases.Cases {
			fmt.Printf("\tCase %s, '%s' (%s)\n", c.ID.EN, c.Title, c.DatabaseID)
			result, _, err := canliiClient.CaseBrowse.CaseMetadata(c.DatabaseID, c.ID.EN)
			if err != nil {
				fmt.Printf("\t\tError: %s\n\t\tCase not found in English. Adding to french case list!\n", err)
				frenchCases.Cases = append(frenchCases.Cases, c)
			}
			if result != nil {
				caseMetadataCollection.Collection = append(caseMetadataCollection.Collection, result...)
			}
			time.Sleep(time.Millisecond * 300)
		}
		resultsJson, _ := json.Marshal(caseMetadataCollection)
		fmt.Printf("Saving %d cases' metadata to %s\n", len(caseMetadataCollection.Collection), caseMetadataCollectionFilename)
		err = ioutil.WriteFile(caseMetadataCollectionFilename, resultsJson, 0644)
		// save french cases to file
		resultsJson, _ = json.Marshal(frenchCases)
		fmt.Printf("Saving %d french to %s\n", len(frenchCases.Cases), frenchCasesFilename)
		err = ioutil.WriteFile(frenchCasesFilename, resultsJson, 0644)
	} else {
		fmt.Printf("Reading case metadata from file at %s\n", caseMetadataCollectionFilename)
		jsonBlob, err := ioutil.ReadFile(caseMetadataCollectionFilename)
		check(err)
		err = json.Unmarshal(jsonBlob, &caseMetadataCollection)
		check(err)
		fmt.Printf("Successfully read %d cases of interest\n", len(caseMetadataCollection.Collection))
	}

	// filter the cases of interest by keywords that have to do with sentencing
	sentencingCases := FullCaseCollection{}
	sentencingCasesFilename := "/home/pbrink/Cloud/Documents/School Documents/2018.01-04/SOEN 498/Project/outputFromCanLii/sentencingCases.json"
	if filterByKeyword {
		for _, c := range caseMetadataCollection.Collection {
			// if sentenc (sentencing, sentence) exists in the kewords, add to sentencing cases
			if strings.Contains(c.Keywords, "sentenc") {
				t := FullCase{}
				t.Metadata = c
				sentencingCases.Collection = append(sentencingCases.Collection, &t)
			}
		}
		resultsJson, _ := json.Marshal(sentencingCases)
		fmt.Printf("Saving %d sentencing-related cases to %s\n", len(sentencingCases.Collection), sentencingCasesFilename)
		err = ioutil.WriteFile(sentencingCasesFilename, resultsJson, 0644)
	} else {
		fmt.Printf("Reading sentencing cases from file at %s\n", sentencingCasesFilename)
		jsonBlob, err := ioutil.ReadFile(sentencingCasesFilename)
		check(err)
		err = json.Unmarshal(jsonBlob, &sentencingCases)
		check(err)
		fmt.Printf("Successfully read %d sentencing cases\n", len(sentencingCases.Collection))
	}

	if downloadFullText {
		for _, c := range sentencingCases.Collection {
			if c.FullText != "" {
				continue
			}
			fmt.Printf("Getting full text for %s: %s (%s)\n", c.Metadata.CaseID, c.Metadata.Title, c.Metadata.DatabaseID)
			fullText, err := downloadPage(c.Metadata.URL)
			if err != nil {
				f, _ := os.Open("/home/pbrink/go/src/sentencingAnalyzer/beep-01a.wav")
				s, format, _ := wav.Decode(f)
				speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
				speaker.Play(s)
				select {}
				break
			}
			whitespaceLeadingTrailing, err := regexp.Compile(`^[\s\p{Zs}]+|[\s\p{Zs}]+$`)
			whitespaceExtraInterior, err := regexp.Compile(`[\s\p{Zs}]{2,}`)
			fullText = whitespaceLeadingTrailing.ReplaceAllString(fullText, "")
			fullText = whitespaceExtraInterior.ReplaceAllString(fullText, "")
			c.FullText = fullText
			resultsJson, _ := json.Marshal(sentencingCases)
			//fmt.Printf("Saving %d sentencing-related cases to %s\n", len(sentencingCases.Collection), sentencingCasesFilename)
			err = ioutil.WriteFile(sentencingCasesFilename, resultsJson, 0644)
			//time.Sleep(time.Duration((rand.Int31n(120) + 30)) * time.Second)
			time.Sleep(time.Duration((rand.Int31n(12) + 3)) * time.Second)
		}
		fmt.Printf("Got full text for %d cases", len(sentencingCases.Collection))
	}

	if generateLinkDocument {
		var links []string
		for _, c := range sentencingCases.Collection {
			if c.FullText == "" {
				linkHtml := fmt.Sprintf("<li><a href=\"%s\">%s</a></li>", c.Metadata.URL, c.Metadata.CaseID)
				links = append(links, linkHtml)
			}
		}
		listLinks := strings.Join(links, "\n")
		linksFilename := "/home/pbrink/Cloud/Documents/School Documents/2018.01-04/SOEN 498/Project/outputFromCanLii/links.txt"
		err = ioutil.WriteFile(linksFilename, []byte(listLinks), 0644)
	}

	if generateFromDownloadedHtml {
		generateFromHtmlOnDisk(&sentencingCases)
		var count int
		for _, c := range sentencingCases.Collection {
			if c.FullText != "" {
				count++
			} else {
				fmt.Println(c.Metadata.URL)
			}
		}
		resultsJson, _ := json.Marshal(sentencingCases)
		err = ioutil.WriteFile(sentencingCasesFilename, resultsJson, 0644)
		fmt.Printf("%d cases now contain full text\n", count)
	}
}

func downloadPage(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if (resp.StatusCode != 200) {
		fmt.Printf("Http request error: %d %s\n", resp.StatusCode, resp.Status)
		return "", errors.New(resp.Status)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", err
	}
	return doc.Find(".documentcontent").Text(), nil
}

func generateFromHtmlOnDisk(collection *FullCaseCollection) {
	directory := "/home/pbrink/Cloud/Documents/School Documents/2018.01-04/SOEN 498/Project/canliiHtml/"
	files, _ := ioutil.ReadDir(directory)
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		caseId := file.Name()
		caseId = strings.Replace(caseId, "CanLII - ", "", -1)
		caseId = strings.Replace(caseId, " (CanLII).html", "", -1)
		dbInParens, _ := regexp.Compile(` \(.*\)\.html`)
		caseId = dbInParens.ReplaceAllString(caseId, "")
		caseId = strings.Replace(caseId, " ", "", -1)
		caseId = strings.ToLower(caseId)

		for _, c := range collection.Collection {
			if c.Metadata.CaseID == caseId {
				if c.FullText != "" {
					break
				}
				//contents, _ := ioutil.ReadFile(directory + file.Name())
				r, _ := os.Open(directory + file.Name())
				doc, _ := goquery.NewDocumentFromReader(r)
				fullText := doc.Find("div.documentcontent").Text()
				whitespaceLeadingTrailing, _ := regexp.Compile(`^[\s\p{Zs}]+|[\s\p{Zs}]+$`)
				whitespaceExtraInterior, _ := regexp.Compile(`[\s\p{Zs}]{2,}`)
				fullText = whitespaceLeadingTrailing.ReplaceAllString(fullText, "")
				fullText = whitespaceExtraInterior.ReplaceAllString(fullText, "")
				c.FullText = fullText
				break
			}
		}
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
		options.Offset = options.Offset + options.ResultCount
		// don't overrun api limits
		time.Sleep(time.Millisecond * 500)
	}
	return cumulativeResults, nil
}

type FullCaseCollection struct {
	Collection []*FullCase `json:"fullCases"`
}

type FullCase struct {
	Metadata canlii.CaseMetadata `json:"caseMetadata"`
	FullText string              `json:"fullText"`
}

type CaseMetadataCollection struct {
	Collection []canlii.CaseMetadata `json:"caseMetadataCollection"`
}

type CaseDatabases struct {
	DBs []*canlii.CaseDatabase `json:"caseDatabases"`
}