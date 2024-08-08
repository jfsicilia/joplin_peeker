// Description: This script is responsible for the search functionality and the
// sidebar behavior.

// Minimum length of the search query to trigger a search.
const MIN_SEARCH_LENGTH = 3;

// Key to store the last search query in the local storage.
const LAST_SEARCH_KEY = 'lastSearch';

/**
 * Updates the search results list based on the given query. Only queries with
 * 3 or more characters are considered.
 *
 * @param {string} query - The search query.
 * @returns {void}
 */
function updateResults(query) {
    const resultsContainer = document.getElementById('results');

    localStorage.setItem(LAST_SEARCH_KEY, query);
    if (query.length < 3) {
        resultsContainer.innerHTML = ''; // Clear previous results
        return
    }

    console.debug('Searching for:', query);
    fetch(`/search/?query=${encodeURIComponent(query)}`)
        .then(data => {
            if (data.ok)
                return data.json()
            return data.text().then(errorMsg => { throw new Error(errorMsg) });
        })
        .then(results => {
            resultsContainer.innerHTML = ''; // Clear previous results

            results.items.forEach(result => {
                const resultItem = document.createElement('div');
                resultItem.className = 'result-item';
                resultItem.innerHTML =
                    `<a href=/id/${result.id}>${result.title}</a>`;
                resultsContainer.appendChild(resultItem);
            });
        })
        .catch(error => {
            showErrorBanner(error.message)
        });
}

/**
 * Creates a tree of notebooks and appends it to the given HTML node. This
 * function is called recursively to create nested notebooks.
 *
 * @param {HTMLElement} htmlNode - The HTML node to which the notebook item
 * will be appended.
 * @param {Object} notebook - The notebook object containing details to be
 * displayed.
 * @param {string} notebook.id - The ID of the notebook.
 * @param {string} notebook.title - The title of the notebook.
 * @returns {void}
 */
function createNotebookNode(htmlNode, notebook) {
    const notebookItem = document.createElement('li');
    notebookItem.className = 'notebook';
    const notebookSpan = document.createElement('a');
    notebookSpan.className =
        (notebook.children.length > 0) ? 'caret' : 'no-caret';
    notebookSpan.innerHTML = notebook.title;
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
 * Updates the notebook tree by fetching the latest notebook data from the
 * server and recreating the tree of notebook nodes.
 * Each notebook node has a click listener to update the search results based
 * on the clicked notebook. Notebooks with sub-notebooks have a caret icon to
 * toggle the visibility of the sub-notebooks.
 *
 * @returns {void}
 */
function updateNotebooks() {
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
                    search.value = `notebook:${this.innerHTML}`;
                    updateResults(search.value);
                });
            });
            // Add listeners to notebooks without sub-notebooks.
            const noTogglers = document.querySelectorAll('.no-caret');
            noTogglers.forEach(noToggler => {
                noToggler.addEventListener('click', function () {
                    const search = document.getElementById('search');
                    search.value = `notebook:${this.innerHTML}`;
                    updateResults(search.value);
                });
            });
        })
        .catch(error => {
            showErrorBanner(error.message)
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

// Main function to be executed when the DOM is fully loaded.
document.addEventListener('DOMContentLoaded', function () {

    // Get searchBox set its value to the last search query,
    // then update the results based on the last search.
    const searchBox = document.getElementById('search');
    searchBox.value = localStorage.getItem(LAST_SEARCH_KEY);
    updateResults(searchBox.value);

    // Add listener to searchBox to update the results on input.
    searchBox.addEventListener('input', function () {
        const query = this.value.trim();
        updateResults(query);
    });

    // Add listener to clearSearchBox to clear the searchBox and focus on it.
    const clearSearchBox = document.getElementById('clearSearchBox');
    const resultsContainer = document.getElementById('results');
    clearSearchBox.addEventListener('click', function () {
        searchBox.value = '';
        searchBox.focus();
        localStorage.setItem(LAST_SEARCH_KEY, '');
        resultsContainer.innerHTML = ''; // Clear previous results
    });


    // ----------------- Divider resizing. -----------------
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
            searchBox.value = '';
            searchBox.focus();
            localStorage.setItem(LAST_SEARCH_KEY, '');
        }
    });

    // Fill the notebooks tree on the sidebar.
    updateNotebooks();
});
