package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
)

// ------------------ CONFIGURATION & COMMAND LINE ------------------

// Joplins server address.
var JOPLIN_SERVER string

// Joplins access token.
var JOPLIN_TOKEN string

// Host and port for the peeker server.
var PEEKER_HOST string
var PEEKER_PORT string

const DEFAULT_JOPLIN_SERVER = "http://localhost:41184"
const DEFAULT_PEEKER_HOST = "127.0.0.1"
const DEFAULT_PEEKER_PORT = "8080"

const CONFIG_FILE = "config.json"

var VERBOSE bool = false

// Reads the configuration from a JSON file.
// The file should have the following structure:
//
//	{
//	  "joplin_server": "http://<your_joplin_host>:<joplin_port>",
//	  "joplin_token": "<your_joplin_token>",
//	  "peeker_host": "<your_host>"
//	  "peeker_port": "<your_port>"
//	}
//
// Returns:
// A map with the configuration values.
func readConfigJSON() (map[string]string, error) {
	// Read the config file
	file, err := os.Open(CONFIG_FILE)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Read the file content
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	// Parse the JSON content
	var config map[string]string
	err = json.Unmarshal(content, &config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

// Prints the usage of the program.
func usage() {
	fmt.Printf("\nUsage: " + filepath.Base(os.Args[0]) + " [options]\n")
	fmt.Printf("\nRuns the peeker server. The following options are available:\n\n")
	flag.PrintDefaults()
	fmt.Printf("\n1. If an option is not provided, the program will look for the\n" +
		"corresponding environment variable (JOPLIN_SERVER, JOPLIN_TOKEN,\n" +
		"PEEKER_HOST, PEEKER_PORT).\n\n" +
		"2. If the environment variable is not set, the program will\n" +
		"try to read " + CONFIG_FILE + " file in the current directory.\n" +
		"The file should be a JSON file with the following structure:\n" +
		"{\n" +
		"  \"joplin_server\": \"http://<your_joplin_host>:<joplin_port>\",\n" +
		"  \"joplin_token\": \"<your_joplin_token>\",\n" +
		"  \"peeker_host\": \"<your_host>\"\n" +
		"  \"peeker_port\": \"<your_port>\"\n" +
		"}\n\n" +
		"3. Finally if the file is not found or the JSON is invalid, the program will\n" +
		"fall back to the default values.\n\n" +
		"NOTE: There's no default value provided for JOPLIN_TOKEN, so it must be\n" +
		"provided either through an option, an environment variable or the config file.\n")
}

// Initializes the program by parsing the command line arguments, environment,
// config file and default values. It sets the JOPLIN_SERVER, JOPLIN_TOKEN,
// PEEKER_HOST and PEEKER_PORT.
func init() {
	flag.StringVar(&JOPLIN_SERVER, "joplin", "", "Joplin server address and port (default: "+DEFAULT_JOPLIN_SERVER+")")
	flag.StringVar(&JOPLIN_TOKEN, "token", "", "Joplin access token")
	flag.StringVar(&PEEKER_HOST, "host", "", "Listen address for the peeker server (default: "+DEFAULT_PEEKER_HOST+")")
	flag.StringVar(&PEEKER_PORT, "port", "", "Listen port for the peeker server (default: "+DEFAULT_PEEKER_PORT+")")
	flag.BoolVar(&VERBOSE, "v", false, "Verbose mode (default: false)")
	flag.Parse()

	if env, exist := os.LookupEnv("JOPLIN_SERVER"); exist && JOPLIN_SERVER == "" {
		JOPLIN_SERVER = env
	}
	if env, exist := os.LookupEnv("JOPLIN_TOKEN"); exist && JOPLIN_TOKEN == "" {
		JOPLIN_TOKEN = env
	}
	if env, exist := os.LookupEnv("PEEKER_HOST"); exist && PEEKER_HOST == "" {
		PEEKER_HOST = env
	}
	if env, exist := os.LookupEnv("PEEKER_PORT"); exist && PEEKER_PORT == "" {
		PEEKER_PORT = env
	}

	if config, err := readConfigJSON(); err == nil {
		if JOPLIN_SERVER == "" {
			JOPLIN_SERVER = config["joplin_server"]
		}
		if JOPLIN_TOKEN == "" {
			JOPLIN_TOKEN = config["joplin_token"]
		}
		if PEEKER_HOST == "" {
			PEEKER_HOST = config["peeker_host"]
		}
		if PEEKER_PORT == "" {
			PEEKER_PORT = config["peeker_port"]
		}
	}

	if JOPLIN_SERVER == "" {
		JOPLIN_SERVER = DEFAULT_JOPLIN_SERVER
	}
	if PEEKER_HOST == "" {
		PEEKER_HOST = DEFAULT_PEEKER_HOST
	}
	if PEEKER_PORT == "" {
		PEEKER_PORT = DEFAULT_PEEKER_PORT
	}

	if JOPLIN_TOKEN == "" {
		fmt.Println("Error: JOPLIN_TOKEN is missing")
		usage()
		os.Exit(1)
	}
}

// Note represents a note in the Joplin server.
type Note struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// NotesResponse represents the JSON response from the Joplin server for notes.
type NotesResponse struct {
	Items   []Note `json:"items"`
	HasMore bool   `json:"has_more"`
}

// Notebook represents a notebook in the Joplin server.
type Notebook struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	ParentID string `json:"parent_id"`
}

// NotebooksResponse represents the JSON response from the Joplin server for notebooks.
type NotebooksResponse struct {
	Items   []Notebook `json:"items"`
	HasMore bool       `json:"has_more"`
}

// NotebookNode represents a node in a notebook tree.
type NotebookNode struct {
	ID       string         `json:"id"`
	Title    string         `json:"title"`
	ParentID string         `json:"parent_id"`
	NumNotes int            `json:"n_children"`
	Children []NotebookNode `json:"children"`
}

// It sets up the HTTP server and starts listening on port 8080.
func main() {
	// Config server handlers.
	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/id/", noteContentHandler)
	http.HandleFunc("/image/", imageHandler)
	http.HandleFunc("/favicon.ico", iconHandler)
	http.HandleFunc("/search/", searchHandler)
	http.HandleFunc("/notebooks/", notebooksHandler)
	http.Handle("/static/",
		http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Start peeker_server.
	peeker_server := fmt.Sprintf("%s:%s", PEEKER_HOST, PEEKER_PORT)
	fmt.Printf("Peeker Server listening on %s\n", peeker_server)
	fmt.Printf("Forwarding requests to Joplin Server on %s\n", JOPLIN_SERVER)
	log.Fatal(http.ListenAndServe(peeker_server, nil))
}

// --------------------- HELPER FUNCTIONS ---------------------

// Modifies the markdown to replace Joplin markdown links with peeker server
// links.
func modifyMarkdown(markdown string) string {
	// Replace Joplin markdown image with peeker server links.
	// ![text](/:/id) to <img src="/image/<id>" alt="text" />
	imgLinksRE := regexp.MustCompile(`!\[(.*?)\]\(:/(.*?)\)`)
	replacement := "<img src='/image/${2}' alt='${1}'/>"
	markdown = imgLinksRE.ReplaceAllString(markdown, replacement)

	// Replace Joplin markdown note links with peeker server links.
	// [text](/:/id) to [text](/id/<id>)
	linksRE := regexp.MustCompile(`([^!])\[(.*?)\]\(:/(.*?)\)`)
	replacement = "${1}[${2}](/id/${3})"
	markdown = linksRE.ReplaceAllString(markdown, replacement)

	// Replace Joplin markdown html img tags with peeker server links.
	// <img src="/:id" /> to <img src="/image/<id>" />
	imgSrcRE := regexp.MustCompile(`<img(.*?)src=["']:/(.*?)["'](.*?)/>`)
	replacement = "<img src='/image/${2}'${3}/>"
	markdown = imgSrcRE.ReplaceAllString(markdown, replacement)

	return markdown
}

// Logs a message if the verbose flag is set.
//
// Parameters:
//
//	logMsg: The log message to print.
//	args: The arguments to the log message if any.
func logInfo(logMsg string, args ...interface{}) {
	if VERBOSE {
		if args == nil {
			log.Println("INFO: " + logMsg)
		} else {
			log.Printf("INFO: "+logMsg, args...)
		}
	}
}

// Logs an error message.
// Parameters:
//
//	logMsg: The log message to print.
//	err: The error that occurred.
//	args: The arguments to the log message if any.
func logError(logMsg string, err error, args ...interface{}) {
	if args == nil {
		log.Printf("ERROR: "+logMsg, err)
	} else {
		allArgs := append([]interface{}{err}, args...)
		log.Printf("ERROR: "+logMsg, allArgs...)
	}
}

// Logs an internal error and returns an HTTP 500 status code to the client.
//
// Parameters:
//
//	logMsg: The log message to print.
//	err: The error that occurred.
//	w: The HTTP response writer [optional]. If nil, no response is sent to the client.
//	clientMsg: The message to send to the client [optional]. If w is nil, this
//		   parameter is ignored.
func serverError(logMsg string, err error, w http.ResponseWriter, clientMsg string) {
	logError(logMsg, err)
	if w != nil {
		http.Error(w, clientMsg, http.StatusInternalServerError)
	}
}

// -------------------- JOPLIN SERVER COMMUNICATION --------------------

// Creates a query to get a note from the Joplin server.
//
// Parameters:
//
//	noteID: The ID of the note to get.
//	fields: The fields to get from the note. If empty, all fields are returned.
//
// Returns:
//
//	The query to get the note.
func createNoteQuery(noteID string, fields string) string {
	query := "%s/notes/%s?token=%s"
	if fields != "" {
		query = query + "&fields=" + fields
	}
	query = fmt.Sprintf(query, JOPLIN_SERVER, noteID, JOPLIN_TOKEN)
	logInfo("createNoteQuery: " + query)
	return query
}

// Creates a query to get a list of notebooks from the Joplin server.
//
// Parameters:
//
//	page: Joplin server will return only N results per page. To get all,
//	      sucesive querys must be create to get all the pages. See
//	      NotebooksResponse.HasMore. First page is number 1.
//	fields: The fields to get from the notebooks. If empty, all fields
//	        are returned.
//
// Returns:
//
//	The query to get the notebooks.
func createNotebooksQuery(page int, fields string) string {
	query := "%s/folders?page=%d&token=%s"
	if fields != "" {
		query = query + "&fields=" + fields
	}
	query = fmt.Sprintf(query, JOPLIN_SERVER, page, JOPLIN_TOKEN)
	logInfo("createNotebooksQuery: " + query)
	return query
}

// Creates a query to get an image from the Joplin server.
//
// Parameters:
//
//	imageID: The ID of the image to retrieve.
//
// Returns:
//
//	The query to get the image.
func createImgQuery(imageID string) string {
	query := "%s/resources/%s/file?token=%s"
	query = fmt.Sprintf(query, JOPLIN_SERVER, imageID, JOPLIN_TOKEN)
	logInfo("createImgQuery: " + query)
	return query
}

// Creates a query to search for notes in the Joplin server.
//
// Parameters:
//
//	search: The search string to query.
//
// Returns:
//
//	The query to search for notes.
func createSearchQuery(search string) string {
	search = url.QueryEscape(search)
	query := "%s/search?query=%s&fields=id,title&token=%s"
	query = fmt.Sprintf(query, JOPLIN_SERVER, search, JOPLIN_TOKEN)
	logInfo("createSearchQuery: " + query)
	return query
}

// Retrieves the markdown content of a note from the Joplin server.
//
// Parameters:
//
//	noteID: The ID of the note to retrieve.
//
// Returns:
//
// A string containing the markdown content of the note. The markdown content
// is modified to replace Joplin markdown links with peeker server links.
// An error if one occurred during the retrieval of the note.
func getNoteMarkdown(noteID string) (string, error) {
	query := createNoteQuery(noteID, "title,body")
	logInfo("Query: %s\n", query)
	resp, err := http.Get(query)
	if err != nil {
		serverError("[getNoteMarkdown] http.Get -> %v", err, nil, "")
		return "", err
	}
	defer resp.Body.Close()

	// Parse the JSON response
	var data map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		serverError("[getNoteMarkdown] json.Decode -> %v", err, nil, "")
		return "", err
	}

	// Extract the "body" (markdown) and modify Joplins markdown links with
	// peeker server links.
	markdown := modifyMarkdown(data["body"].(string))

	// Add home link and open in Joplin link to the top of the markdown.
	const HOME_TEMPLATE = "<a href='/'>" +
		"<img src='/static/img/home.png' width='20px' height='20px' alt='Joplin Peeker Home'/>" +
		" Home</a>"
	const OPEN_IN_JOPLIN_TEMPLATE = "<a href='joplin://x-callback-url/openNote?id=%s'>" +
		"<img src='/static/img/joplin_logo.png' width='20px' height='20px' alt='Joplin Logo'/>" +
		" Open in Joplin</a>"
	nav_links := HOME_TEMPLATE + "   |   " + fmt.Sprintf(OPEN_IN_JOPLIN_TEMPLATE, noteID)
	markdown = nav_links + "\n\n" + markdown + "\n\n" + nav_links

	return markdown, nil
}

// getImageFromJoplin retrieves an image from a Joplin server using the
// provided image ID. It returns the image data as a byte slice, along with
// the content type and content length of the image.
//
// Parameters:
//
//	imageID: The ID of the image to retrieve.
//
// Returns:
//
//		[]byte: The image data as a byte slice.
//		string: The content type of the image (e.g., "image/png").
//		string: The content length of the image.
//	 error: An error if one occurred during the retrieval of the image.
func getImageFromJoplin(imageID string) ([]byte, string, string, error) {
	query := createImgQuery(imageID)

	resp, err := http.Get(query)
	if err != nil {
		serverError("[getImageFromJoplin] http.Get -> %v", err, nil, "")
		return nil, "", "", err
	}
	defer resp.Body.Close()

	// Read image from body into a []byte.
	image, err := io.ReadAll(resp.Body)
	if err != nil {
		serverError("[getImageFromJoplin] io.ReadAll -> %v", err, nil, "")
		return nil, "", "", err
	}

	return image,
		resp.Header.Get("Content-Type"),
		resp.Header.Get("Content-Length"),
		nil
}

// Retrieves all notebooks from the Joplin server.
//
// Returns:
//
//		A slice of Notebook structs representing the list of notebooks.
//	 An error if one occurred during the retrieval of the notebooks.
func getNotebooks() ([]Notebook, error) {
	page := 1
	// Resulting slice of notebooks.
	notebooks := []Notebook{}
	for {
		resp, err := http.Get(createNotebooksQuery(page, "id,title,parent_id"))
		if err != nil {
			serverError("[getNotebooks] http.Get -> %v", err, nil, "")
			return nil, err
		}
		defer resp.Body.Close()

		// Parse the JSON response
		var notebooksResponse NotebooksResponse
		err = json.NewDecoder(resp.Body).Decode(&notebooksResponse)
		if err != nil {
			serverError("[getNotebooks] json.Decode -> %v", err, nil, "")
			return nil, err
		}
		notebooks = append(notebooks, notebooksResponse.Items...)
		// If there are no more notebooks left, break the loop.
		if !notebooksResponse.HasMore {
			break
		}
		page++
	}

	return notebooks, nil
}

// Creates a tree of notebooks from a list of notebooks. Each notebook has a
// parent ID. The tree is created by recursively filling the children of each
// notebook.
//
// Parameters:
//
//	notebooks: A slice of Notebook structs representing the list of notebooks.
//
// Returns:
//
//	The root notebook node of the tree.
func createNotebooksTree(notebooks []Notebook) NotebookNode {
	const ROOT_TAG = "_root_"
	// Create a map of notebook IDs to their respective notebooks.
	var notebookIDMap = make(map[string]Notebook)
	for _, notebook := range notebooks {
		notebookIDMap[notebook.ID] = notebook
	}
	// Add the flag root notebook to the map.
	notebookIDMap[ROOT_TAG] =
		Notebook{ID: ROOT_TAG, Title: ROOT_TAG, ParentID: ""}

	// Create a map of parent IDs to a slice of its children notebooks.
	var notebookParentIDMap = make(map[string][]Notebook)
	for _, notebook := range notebooks {
		key := notebook.ParentID
		if key == "" {
			key = ROOT_TAG
		}
		notebookParentIDMap[key] = append(notebookParentIDMap[key], notebook)
	}
	// Call to the recursive function to fill the tree.
	return fillNotebooksTree(ROOT_TAG, notebookParentIDMap, notebookIDMap)
}

// Define a type for the slice of Notebook structs.
type ByTitle []Notebook

// Implement the sort.Interface for ByTitle
func (a ByTitle) Len() int           { return len(a) }
func (a ByTitle) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByTitle) Less(i, j int) bool { return a[i].Title < a[j].Title }

