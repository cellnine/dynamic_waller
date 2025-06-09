document.addEventListener('DOMContentLoaded', () => {

    async function loadGallery() {
        const galleryGrid = document.getElementById('gallery-grid');

        try {
            // Step 1: Try to fetch the data from the API.
            const response = await fetch('/api/gallery');

            // Step 2: Check if the server responded with an error status (like 404 or 500).
            if (!response.ok) {
                throw new Error('Failed to load gallery data from the server.');
            }

            // Step 3: Try to parse the response as JSON.
            const wallpapers = await response.json();

            // Step 4 (The Happy Path): If everything above succeeded, clear the "Loading..."
            // message and build the gallery. This is where your code snippet belongs.
            galleryGrid.innerHTML = '';

            if (wallpapers && wallpapers.length > 0) {
                wallpapers.forEach(wallpaper => {
                    const link = document.createElement('a');
                    link.href = wallpaper.final_url;
                    link.setAttribute('download', '');

                    const img = document.createElement('img');
                    img.src = wallpaper.preview_url;
                    img.alt = 'Dynamic Wallpaper Preview';
                    img.loading = 'lazy';

                    link.appendChild(img);
                    galleryGrid.appendChild(link);
                });
            } else {
                // This is still a "happy path" - we got a valid, empty response.
                galleryGrid.innerHTML = '<p>No wallpapers have been created yet.</p>';
            }

        } catch (error) {
            // The Sad Path: If any part of the 'try' block failed, we end up here.
            console.error('Gallery loading error:', error);
            galleryGrid.innerHTML = '<p>Could not load the gallery. Please try again later.</p>';
        }
    }

    // Load the gallery as soon as the page opens
    loadGallery();
});
