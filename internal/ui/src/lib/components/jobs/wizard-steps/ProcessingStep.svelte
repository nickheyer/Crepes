<script>
    import { onMount } from "svelte";
    import { createEventDispatcher } from "svelte";
    
    // Create dispatch function
    const dispatch = createEventDispatcher();
    
    // Props
    let { formData = {} } = $props();
    
    // Local state - breaking the reactive cycle
    let processing = $state(
        formData.processing || {
            thumbnails: true,
            metadata: true,
            imageResize: false,
            imageWidth: 1280,
            videoConvert: false,
            videoFormat: "mp4",
            extractText: false,
            deduplication: true,
        }
    );
    
    let isValid = $state(true);
    let shouldUpdateFormData = $state(false);
    
    // Initialize
    onMount(() => {
        validate();
    });
    
    // Validate the step
    function validate() {
        let valid = true;
        // Validate image width is a reasonable number if resize is enabled
        if (
            processing.imageResize &&
            (processing.imageWidth < 100 || processing.imageWidth > 10000)
        ) {
            valid = false;
        }
        isValid = valid;
        dispatch("validate", isValid);
        return isValid;
    }
    
    // Update form data WITHOUT a reactive effect
    function updateFormData() {
        if (!shouldUpdateFormData) return;
        
        const updatedData = {
            ...formData,
            processing: { ...processing },
        };
        
        // Convert width to number
        updatedData.processing.imageWidth = parseInt(processing.imageWidth) || 1280;
        
        const isValid = validate();
        if (isValid) {
            dispatch("update", updatedData);
        }
        
        // Set flag back to false to prevent infinite loop
        shouldUpdateFormData = false;
    }
    
    // Handle form input changes
    function handleInputChange() {
        shouldUpdateFormData = true;
        validate();
        // Use setTimeout to break the reactive cycle
        setTimeout(updateFormData, 0);
    }
