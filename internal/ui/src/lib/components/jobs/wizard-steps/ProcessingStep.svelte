<script>
    import { onMount } from "svelte";
    import { state as jobState, setStepValidity } from "$lib/stores/jobStore.svelte";
    
    // LOCAL STATE - SET DEFAULTS TO AVOID UNDEFINED
    let processing = $state({
        thumbnails: jobState.formData.data.processing?.thumbnails ?? true,
        metadata: jobState.formData.data.processing?.metadata ?? true,
        imageResize: jobState.formData.data.processing?.imageResize ?? false,
        imageWidth: jobState.formData.data.processing?.imageWidth ?? 1280,
        videoConvert: jobState.formData.data.processing?.videoConvert ?? false,
        videoFormat: jobState.formData.data.processing?.videoFormat ?? "mp4",
        extractText: jobState.formData.data.processing?.extractText ?? false,
        deduplication: jobState.formData.data.processing?.deduplication ?? true,
        headless: jobState.formData.data.processing?.headless ?? true
    });
    
    let isValid = $state(true);
    
    // INITIALIZE
    onMount(() => {
        validate();
        updateFormData();
    });
    
    // VALIDATE THE STEP
    function validate() {
        let valid = true;
        
        // VALIDATE IMAGE WIDTH IS A REASONABLE NUMBER IF RESIZE IS ENABLED
        if (processing.imageResize &&
            (processing.imageWidth < 100 || processing.imageWidth > 10000)) {
            valid = false;
        }
        
        isValid = valid;
        setStepValidity(4, valid);
        return valid;
    }
    
    // UPDATE FORM DATA WITH VALIDATION
    function updateFormData() {
        // CLONE THE PROCESSING OBJECT
        const updatedProcessing = { ...processing };
        
        // CONVERT WIDTH TO NUMBER
        updatedProcessing.imageWidth = parseInt(processing.imageWidth) || 1280;
        
        // ONLY UPDATE IF VALUES ACTUALLY CHANGED
        if (JSON.stringify(jobState.formData.data.processing) !== JSON.stringify(updatedProcessing)) {
            jobState.formData.data.processing = updatedProcessing;
        }
        
        validate();
    }
    
    // HANDLE FORM INPUT CHANGES
    function handleInputChange() {
        updateFormData();
    }
    
    // FIXED EFFECT - TRACK ALL PROCESSING PROPERTIES EXPLICITLY
    $effect(() => {
        // TRACK ALL PROPERTIES THAT SHOULD TRIGGER UPDATES
        const watchedProcessing = {
            thumbnails: processing.thumbnails,
            metadata: processing.metadata,
            imageResize: processing.imageResize,
            imageWidth: processing.imageWidth,
            videoConvert: processing.videoConvert,
            videoFormat: processing.videoFormat,
            extractText: processing.extractText,
            deduplication: processing.deduplication,
            headless: processing.headless
        };
        
        // NOW UPDATEFORMDATA ONLY RUNS WHEN THESE VALUES CHANGE
        updateFormData();
    });
</script>

<div>
    <h2 class="text-xl font-semibold mb-4">Processing Options</h2>
    <p class="text-dark-300 mb-6">
        Configure how downloaded assets are processed
    </p>
    
    <!-- MAIN OPTIONS -->
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
            <div class="flex items-center">
                <input
                    id="headless-mode"
                    type="checkbox"
                    bind:checked={processing.headless}
                    onchange={handleInputChange}
                    class="checkbox checkbox-primary h-4 w-4"
                />
                <label
                    for="headless-mode"
                    class="ml-2 block text-sm text-white"
                >
                    Headless Mode
                </label>
                <div class="ml-2 text-xs text-dark-400">
                    (Run browser without visible UI, faster but less interactive)
                </div>
            </div>
        </div>
    </div>
    
    <!-- IMAGE PROCESSING -->
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
    
    <!-- VIDEO PROCESSING -->
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
    
    <!-- TEXT EXTRACTION -->
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
</div>
