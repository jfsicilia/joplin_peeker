// Description: This script is responsible for the search functionality and the
// sidebar behavior.

// Minimum length of the search query to trigger a search.
const MIN_SEARCH_LENGTH = 3;

// Key to store the last search query in the local storage.
const LAST_SEARCH_KEY = 'lastSearch';

/**
 * Clears the search box and optionally focuses on it.
 *
 * @param {boolean} focus - If true, the search box will be focused.
 * @returns {void}
 */
function clearSearchBox(focus = false) {
    const searchBox = document.getElementById('search');
    searchBox.value = '';
    if (focus) {
        searchBox.focus();
    }
}

/**
 * Clears the note list.
 * @returns {void}
 */
function clearNoteList() {
    const resultsContainer = document.getElementById('results');
    resultsContainer.innerHTML = '';
}

/**
 *  Updates the note list based on the given search results.
 *
 * @param {Object} results - The search results object containing the list of
 * notes.
 * @param {Object[]} results - The list of notes.
 * @param {string} results.id - The ID of the note.
 * @param {string} results.title - The title of the note.
 * @returns {void}
 */
function updateNoteList(results) {
    const resultsContainer = document.getElementById('results');
    resultsContainer.innerHTML = ''; // Clear previous results

    results.forEach(result => {
        const resultItem = document.createElement('div');
        resultItem.className = 'result-item';
        resultItem.innerHTML =
            `<a href=/note/${result.id}>${result.title}</a>`;
        resultsContainer.appendChild(resultItem);
    });
}

// Function to show an error banner with the given message.
function showErrorBanner(message) {
    // Set the error message
    const banner = document.getElementById('error-banner');
    banner.textContent = message + ' (Click anywhere or press any key to dismiss).';

    // Show the overlay and the banner
    const overlay = document.getElementById('overlay');
    overlay.style.display = 'block';
    banner.style.display = 'block';
}

// Function to hide the error banner.
function hideErrorBanner() {
    const overlay = document.getElementById('overlay');
    const banner = document.getElementById('error-banner');
    overlay.style.display = 'none';
    banner.style.display = 'none';
}

/**
 * Updates the note list based on the given query. Only queries with
 * 3 or more characters are considered.
 *
 * @param {string} query - The search query.
 * @returns {void}
 */
function searchNotes(query) {
    localStorage.setItem(LAST_SEARCH_KEY, query);
    if (query.length < 3) {
        clearNoteList();
        return
    }
    query = query.trim();

    console.debug('Searching for:', query);
    // "NotebookID:<notebook_id>" search query is a special case to fetch notes
    // from a notebook. Detect if this is the case, if not default to normal
    // search.
    const regex = /^notebookID:([\w-]+)$/;
    const match = query.match(regex);
    let serverQuery = match ? `/notebook/${match[1]}` : `/search/?query=${encodeURIComponent(query)}`;

    fetch(serverQuery)
        .then(data => {
            if (data.ok)
                return data.json()
            return data.text().then(errorMsg => { throw new Error(errorMsg) });
        })
        .then(results => { updateNoteList(results) })
        .catch(error => {
            showErrorBanner(error.message)
        });
}

/**
 * Updates the notebook tree by fetching the latest notebook data from the
 * server and recreating the tree of notebook nodes.
 * Each notebook node has a click listener to update the search results based
 * on the clicked notebook. Notebooks with sub-notebooks have a caret icon to
 * toggle the visibility of the sub-notebooks.
 *
 * @returns {void}
 */
function updateNotebooksTree() {
    const tree = document.getElementById('notebook-tree');
    tree.innerHTML = '';

    fetch('/notebooks/')
        .then(data => {
            if (data.ok)
                return data.json()
            return data.text().then(errorMsg => { throw new Error(errorMsg) });
        })
        .then(root => root.children.forEach(child => createNotebookNode(tree, child)))
        // After creating the tree, add click listeners to the elements.
        .then(() => {
            // Add listeners to notebooks with sub-notebooks.
            const togglers = document.querySelectorAll('.caret');
            togglers.forEach(toggler => {
                toggler.addEventListener('click', function () {
                    this.parentElement.querySelector('.nested').classList.toggle('active');
                    this.classList.toggle('caret-down');
                    const search = document.getElementById('search');
                    search.value = `notebookID:${this.getAttribute('data-id')}`;
                    searchNotes(search.value);
                });
            });
            // Add listeners to notebooks without sub-notebooks.
            const noTogglers = document.querySelectorAll('.no-caret');
            noTogglers.forEach(noToggler => {
                noToggler.addEventListener('click', function () {
                    const search = document.getElementById('search');
                    search.value = `notebookID:${this.getAttribute('data-id')}`;
                    searchNotes(search.value);
                });
            });
        })
        .catch(error => {
            showErrorBanner(error.message)
        });
}

