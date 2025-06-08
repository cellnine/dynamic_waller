document.addEventListener('DOMContentLoaded', () => {
    // Get references to our HTML elements
    const form = document.getElementById('upload-form');
    const lightFileInput = document.getElementById('light-file');
    const darkFileInput = document.getElementById('dark-file');
    const submitButton = document.getElementById('submit-button');
    const statusArea = document.getElementById('status-area');
    const statusText = document.getElementById('status-text');
    const downloadLink = document.getElementById('download-link');
    const galleryGrid = document.getElementById('gallery-grid');

    let pollingInterval; // To store our setInterval ID

    // --- FORM SUBMISSION ---
    form.addEventListener('submit', async (e) => {
        e.preventDefault(); // Stop the browser from actually submitting the form

        if (!lightFileInput.files[0] || !darkFileInput.files[0]) {
            alert('Please select both a light and a dark image.');
            return;
        }

        // Create a FormData object to send the files
        const formData = new FormData();
        formData.append('light', lightFileInput.files[0]);
        formData.append('dark', darkFileInput.files[0]);

        // Update the UI to show we're working
        submitButton.disabled = true;
        submitButton.textContent = 'Uploading...';
        statusArea.classList.remove('hidden');
        statusText.textContent = 'Uploading images...';
        downloadLink.classList.add('hidden');

        try {
            // Send the files to our Go backend
            const response = await fetch('/api/create', {
                method: 'POST',
                body: formData,
            });

            if (!response.ok) {
                throw new Error(`Server error: ${response.statusText}`);
            }

            const result = await response.json();
            const { id } = result;

            // Start polling for the status of our job
            statusText.textContent = 'Processing... This may take a moment.';
            pollStatus(id);

        } catch (error) {
            console.error('Error creating wallpaper:', error);
            statusText.textContent = `Error: ${error.message}. Please try again.`;
            submitButton.disabled = false;
            submitButton.textContent = 'Create Wallpaper';
        }
    });

    // --- POLLING FOR STATUS ---
    function pollStatus(jobId) {
        // Clear any previous polling
        if (pollingInterval) {
            clearInterval(pollingInterval);
        }

        pollingInterval = setInterval(async () => {
            try {
                const response = await fetch(`/api/status/${jobId}`);
                if (!response.ok) {
                    throw new Error('Status check failed');
                }

                const data = await response.json();

                if (data.status === 'completed') {
                    clearInterval(pollingInterval); // Stop polling
                    statusText.textContent = 'Your wallpaper is ready!';
                    downloadLink.href = data.final_url;
                    downloadLink.classList.remove('hidden');
                    submitButton.disabled = false;
                    submitButton.textContent = 'Create Another Wallpaper';
                    loadGallery(); // Refresh the gallery with the new image
                } else if (data.status === 'failed') {
                    clearInterval(pollingInterval); // Stop polling
                    statusText.textContent = 'Sorry, something went wrong during processing.';
                    submitButton.disabled = false;
                    submitButton.textContent = 'Create Wallpaper';
                }
                // If status is 'pending' or 'processing', we just wait for the next interval
            } catch (error) {
                console.error('Polling error:', error);
                clearInterval(pollingInterval);
                statusText.textContent = 'Error checking status. Please check the gallery later.';
            }
        }, 3000); // Poll every 3 seconds
    }

    // --- GALLERY LOADING ---
    async function loadGallery() {
        try {
            const response = await fetch('/api/gallery');
            if (!response.ok) {
                throw new Error('Failed to load gallery');
            }

            const wallpapers = await response.json();
            galleryGrid.innerHTML = ''; // Clear existing gallery

            if (wallpapers && wallpapers.length > 0) {
                wallpapers.forEach(wallpaper => {
                    const img = document.createElement('img');
                    img.src = wallpaper.final_url;
                    img.alt = 'Dynamic Wallpaper';
                    img.loading = 'lazy'; // Lazy load images for performance
                    galleryGrid.appendChild(img);
                });
            } else {
                galleryGrid.innerHTML = '<p>No wallpapers have been created yet. Be the first!</p>';
            }
        } catch (error) {
            console.error('Gallery loading error:', error);
            galleryGrid.innerHTML = '<p>Could not load the gallery.</p>';
        }
    }

    // Initial load of the gallery when the page opens
    loadGallery();
});
