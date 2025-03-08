<script>
    import { fade, fly } from "svelte/transition";
    import { createEventDispatcher } from "svelte";
    import Button from "$lib/components/common/Button.svelte";
    import { formatFileSize, formatDate } from "$lib/utils/formatters";
    import {
        assets,
        selectedAsset,
        closeAssetViewer,
        removeAsset,
        regenerateAssetThumbnail,
    } from "$lib/stores/assetStore";
    import { addToast } from "$lib/stores/uiStore";
    import {
        Trash,
        ChevronLeft,
        ChevronRight,
        CloudDownload,
        X,
        RefreshCw
    } from "lucide-svelte";
    
    // LOCAL STATE
    let loading = $state(false);
    
    // GET CURRENT INDEX
    let currentIndex = $derived(() => {
        if (!$selectedAsset) return -1;
        // ENSURE ASSETS IS ALWAYS AN ARRAY
        const assetArray = Array.isArray($assets) ? $assets : [];
        return assetArray.findIndex((a) => a.id === $selectedAsset.id);
    });
    
    // GET ASSETS LENGTH SAFELY
    let assetsLength = $derived(() => {
        // ENSURE ASSETS IS ALWAYS AN ARRAY
        const assetArray = Array.isArray($assets) ? $assets : [];
        return assetArray.length;
    });
    
    // NAVIGATE TO NEXT/PREVIOUS ASSET
    function navigateToPrev() {
        if (currentIndex > 0 && Array.isArray($assets)) {
            $selectedAsset = $assets[currentIndex - 1];
        }
    }
    
    function navigateToNext() {
        if (currentIndex < assetsLength - 1 && Array.isArray($assets)) {
            $selectedAsset = $assets[currentIndex + 1];
        }
    }
    
    // HANDLE KEYBOARD NAVIGATION
    function handleKeydown(event) {
        if (event.key === "ArrowLeft") {
            navigateToPrev();
        } else if (event.key === "ArrowRight") {
            navigateToNext();
        } else if (event.key === "Escape") {
            closeAssetViewer();
        }
    }
    
    // ASSET ACTIONS
    async function handleDelete() {
        if (!confirm("Are you sure you want to delete this asset?")) {
            return;
        }
        try {
            loading = true;
            await removeAsset($selectedAsset.id);
            addToast("Asset deleted successfully", "success");
            // Navigate to next asset or close viewer if no more assets
            if (Array.isArray($assets)) {
                if (currentIndex < $assets.length - 1) {
                    $selectedAsset = $assets[currentIndex + 1];
                } else if (currentIndex > 0) {
                    $selectedAsset = $assets[currentIndex - 1];
                } else {
                    closeAssetViewer();
                }
            } else {
                closeAssetViewer();
            }
        } catch (error) {
            addToast(`Failed to delete asset: ${error.message}`, "error");
        } finally {
            loading = false;
        }
    }
    
    async function handleRegenerate() {
        try {
            loading = true;
            await regenerateAssetThumbnail($selectedAsset.id);
            addToast("Thumbnail regenerated successfully", "success");
        } catch (error) {
            addToast(
                `Failed to regenerate thumbnail: ${error.message}`,
                "error",
            );
        } finally {
            loading = false;
        }
    }
    
    function downloadAsset() {
        if (!$selectedAsset.localPath) {
            addToast("This asset does not have a local file", "error");
            return;
        }
        const link = document.createElement("a");
        link.href = `/assets/${$selectedAsset.localPath}`;
        link.download = $selectedAsset.title || "download";
        link.click();
    }
