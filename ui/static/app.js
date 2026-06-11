document.addEventListener('DOMContentLoaded', () => {
    // ============================================
    // 1. ADDING A NOTE (Explicit form listener)
    // ============================================
    const addForm = document.getElementById('add-note-form');
    const notesContainer = document.getElementById('notes-list-container');

    if (addForm) {
        addForm.addEventListener('submit', function(event) {
            // Prevent the default full page reload
            event.preventDefault();

            // Gather the data from the form
            const formData = new FormData(addForm);

            // Fetch the /add URL explicitly using the form's action and method
            fetch(addForm.action, {
                method: addForm.method,
                // Convert FormData to URL-encoded string because our Go server expects standard form submission
                body: new URLSearchParams(formData), 
                headers: {
                    'Content-Type': 'application/x-www-form-urlencoded',
                    // This custom header tells our Go server it's an AJAX request
                    'X-Requested-With': 'Fetch' 
                }
            })
            .then(response => {
                if (!response.ok) throw new Error("Network response was not ok");
                return response.text();
            })
            .then(html => {
                // We received the partial HTML string from the server. 
                // Now we swap it into the container explicitly!
                notesContainer.innerHTML = html;
                
                // Clear the input field for the next note
                addForm.reset();
            })
            .catch(error => console.error("Error adding note:", error));
        });
    }

    // ============================================
    // 2. DELETING A NOTE (The Event Delegation strategy)
    // ============================================
    // We hit the "Dynamic Element" trap here!
    // If we just attach listeners directly to existing .delete-form elements,
    // any *newly added* notes won't have the listener and will do a full page reload.
    // Instead, we listen on the container (which always exists) and check what was submitted.
    
    if (notesContainer) {
        notesContainer.addEventListener('submit', function(event) {
            // Check if the submit event originated from a delete form
            if (event.target && event.target.classList.contains('delete-form')) {
                // Prevent full reload
                event.preventDefault();
                
                const deleteForm = event.target;
                
                fetch(deleteForm.action, {
                    method: deleteForm.method,
                    headers: {
                        'X-Requested-With': 'Fetch'
                    }
                })
                .then(response => {
                    if (!response.ok) throw new Error("Network response was not ok");
                    return response.text();
                })
                .then(html => {
                    // Update the list with the newly returned list (which excludes the deleted note)
                    notesContainer.innerHTML = html;
                })
                .catch(error => console.error("Error deleting note:", error));
            }
        });
    }
});
