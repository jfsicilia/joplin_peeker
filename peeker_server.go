package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
)

const PEEKER_HOST = "127.0.0.1"
const PEEKER_PORT = "8080"

// Joplins server address.
const JOPLIN_SERVER = "http://localhost:51184"

// Joplins access token.
const JOPLIN_TOKEN = "8ec6a65bc4a11dfc183779101c9315c3eb8a3062014a6eeb7b78d18f638f96acfdc9e2dfc92214aec0934105eb0f4657900f0efd8a9ae8a144f393606f0d535e"

const HOME_TEMPLATE = "<a href='/'>" +
	"<img src='/static/img/home.png' width='20px' height='20px' alt='Joplin Peeker Home'/>" +
	" Home</a>"

// Open in Joplin template.
const OPEN_IN_JOPLIN_TEMPLATE = "<a href='joplin://x-callback-url/openNote?id=%s'>" +
	"<img src='/static/img/joplin_logo.png' width='20px' height='20px' alt='Joplin Logo'/>" +
	" Open in Joplin</a>"

// Example Note ID.
const NOTE_ID = "e8ea1a3a654f48e88e5433903a49e341"

type Note struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type NotesResponse struct {
	Items   []Note `json:"items"`
	HasMore bool   `json:"has_more"`
}

type Notebook struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	ParentID string `json:"parent_id"`
}

type NotebooksResponse struct {
	Items   []Notebook `json:"items"`
	HasMore bool       `json:"has_more"`
}

type NotebookNode struct {
	ID       string         `json:"id"`
	Title    string         `json:"title"`
	ParentID string         `json:"parent_id"`
	Children []NotebookNode `json:"children"`
}

type NotebooksTree struct {
	Items []NotebookNode `json:"items"`
}

// main is the entry point of the program.
// It sets up the HTTP server and starts listening on port 8080.
func main() {
	// Config server handlers.
	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/id/", noteHandler)
	http.HandleFunc("/image/", imageHandler)
	http.HandleFunc("/favicon.ico", iconHandler)
	http.HandleFunc("/search/", searchHandler)
	http.HandleFunc("/notebooks/", notebooksHandler)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Start server.
	server := fmt.Sprintf("%s:%s", PEEKER_HOST, PEEKER_PORT)
	fmt.Printf("Server listening on %s...\n", server)
	log.Fatal(http.ListenAndServe(server, nil))
}

const ROOT_TAG = "_root_"

func createNotebooksTree(notebooks []Notebook) NotebookNode {
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
	return fillNotebooksTree(ROOT_TAG, notebookParentIDMap, notebookIDMap)
}

func fillNotebooksTree(parentID string,
	notebookParentIDMap map[string][]Notebook,
	notebookIDMap map[string]Notebook) NotebookNode {

	var parent = NotebookNode{
		ID:       parentID,
		Title:    notebookIDMap[parentID].Title,
		ParentID: notebookIDMap[parentID].ParentID,
		Children: []NotebookNode{},
	}
	for _, notebook := range notebookParentIDMap[parentID] {
		child := fillNotebooksTree(notebook.ID, notebookParentIDMap, notebookIDMap)
		parent.Children = append(parent.Children, child)
	}
	return parent
}

func modifyMarkdown(markdown string) string {
	// Replace Joplin markdown image with server links.
	// ![text](/:/id) to <img src="/image/<id>" alt="text" />
	imgLinksRE := regexp.MustCompile(`!\[(.*?)\]\(:/(.*?)\)`)
	replacement := "<img src='/image/${2}' alt='${1}'/>"
	markdown = imgLinksRE.ReplaceAllString(markdown, replacement)

	// Replace Joplin markdown note links with the server links.
	// [text](/:/id) to [text](/id/<id>)
	linksRE := regexp.MustCompile(`([^!])\[(.*?)\]\(:/(.*?)\)`)
	replacement = "${1}[${2}](/id/${3})"
	markdown = linksRE.ReplaceAllString(markdown, replacement)

	// Replace Joplin markdown html img tags with the server links.
	// <img src="/:id" /> to <img src="/image/<id>" />
	imgSrcRE := regexp.MustCompile(`<img(.*?)src=["']:/(.*?)["'](.*?)/>`)
	replacement = "<img src='/image/${2}'${3}/>"
	markdown = imgSrcRE.ReplaceAllString(markdown, replacement)

	return markdown
}

func createNoteQuery(noteID string, fields string) string {
	query := "%s/notes/%s?token=%s"
	if fields != "" {
		query = query + "&fields=" + fields
	}
	query = fmt.Sprintf(query, JOPLIN_SERVER, noteID, JOPLIN_TOKEN)
	log.Println("createNoteQuery: " + query)
	return query
}

func createNotebooksQuery(page int, fields string) string {
	query := "%s/folders?page=%d&token=%s"
	if fields != "" {
		query = query + "&fields=" + fields
	}
	query = fmt.Sprintf(query, JOPLIN_SERVER, page, JOPLIN_TOKEN)
	log.Println("createNotebooksQuery: " + query)
	return query
}

func createImgQuery(imageID string) string {
	query := "%s/resources/%s/file?token=%s"
	query = fmt.Sprintf(query, JOPLIN_SERVER, imageID, JOPLIN_TOKEN)
	log.Println("createImgQuery: " + query)
	return query
}

func createSearchQuery(search string) string {
	search = url.QueryEscape(search)
	query := "%s/search?query=%s&fields=id,title&token=%s"
	query = fmt.Sprintf(query, JOPLIN_SERVER, search, JOPLIN_TOKEN)
	log.Println("createSearchQuery: " + query)
	return query
}

func getNotebooks() []Notebook {
	// Make an HTTP GET request to JOPLIN_SERVER with the search query
	page := 1
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
		if !notebooksResponse.HasMore {
			break
		}
		page++
	}

	return notebooks
}

func getNoteMarkdown(noteID string) string {
	// Make an HTTP GET request to JOPLIN_SERVER with the extracted note ID
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

	// Extract the "body" (markdown), dress it and return it
	markdown := modifyMarkdown(data["body"].(string))
	markdown = HOME_TEMPLATE + "   |   " + fmt.Sprintf(OPEN_IN_JOPLIN_TEMPLATE, noteID) + "\n\n" + markdown

	return markdown
}

// getImageFromJoplin retrieves an image from a Joplin server using the provided image ID.
// It returns the image data as a byte slice, along with the content type and
// content length of the image.
//
// Parameters:
// - imageID: The ID of the image to retrieve.
//
// Returns:
// - []byte: The image data as a byte slice.
// - string: The content type of the image.
// - string: The content length of the image.
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

func rootHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Handling root: " + r.URL.Path)
	http.ServeFile(w, r, "static/index.html")
}

func noteHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the note ID from the URL path
	noteID := r.URL.Path[len("/id/"):]
	log.Println("Handling noteID: " + noteID)

	// Write the markdown response
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	fmt.Fprintf(w, "%s", getNoteMarkdown(noteID))
}

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

func iconHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Handling favicon.ico")
	http.ServeFile(w, r, "static/img/favicon.ico")
}
