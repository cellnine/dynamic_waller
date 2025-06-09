document.addEventListener('DOMContentLoaded', () => {
    // --- Element References ---
    const form = document.getElementById('upload-form');
    const submitButton = document.getElementById('submit-button');

    // Uploader specific elements
    const lightUploader = document.getElementById('light-uploader');
    const lightFileInput = document.getElementById('light-file');
    const lightPreview = document.getElementById('light-preview');

    const darkUploader = document.getElementById('dark-uploader');
    const darkFileInput = document.getElementById('dark-file');
    const darkPreview = document.getElementById('dark-preview');

    // Status/Result elements
    const statusArea = document.getElementById('status-area');
    const statusText = document.getElementById('status-text');

    // Gallery
    const galleryGrid = document.getElementById('gallery-grid');
    let pollingInterval;

    // --- Logic ---

    // Reusable function to set up our uploader cards
    function setupUploader(uploaderElement, fileInputElement, previewElement) {
        uploaderElement.addEventListener('click', () => fileInputElement.click());

        fileInputElement.addEventListener('change', () => {
            const file = fileInputElement.files[0];
            if (file) {
                const reader = new FileReader();
                reader.onload = (e) => {
                    previewElement.src = e.target.result;
                    uploaderElement.classList.add('has-preview');
                };
                reader.readAsDataURL(file);
            }
        });
    }

    setupUploader(lightUploader, lightFileInput, lightPreview);
    setupUploader(darkUploader, darkFileInput, darkPreview);

    // Main form submission handler
    form.addEventListener('submit', async (e) => {
        e.preventDefault();

        if (!lightFileInput.files[0] || !darkFileInput.files[0]) {
            alert('Please select both a light and a dark image.');
            return;
        }

        const formData = new FormData();
        formData.append('light', lightFileInput.files[0]);
        formData.append('dark', darkFileInput.files[0]);

        // Update UI to show processing state
        submitButton.disabled = true;
        submitButton.textContent = 'Generating...';
        statusArea.classList.remove('hidden');
        statusText.textContent = 'Uploading images...';

        try {
            const response = await fetch('/api/create', { method: 'POST', body: formData });
            if (!response.ok) throw new Error(`Server error: ${response.statusText}`);

            const result = await response.json();
            statusText.textContent = 'Processing... this may take a moment.';
            pollStatus(result.id);

        } catch (error) {
            statusText.textContent = `Error: ${error.message}. Please try again.`;
            resetSubmitButton();
        }
    });

    // Function to poll for job status
    function pollStatus(jobId) {
        if (pollingInterval) clearInterval(pollingInterval);

        pollingInterval = setInterval(async () => {
            try {
                const response = await fetch(`/api/status/${jobId}`);
                if (!response.ok) throw new Error('Status check failed');

                const data = await response.json();

                if (data.status === 'completed') {
                    clearInterval(pollingInterval);
                    statusText.textContent = 'Your wallpaper is ready!';
                    setDownloadButton(data.final_url);
                } else if (data.status === 'failed') {
                    clearInterval(pollingInterval);
                    statusText.textContent = 'Sorry, something went wrong during processing.';
                    resetSubmitButton();
                }
            } catch (error) {
                clearInterval(pollingInterval);
                statusText.textContent = 'Error checking status. Please check the gallery later.';
            }
        }, 3000); // Poll every 3 seconds
    }


    function setDownloadButton(url) {
        submitButton.remove(); // Remove the old button

        const downloadButton = document.createElement('a');
        downloadButton.href = url;
        downloadButton.textContent = 'Download';
        downloadButton.className = 'primary-action success';
        downloadButton.setAttribute('download', ''); // Make the browser download the file

        form.appendChild(downloadButton);
    }

    function resetSubmitButton() {
        submitButton.disabled = false;
        submitButton.textContent = 'Generate';
    }


});
