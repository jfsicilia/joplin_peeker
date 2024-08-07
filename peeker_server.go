package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"sort"
)

// Default host and port for the HTTP server.
const PEEKER_HOST = "127.0.0.1"
const PEEKER_PORT = "8080"

// Joplins server address.
const JOPLIN_SERVER = "http://localhost:51184"

// Joplins access token.
const JOPLIN_TOKEN = "8ec6a65bc4a11dfc183779101c9315c3eb8a3062014a6eeb7b78d18f638f96acfdc9e2dfc92214aec0934105eb0f4657900f0efd8a9ae8a144f393606f0d535e"

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

	// Start server.
	server := fmt.Sprintf("%s:%s", PEEKER_HOST, PEEKER_PORT)
	fmt.Printf("Server listening on %s...\n", server)
	log.Fatal(http.ListenAndServe(server, nil))
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
	log.Println("createNoteQuery: " + query)
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
	log.Println("createNotebooksQuery: " + query)
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
	log.Println("createImgQuery: " + query)
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
	log.Println("createSearchQuery: " + query)
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
func getNoteMarkdown(noteID string) string {
	query := createNoteQuery(noteID, "title,body")
	log.Printf("Query: %s\n", query)
	resp, err := http.Get(query)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	// Parse the JSON response
	var data map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		log.Fatalf("json.Decode -> %v", err)
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
	markdown = HOME_TEMPLATE + "   |   " + fmt.Sprintf(OPEN_IN_JOPLIN_TEMPLATE, noteID) + "\n\n" + markdown

	return markdown
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
//	[]byte: The image data as a byte slice.
//	string: The content type of the image (e.g., "image/png").
//	string: The content length of the image.
func getImageFromJoplin(imageID string) ([]byte, string, string) {
	query := createImgQuery(imageID)

	resp, err := http.Get(query)
	if err != nil {
		log.Fatalf("http.Get -> %v", err)
	}
	defer resp.Body.Close()

	// Read image from body into a []byte.
	image, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("io.ReadAll -> %v", err)
	}

	return image,
		resp.Header.Get("Content-Type"),
		resp.Header.Get("Content-Length")
}

// Retrieves all notebooks from the Joplin server.
//
// Returns:
//
//	A slice of Notebook structs representing the list of notebooks.
func getNotebooks() []Notebook {
	page := 1
	// Resulting slice of notebooks.
	notebooks := []Notebook{}
	for {
		resp, err := http.Get(createNotebooksQuery(page, "id,title,parent_id"))
		if err != nil {
			log.Fatalf("http.Get -> %v", err)
		}
		defer resp.Body.Close()

		// Parse the JSON response
		var notebooksResponse NotebooksResponse
		err = json.NewDecoder(resp.Body).Decode(&notebooksResponse)
		if err != nil {
			log.Fatalf("json.Decode -> %v", err)
		}
		notebooks = append(notebooks, notebooksResponse.Items...)
		// If there are no more notebooks left, break the loop.
		if !notebooksResponse.HasMore {
			break
		}
		page++
	}

	return notebooks
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
	log.Println("Handling root: " + r.URL.Path)
	http.ServeFile(w, r, "static/index.html")
}

// Handles the /id/ path of the server. It serves the markdown content of a
// note from the Joplin server.
func noteContentHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the note ID from the URL path
	noteID := r.URL.Path[len("/id/"):]
	log.Println("Handling noteID: " + noteID)

	// Write the markdown response
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	fmt.Fprintf(w, "%s", getNoteMarkdown(noteID))
}

// Handles the /search/ path of the server. It serves the search results from
// the Joplin server. The format of the JSON response is the same as the one
// returned by the Joplin server (items and has_more fields).
func searchHandler(w http.ResponseWriter, r *http.Request) {
	query := createSearchQuery(r.URL.Query().Get("query"))
	log.Println("Handling search: " + query)

	// Make an HTTP GET request to JOPLIN_SERVER with the search query
	resp, err := http.Get(query)
	if err != nil {
		log.Fatalf("http.Get -> %v", err)
	}
	defer resp.Body.Close()

	// Return the same JSON response from Joplin Server to the client.
	w.Header().Set("Content-Type", "application/json")
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("io.ReadAll -> %v", err)
	}
	_, err = w.Write(data)
	if err != nil {
		log.Fatalf("w.Write -> %v", err)
	}
}

// Handles the /notebooks/ path of the server. It serves the notebooks tree
// from the Joplin server. The format of the JSON response is
// {id, title, parent_id, children} where children is a list of
// {id, title, parent_id, children}. The root notebook has an ID of "_root_".
func notebooksHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Handling notebooks")

	// Get the tree from Joplin Server
	tree := createNotebooksTree(getNotebooks())

	// Return the same JSON response from Joplin Server to the client.
	w.Header().Set("Content-Type", "application/json")
	notebooksJSON, err := json.Marshal(tree)
	if err != nil {
		log.Fatalf("json.Marshal -> %v", err)
	}
	_, err = w.Write(notebooksJSON)
	if err != nil {
		log.Fatalf("w.Write -> %v", err)
	}
}

// Handles the /image/ path of the server. It serves the image content of a
// note from the Joplin server. It sets the content type and content length
// of the image in the response headers.
func imageHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the note ID from the URL path
	imageID := r.URL.Path[len("/image/"):]
	log.Println("Handling imgID: " + imageID)

	// Write the markdown response
	img, imgType, imgLength := getImageFromJoplin(imageID)
	w.Header().Set("Content-Type", imgType)
	w.Header().Set("Content-Length", imgLength)
	w.Write(img)
}

// Handles the /favicon.ico path of the server. It serves the favicon.ico
// file from the static directory.
func iconHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Handling favicon.ico")
	http.ServeFile(w, r, "static/img/favicon.ico")
}
