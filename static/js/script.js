const LAST_SEARCH_KEY = 'lastSearch';

/**
 * Updates the search results based on the given query.
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
    fetch('/search/?query=' + encodeURIComponent(query))
        .then(data => data.json())
        .then(results => {
            resultsContainer.innerHTML = ''; // Clear previous results

            results.items.forEach(result => {
                const resultItem = document.createElement('div');
                resultItem.className = 'result-item';
                resultItem.innerHTML = '<a href=/id/' + result.id + '>' + result.title + '</a>';
                resultsContainer.appendChild(resultItem);
            });
        });
}

function createNotebookNode(htmlNode, notebook) {
    const notebookItem = document.createElement('li');
    notebookItem.className = 'notebook';
    const notebookSpan = document.createElement('a');
    notebookSpan.className =
        (notebook.children.length > 0) ? 'caret' : 'no-caret';
    notebookSpan.innerHTML = notebook.title;
    notebookItem.appendChild(notebookSpan);
    if (notebook.children.length > 0) {
        const nested = document.createElement('ul');
        nested.className = 'nested';
        notebookItem.appendChild(nested);
        notebook.children.forEach(child => createNotebookNode(nested, child));
    }
    htmlNode.appendChild(notebookItem);
}

function updateNotebooks() {
    const tree = document.getElementById('notebook-tree');
    tree.innerHTML = '';

    fetch('/notebooks/')
        .then(data => data.json())
        .then(root => root.children.forEach(child => createNotebookNode(tree, child)))
        .then(() => {
            const togglers = document.querySelectorAll('.caret');
            togglers.forEach(toggler => {
                toggler.addEventListener('click', function () {
                    this.parentElement.querySelector('.nested').classList.toggle('active');
                    this.classList.toggle('caret-down');
                    const search = document.getElementById('search');
                    search.value = 'notebook:' + this.innerHTML;
                    updateResults(search.value);
                });
            });
            const noTogglers = document.querySelectorAll('.no-caret');
            noTogglers.forEach(noToggler => {
                noToggler.addEventListener('click', function () {
                    const search = document.getElementById('search');
                    search.value = 'notebook:' + this.innerHTML;
                    updateResults(search.value);
                });
            });
        });
}

document.addEventListener('DOMContentLoaded', function () {
    // Add listeners to all toggler elements.
    const togglers = document.querySelectorAll('.caret');
    togglers.forEach(toggler => {
        toggler.addEventListener('click', function () {
            this.parentElement.querySelector('.nested').classList.toggle('active');
            this.classList.toggle('caret-down');
        });
    });

    // Get searchBox, focus on it and set its value to the last search query,
    // then update the results based on the last search.
    const searchBox = document.getElementById('search');
    searchBox.value = localStorage.getItem(LAST_SEARCH_KEY);
    updateResults(searchBox.value);
    // searchBox.select();

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


    // Divider resizing.
    const divider = document.getElementById('divider');
    const sidebar = document.getElementById('sidebar');
    const main = document.getElementById('main');

    let isDragging = false;

    divider.addEventListener('mousedown', function (e) {
        isDragging = true;
        document.body.style.cursor = 'ew-resize';
    });

    document.addEventListener('mousemove', function (e) {
        if (!isDragging) return;
        const containerWidth = document.querySelector('.container').offsetWidth;
        const newSidebarWidth = (e.clientX / containerWidth) * 100;
        sidebar.style.width = `${newSidebarWidth}%`;
        main.style.width = `${100 - newSidebarWidth}%`;
    });

    document.addEventListener('mouseup', function () {
        isDragging = false;
        document.body.style.cursor = 'default';
    });

    // '/' and 'x' have special functionality outside the searchbox.
    // '/' focuses on the search box and selects all text.
    // 'x' clears the search box.0
    document.addEventListener('keydown', function (event) {
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