</script>
<div>
    <h2 class="text-xl font-semibold mb-4">Processing Options</h2>
    <p class="text-dark-300 mb-6">
        Configure how downloaded assets are processed
    </p>
    <!-- Main options -->
    <div class="bg-base-800 rounded-lg p-4 mb-6">
        <h3 class="text-sm font-medium mb-4">Basic Processing</h3>
        <div class="grid grid-cols-1 gap-4">
            <div class="flex items-center">
                <input
                    id="generate-thumbnails"
                    type="checkbox"
                    bind:checked={processing.thumbnails}
                    onchange={handleInputChange}
                    class="checkbox checkbox-primary h-4 w-4"
                />
                <label
                    for="generate-thumbnails"
                    class="ml-2 block text-sm text-white"
                >
                    Generate Thumbnails
                </label>
                <div class="ml-2 text-xs text-dark-400">
                    (Creates preview thumbnails for images and videos)
                </div>
            </div>
            <div class="flex items-center">
                <input
                    id="extract-metadata"
                    type="checkbox"
                    bind:checked={processing.metadata}
                    onchange={handleInputChange}
                    class="checkbox checkbox-primary h-4 w-4"
                />
                <label
                    for="extract-metadata"
                    class="ml-2 block text-sm text-white"
                >
                    Extract Metadata
                </label>
                <div class="ml-2 text-xs text-dark-400">
                    (Extract titles, descriptions, dates from pages)
                </div>
            </div>
            <div class="flex items-center">
                <input
                    id="deduplication"
                    type="checkbox"
                    bind:checked={processing.deduplication}
                    onchange={handleInputChange}
                    class="checkbox checkbox-primary h-4 w-4"
                />
                <label
                    for="deduplication"
                    class="ml-2 block text-sm text-white"
                >
                    Enable Deduplication
                </label>
                <div class="ml-2 text-xs text-dark-400">
                    (Skip downloading duplicate assets)
                </div>
            </div>
        </div>
    </div>
    <!-- Image processing -->
    <div class="bg-base-800 rounded-lg p-4 mb-6">
        <h3 class="text-sm font-medium mb-4">Image Processing</h3>
        <div class="grid grid-cols-1 gap-4">
            <div class="flex items-center">
                <input
                    id="image-resize"
                    type="checkbox"
                    bind:checked={processing.imageResize}
                    onchange={handleInputChange}
                    class="checkbox checkbox-primary h-4 w-4"
                />
                <label for="image-resize" class="ml-2 block text-sm text-white">
                    Resize Large Images
                </label>
            </div>
            {#if processing.imageResize}
                <div class="ml-6">
                    <label
                        for="image-width"
                        class="block text-sm font-medium text-dark-300 mb-1"
                    >
                        Maximum Width (px)
                    </label>
                    <input
                        id="image-width"
                        type="number"
                        min="100"
                        max="10000"
                        bind:value={processing.imageWidth}
                        onchange={handleInputChange}
                        class="input input-bordered w-full max-w-xs"
                    />
                    <p class="mt-1 text-xs text-dark-400">
                        Images larger than this will be resized (preserving
                        aspect ratio)
                    </p>
                </div>
            {/if}
        </div>
    </div>
    <!-- Video processing -->
    <div class="bg-base-800 rounded-lg p-4 mb-6">
        <h3 class="text-sm font-medium mb-4">Video Processing</h3>
        <div class="grid grid-cols-1 gap-4">
            <div class="flex items-center">
                <input
                    id="video-convert"
                    type="checkbox"
                    bind:checked={processing.videoConvert}
                    onchange={handleInputChange}
                    class="checkbox checkbox-primary h-4 w-4"
                />
                <label
                    for="video-convert"
                    class="ml-2 block text-sm text-white"
                >
                    Convert Videos to Standard Format
                </label>
            </div>
            {#if processing.videoConvert}
                <div class="ml-6">
                    <label
                        for="video-format"
                        class="block text-sm font-medium text-dark-300 mb-1"
                    >
                        Output Format
                    </label>
                    <select
                        id="video-format"
                        bind:value={processing.videoFormat}
                        onchange={handleInputChange}
                        class="select select-bordered w-full max-w-xs"
                    >
                        <option value="mp4">MP4 (H.264)</option>
                        <option value="webm">WebM (VP9)</option>
                        <option value="mkv">MKV</option>
                    </select>
                    <p class="mt-1 text-xs text-dark-400">
                        All downloaded videos will be converted to this format
                    </p>
                </div>
            {/if}
        </div>
    </div>
    <!-- Text extraction -->
    <div class="bg-base-800 rounded-lg p-4">
        <h3 class="text-sm font-medium mb-4">Text Extraction</h3>
        <div class="grid grid-cols-1 gap-4">
            <div class="flex items-center">
                <input
                    id="extract-text"
                    type="checkbox"
                    bind:checked={processing.extractText}
                    onchange={handleInputChange}
                    class="checkbox checkbox-primary h-4 w-4"
                />
                <label for="extract-text" class="ml-2 block text-sm text-white">
                    Extract Text from Documents
                </label>
                <div class="ml-2 text-xs text-dark-400">
                    (Extract readable text from PDFs and other documents)
                </div>
            </div>
        </div>
    </div>
    <!-- Help note -->
    <div class="mt-6 bg-base-850 rounded-lg p-4">
        <div class="flex">
            <div class="flex-shrink-0">
                <svg
                    xmlns="http://www.w3.org/2000/svg"
                    class="h-6 w-6 text-primary-400"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                >
                    <path
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        stroke-width="2"
                        d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                    />
                </svg>
            </div>
            <div class="ml-3">
                <p class="text-sm text-dark-300">
                    More processing options requires more CPU power and storage
                    space. For large scraping jobs, you may want to disable some
                    processing features.
                </p>
            </div>
        </div>
    </div>
</div>