// Recursively fills the tree of notebooks.
//
// Parameters:
//
//	parentID: The ID of the parent notebook.
//	notebookParentIDMap: A map of parent IDs to a slice of its children notebooks.
//	notebookIDMap: A map of notebook IDs to their respective notebooks.
//
// Returns:
//
//	The parent notebook node with its children filled.
func fillNotebooksTree(parentID string,
	notebookParentIDMap map[string][]Notebook,
	notebookIDMap map[string]Notebook) NotebookNode {

	var parent = NotebookNode{
		ID:       parentID,
		Title:    notebookIDMap[parentID].Title,
		ParentID: notebookIDMap[parentID].ParentID,
		NumNotes: 0,
		Children: []NotebookNode{},
	}

	sort.Sort(ByTitle(notebookParentIDMap[parentID]))
	for _, notebook := range notebookParentIDMap[parentID] {
		child := fillNotebooksTree(notebook.ID, notebookParentIDMap, notebookIDMap)
		parent.Children = append(parent.Children, child)
	}
	return parent
}

// -------------------- HANDLERS --------------------

// Handles the root path of the server. It serves the index.html
// file from the static directory.
func rootHandler(w http.ResponseWriter, r *http.Request) {
	logInfo("Handling root: " + r.URL.Path)
	http.ServeFile(w, r, "static/index.html")
}