</script>
<svelte:window onkeydown={handleKeydown} />
{#if $selectedAsset}
    <div
        class="fixed inset-0 z-50 bg-black bg-opacity-90 flex flex-col"
        transition:fade={{ duration: 200 }}
    >
        <!-- Header toolbar -->
        <div class="p-4 flex justify-between items-center">
            <div class="text-white flex items-center space-x-2">
                <button
                    class="p-2 rounded-full hover:bg-base-700 focus:outline-none"
                    onclick={closeAssetViewer}
                    aria-label="Close viewer"
                >
                    <X class="h-5 w-5" />
                </button>
                <div class="text-sm">
                    <div class="font-medium">
                        {$selectedAsset.title || "Untitled"}
                    </div>
                    <div class="text-xs text-dark-300">
                        {currentIndex + 1} of {assetsLength}
                    </div>
                </div>
            </div>
            <div class="flex items-center space-x-2">
                <Button variant="outline" size="sm" onclick={downloadAsset}>
                    <CloudDownload class="h-4 w-4 mr-1" />
                    Download
                </Button>
                <Button
                    variant="outline"
                    size="sm"
                    onclick={handleRegenerate}
                    disabled={loading}
                >
                    <RefreshCw class="h-4 w-4 mr-1" />
                    Regenerate
                </Button>
                <Button
                    variant="danger"
                    size="sm"
                    onclick={handleDelete}
                    disabled={loading}
                >
                    <Trash class="h-4 w-4 mr-1" />
                    Delete
                </Button>
            </div>
        </div>
        <!-- Main content -->
        <div class="flex-1 flex">
            <!-- Previous button -->
            {#if currentIndex > 0}
                <button
                    class="absolute left-4 top-1/2 transform -translate-y-1/2 z-10 p-2 rounded-full bg-base-800 bg-opacity-60 hover:bg-opacity-80 text-white focus:outline-none"
                    onclick={navigateToPrev}
                    aria-label="Previous asset"
                >
                    <ChevronLeft class="h-6 w-6" />
                </button>
            {/if}
            <!-- Content area -->
            <div class="flex-1 flex items-center justify-center p-4">
                {#if $selectedAsset.type === "image" && $selectedAsset.localPath}
                    <img
                        src={`/assets/${$selectedAsset.localPath}`}
                        alt={$selectedAsset.title || "Image"}
                        class="max-h-full max-w-full object-contain"
                    />
                {:else if $selectedAsset.type === "video" && $selectedAsset.localPath}
                    <video
                        src={`/assets/${$selectedAsset.localPath}`}
                        controls
                        autoplay
                        class="max-h-full max-w-full"
                    >
                        <track kind="captions" />
                        Your browser does not support the video tag.
                    </video>
                {:else if $selectedAsset.type === "audio" && $selectedAsset.localPath}
                    <div class="bg-base-800 p-6 rounded-lg w-full max-w-2xl">
                        <div class="mb-4 flex justify-center">
                            <div
                                class="w-32 h-32 bg-base-700 rounded-full flex items-center justify-center text-4xl"
                            >
                                🔊
                            </div>
                        </div>
                        <h3 class="text-xl font-medium text-center mb-4">
                            {$selectedAsset.title || "Audio File"}
                        </h3>
                        <audio
                            src={`/assets/${$selectedAsset.localPath}`}
                            controls
                            class="w-full"
                            autoplay
                        >
                            Your browser does not support the audio element.
                        </audio>
                    </div>
                {:else}
                    <div class="bg-base-800 p-6 rounded-lg">
                        <div class="flex flex-col items-center">
                            <div class="text-6xl mb-4">
                                {#if $selectedAsset.type === "document"}
                                    📄
                                {:else}
                                    ❓
                                {/if}
                            </div>
                            <h3 class="text-xl font-medium mb-2">
                                {$selectedAsset.title || "File"}
                            </h3>
                            <p class="text-dark-400 mb-4">
                                {formatFileSize($selectedAsset.size)}
                            </p>
                            <Button variant="primary" onclick={downloadAsset}>
                                <CloudDownload class="h-5 w-5 mr-2" />
                                Download File
                            </Button>
                        </div>
                    </div>
                {/if}
            </div>
            <!-- Next button -->
            {#if currentIndex < assetsLength - 1}
                <button
                    class="absolute right-4 top-1/2 transform -translate-y-1/2 z-10 p-2 rounded-full bg-base-800 bg-opacity-60 hover:bg-opacity-80 text-white focus:outline-none"
                    onclick={navigateToNext}
                    aria-label="Next asset"
                >
                    <ChevronRight class="h-6 w-6" />
                </button>
            {/if}
            <!-- Metadata panel -->
            <div
                class="w-80 bg-base-800 border-l border-dark-700 p-4 overflow-y-auto"
            >
                <h3 class="text-lg font-medium mb-4">Asset Details</h3>
                <div class="space-y-4">
                    <div>
                        <h4 class="text-sm font-medium text-dark-300 mb-1">
                            Title
                        </h4>
                        <p>{$selectedAsset.title || "Untitled"}</p>
                    </div>
                    {#if $selectedAsset.description}
                        <div>
                            <h4 class="text-sm font-medium text-dark-300 mb-1">
                                Description
                            </h4>
                            <p class="text-sm">{$selectedAsset.description}</p>
                        </div>
                    {/if}
                    <div>
                        <h4 class="text-sm font-medium text-dark-300 mb-1">
                            Type
                        </h4>
                        <p>{$selectedAsset.type}</p>
                    </div>
                    <div>
                        <h4 class="text-sm font-medium text-dark-300 mb-1">
                            Size
                        </h4>
                        <p>{formatFileSize($selectedAsset.size)}</p>
                    </div>
                    {#if $selectedAsset.date}
                        <div>
                            <h4 class="text-sm font-medium text-dark-300 mb-1">
                                Date
                            </h4>
                            <p>{formatDate($selectedAsset.date)}</p>
                        </div>
                    {/if}
                    {#if $selectedAsset.url}
                        <div>
                            <h4 class="text-sm font-medium text-dark-300 mb-1">
                                Source URL
                            </h4>
                            <p class="text-sm break-all">
                                <a
                                    href={$selectedAsset.url}
                                    target="_blank"
                                    rel="noopener noreferrer"
                                    class="text-primary-400 hover:text-primary-300"
                                >
                                    {$selectedAsset.url}
                                </a>
                            </p>
                        </div>
                    {/if}
                    {#if $selectedAsset.metadata && Object.keys($selectedAsset.metadata).length > 0}
                        <div>
                            <h4 class="text-sm font-medium text-dark-300 mb-2">
                                Metadata
                            </h4>
                            <div class="bg-base-900 rounded-md p-3 text-sm">
                                {#each Object.entries($selectedAsset.metadata) as [key, value]}
                                    <div
                                        class="mb-1 pb-1 border-b border-dark-800 last:border-b-0 last:mb-0 last:pb-0"
                                    >
                                        <span class="text-dark-400">{key}:</span
                                        >
                                        {value}
                                    </div>
                                {/each}
                            </div>
                        </div>
                    {/if}
                </div>
            </div>
        </div>
    </div>
{/if}