/**
 * Creates a tree of notebooks elements and appends it to the given HTML node.
 * This * function is called recursively to create nested notebooks.
 *
 * @param {HTMLElement} htmlNode - The HTML node to which a new notebook element
 * will be appended as a child.
 * @param {Object} notebook - The notebook object containing details to be
 * displayed.
 * @param {string} notebook.id - The ID of the notebook.
 * @param {string} notebook.title - The title of the notebook.
 * @param {string} notebook.children - The list of child notebooks.
 * @returns {void}
 */
function createNotebookNode(htmlNode, notebook) {
    const notebookItem = document.createElement('li');
    notebookItem.className = 'notebook';
    const notebookSpan = document.createElement('a');
    notebookSpan.className =
        (notebook.children.length > 0) ? 'caret' : 'no-caret';
    notebookSpan.innerHTML = notebook.title;
    notebookSpan.setAttribute('data-id', notebook.id);
    notebookItem.appendChild(notebookSpan);

    // Add nested notebooks recursively if there are any.
    if (notebook.children.length > 0) {
        const nested = document.createElement('ul');
        nested.className = 'nested';
        notebookItem.appendChild(nested);
        notebook.children.forEach(child => createNotebookNode(nested, child));
    }
    htmlNode.appendChild(notebookItem);
}

/**
 * Initializes the search functionality by setting the search box value to the
 * last search query and updating the results based on the last search.
 * Also, adds listeners to the search box and the clear search box button.
 * @returns {void}
 */
function initSearch() {
    // Get searchBox set its value to the last search query,
    // then update the results based on the last search.
    const searchBox = document.getElementById('search');
    searchBox.value = localStorage.getItem(LAST_SEARCH_KEY);
    searchNotes(searchBox.value);

    // Add listener to searchBox to update the results on input.
    searchBox.addEventListener('input', function () {
        const query = this.value.trim();
        searchNotes(query);
    });

    // Add listener to clearSearchBox to clear the searchBox and focus on it.
    const clearSearchBox = document.getElementById('clearSearchBox');
    clearSearchBox.addEventListener('click', function () {
        clearSearchBox(true);
        clearNoteList();
        localStorage.setItem(LAST_SEARCH_KEY, '');
    });
}

/**
 * Initializes the divider between the sidebar and the main content. The divider
 * can be dragged to resize the sidebar and the main content.
 * @returns {void}
 */
function initDivider() {
    const divider = document.getElementById('divider');
    const sidebar = document.getElementById('sidebar');
    const main = document.getElementById('main');

    let isDragging = false;

    // Add listeners to the divider to start, move, and stop dragging.
    divider.addEventListener('mousedown', function (e) {
        isDragging = true;
        document.body.style.cursor = 'ew-resize';
    });

    // Resize the sidebar and main elements based on the mouse position.
    document.addEventListener('mousemove', function (e) {
        if (!isDragging) return;
        const containerWidth = document.querySelector('.container').offsetWidth;
        const newSidebarWidth = (e.clientX / containerWidth) * 100;
        sidebar.style.width = `${newSidebarWidth}%`;
        main.style.width = `${100 - newSidebarWidth}%`;
    });

    // Stop dragging when the mouse is released.
    document.addEventListener('mouseup', function () {
        isDragging = false;
        document.body.style.cursor = 'default';
    });
}

/**
 * Initializes the error banner and keyboard shortcuts. The error banner is
 * hidden when the user clicks anywhere on the document or presses any key.
 * The search box is focused and all text is selected when the user presses
 * the '/' key. The search box is cleared when the user presses the 'x' key.
 * @returns {void}
 */
function initErrorBannerAndKeyboardShortcuts() {
    // Hide the error banner, if open, when the user clicks anywhere on the
    // document.
    document.addEventListener('click', function (event) {
        const banner = document.getElementById('error-banner');
        // If banner is open, hide it and prevent the default action.
        if (banner.style.display === 'block') {
            event.preventDefault();
            hideErrorBanner();
            return;
        }
    });

    // Hide the error banner, if open, when the user presses any key.
    //
    // '/' and 'x' have special functionality outside the searchbox, so we
    // check if those keys have been pressed.
    // '/' focuses on the search box and selects all text.
    // 'x' clears the search box.
    document.addEventListener('keydown', function (event) {
        const banner = document.getElementById('error-banner');
        // If banner is open, hide it and prevent the default action.
        if (banner.style.display === 'block') {
            event.preventDefault();
            hideErrorBanner();
            return;
        }

        const searchBox = document.getElementById('search');
        if (document.activeElement === searchBox)
            return; // Do nothing if the search box already has focus

        if (event.key === '/') {
            event.preventDefault(); // Prevent the default action of the "/" key
            searchBox.focus();
            searchBox.select();
        } else if (event.key === 'x') {
            event.preventDefault(); // Prevent the default action of the "/" key
            clearSearchBox(true);
            localStorage.setItem(LAST_SEARCH_KEY, '');
        }
    });
}

// Main function to be executed when the DOM is fully loaded.
document.addEventListener('DOMContentLoaded', function () {
    initSearch();
    initDivider();
    initErrorBannerAndKeyboardShortcuts();
    // Fill the notebooks tree on the sidebar.
    updateNotebooksTree();
});