// Handles the /id/ path of the server. It serves the markdown content of a
// note from the Joplin server.
func noteContentHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the note ID from the URL path
	noteID := r.URL.Path[len("/id/"):]
	logInfo("Handling noteID: " + noteID)

	// Write the markdown response
	markdown, err := getNoteMarkdown(noteID)
	if err != nil {
		serverError("[noteContentHandler] getNoteMarkdown -> %v", err, w, "Failed to get note")
		return
	}
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	fmt.Fprintf(w, "%s", markdown)
}

// Handles the /search/ path of the server. It serves the search results from
// the Joplin server. The format of the JSON response is the same as the one
// returned by the Joplin server (items and has_more fields).
func searchHandler(w http.ResponseWriter, r *http.Request) {
	query := createSearchQuery(r.URL.Query().Get("query"))
	logInfo("Handling search: " + query)

	// Make an HTTP GET request to JOPLIN_SERVER with the search query
	resp, err := http.Get(query)
	if err != nil {
		serverError("[searchHandler] http.Get -> %v", err, w, "Failed to search notes")
		return
	}
	defer resp.Body.Close()

	// Return the same JSON response from Joplin Server to the client.
	w.Header().Set("Content-Type", "application/json")
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		serverError("[searchHandler] io.ReadAll -> %v", err, w, "Failed to search notes")
		return
	}
	_, err = w.Write(data)
	if err != nil {
		serverError("[searchHandler] w.Write -> %v", err, w, "Failed to search notes")
		return
	}
}

