# <img src="/assets/img/joplin_peeker_logo.png" width="100px"></img> Joplin Peeker Server

A simple local web server to peek on your Joplin notes and notebooks.

<img src="/assets/img/joplin_peeker_usage.gif" width="600px"></img>

Joplin Peeker is a simple pet project that allows to browse your Joplin's notes and notebooks in your favourite web browser. This allows to use your favourite extensions in your browser when searching and viewing your notes. For example, you can use [Vimium](https://chromewebstore.google.com/detail/vimium/dbepggeogbaibhgnhhndojpepiihcmeb) to navigate your notes using only the keyboard and [Markdown Viewer](https://chromewebstore.google.com/detail/markdown-viewer/ckkdlimhmcjmikdlpkmbgfkaikojcbjk) to view an enhanced version of your note's markdown.

## Building & Running.

Joplin Peeker is written in Go in the backend and JS/CSS/HTML in the frontend. No external modules have been used, so the build/run of the server is straight forward. After cloning the repository, to run the server with the default configuration execute this command in the root folder of the project:

```bash
$ go run peeker_server.go
```

You sould see something like:

<img src="/assets/img/running_peeker_server.png" width="400px"></img>

## Installation



