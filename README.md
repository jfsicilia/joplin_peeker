# <img src="/assets/img/joplin_peeker_logo.png" width="100px"></img> Joplin Peeker Server

A simple local web server to peek on your Joplin notes and notebooks.

<img src="/assets/img/joplin_peeker_usage.gif" width="600px"></img>

Joplin Peeker is a simple pet project that allows to browse your Joplin's notes and notebooks in your favourite web browser. This allows to use your favourite extensions in your browser when searching and viewing your notes. For example, you can use [Vimium](https://chromewebstore.google.com/detail/vimium/dbepggeogbaibhgnhhndojpepiihcmeb) to navigate your notes using only the keyboard and [Markdown Viewer](https://chromewebstore.google.com/detail/markdown-viewer/ckkdlimhmcjmikdlpkmbgfkaikojcbjk) to view an enhanced version of your note's markdown.

## Running

Joplin Peeker is written in Go in the backend and JS/CSS/HTML in the frontend. No external modules have been used, so the build/run of the server is straight forward. After cloning the repository, to run the server with the default configuration execute this command in the root folder of the project:

```bash
$ go run peeker_server.go
```

You sould see something like:

<img src="/assets/img/running_peeker_server.png" width="400px"></img>

**NOTE:** Joplin desktop/cli app must be running and the Webclipper server enabled.

<img src="/assets/img/joplin_settings.png" width="400px"></img>

The following parameters can be set to configure the server:

> - *JOPLIN_SERVER*: The address and port of the Joplin Webclipper Server (default: `http://localhost:41184`).
> - *JOPLIN_TOKEN*: The Joplin Authorization token ([`REQUIRED`] No default provided).
> - *PEEKER_HOST*: The address of the Joplin Peeker Server (default: `127.0.0.1`).
> - *PEEKER_PORT*: The port of the Joplin Peeker Server (default: `8080`).

The are three ways to set the parameters. If none is used, harcoded default values are used. This is the order of prevalence for setting the parameters:

### 1. Using the command line to set the parameters.

```bash
Usage of peeker_server:
  -host string
        Listen address for the peeker server (default: 127.0.0.1)
  -joplin string
        Joplin server address and port (default: http://localhost:41184)
  -port string
        Listen port for the peeker server (default: 8080)
  -token string
        Joplin access token
  -v    Verbose mode (default: false)
```
   
### 2. Set the parameters using environment variables.

```bash
# Example of setting PEEKER_HOST as an environment variable.
export PEEKER_HOST=192.168.1.1
```

### 3. Use a `config.json` file to set the parameters. 

The file must be in the same folder as the `peeker_server.go` or the `peeker_server` executable.

```json
{
    "joplin_server": "http://localhost:41184",
    "joplin_token": "...c183779101c931...",
    "peeker_host": "127.0.0.1",
    "peeker_port": "8080"
}
```

Finally, when configured for own settings and run, the Peeker Server could be access in any browser navigation to `http://<PEEKER_HOST>:<PEEKER_PORT>`. The initial web page will look something like:

<img src="assets/img/peeker_server_main.png" width="500px"></img>

On the left the notebooks tree. Clicking on any notebook will fetch all the notes of that notebook. On the right the search box. You can use the same search syntax that Joplins use ([Joplin Searching](https://joplinapp.org/help/apps/search/)).

## Building & Installing.

To create.