// Handles the /notebooks/ path of the server. It serves the notebooks tree
// from the Joplin server. The format of the JSON response is
// {id, title, parent_id, children} where children is a list of
// {id, title, parent_id, children}. The root notebook has an ID of "_root_".
func notebooksHandler(w http.ResponseWriter, r *http.Request) {
	logInfo("Handling notebooks")

	// Get the tree from Joplin Server
	notebooks, err := getNotebooks()
	if err != nil {
		serverError("[notebooksHandler] getNotebooks -> %v", err, w, "Failed to get notebooks tree")
		return
	}
	tree := createNotebooksTree(notebooks)

	// Return the same JSON response from Joplin Server to the client.
	w.Header().Set("Content-Type", "application/json")
	notebooksJSON, err := json.Marshal(tree)
	if err != nil {
		serverError("[notebooksHandler] json.Marshal -> %v", err, w, "Failed to get notebooks tree")
		return
	}

	// Write the notebook tree JSON response.
	_, err = w.Write(notebooksJSON)
	if err != nil {
		serverError("[notebooksHandler] w.Write -> %v", err, w, "Failed to get notebooks tree")
		return
	}
}

// Handles the /image/ path of the server. It serves the image content of a
// note from the Joplin server. It sets the content type and content length
// of the image in the response headers.
func imageHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the note ID from the URL path
	imageID := r.URL.Path[len("/image/"):]
	logInfo("Handling imgID: " + imageID)

	// Write the markdown response
	img, imgType, imgLength, err := getImageFromJoplin(imageID)
	if err != nil {
		serverError("[notebooksHandler] imageHandler -> %v", err, w, "Failed to get image")
		return
	}
	w.Header().Set("Content-Type", imgType)
	w.Header().Set("Content-Length", imgLength)
	w.Write(img)
}

// Handles the /favicon.ico path of the server. It serves the favicon.ico
// file from the static directory.
func iconHandler(w http.ResponseWriter, r *http.Request) {
	logInfo("Handling favicon.ico")
	http.ServeFile(w, r, "static/img/favicon.ico")
}
