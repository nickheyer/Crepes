<script>
    import { fade, fly } from "svelte/transition";
    import { createEventDispatcher } from "svelte";
    import Button from "$lib/components/common/Button.svelte";
    import { formatFileSize, formatDate } from "$lib/utils/formatters";
    import {
        state as assetState,
        removeAsset,
        regenerateAssetThumbnail,
        closeAssetViewer
    } from "$lib/stores/assetStore.svelte";
    import { addToast } from "$lib/stores/uiStore.svelte";
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
        if (!assetState.selectedAsset) return -1;
        // ENSURE ASSETS IS ALWAYS AN ARRAY
        const assetArray = Array.isArray(assetState.assets) ? assetState.assets : [];
        return assetArray.findIndex((a) => a.id === assetState.selectedAsset.id);
    });
    
    // GET ASSETS LENGTH SAFELY
    let assetsLength = $derived(() => {
        // ENSURE ASSETS IS ALWAYS AN ARRAY
        const assetArray = Array.isArray(assetState.assets) ? assetState.assets : [];
        return assetArray.length;
    });
    
    // NAVIGATE TO NEXT/PREVIOUS ASSET
    function navigateToPrev() {
        if (currentIndex > 0 && Array.isArray(assetState.assets)) {
            assetState.selectedAsset = assetState.assets[currentIndex - 1];
        }
    }
    
    function navigateToNext() {
        if (currentIndex < assetsLength - 1 && Array.isArray(assetState.assets)) {
            assetState.selectedAsset = assetState.assets[currentIndex + 1];
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
            await removeAsset(assetState.selectedAsset.id);
            addToast("Asset deleted successfully", "success");
            // Navigate to next asset or close viewer if no more assets
            if (Array.isArray(assetState.assets)) {
                if (currentIndex < assetState.assets.length - 1) {
                    assetState.selectedAsset = assetState.assets[currentIndex + 1];
                } else if (currentIndex > 0) {
                    assetState.selectedAsset = assetState.assets[currentIndex - 1];
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
            await regenerateAssetThumbnail(assetState.selectedAsset.id);
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
        if (!assetState.selectedAsset.localPath) {
            addToast("This asset does not have a local file", "error");
            return;
        }
        const link = document.createElement("a");
        link.href = `/assets/${assetState.selectedAsset.localPath}`;
        link.download = assetState.selectedAsset.title || "download";
        link.click();
    }
</script>
<svelte:window onkeydown={handleKeydown} />
{#if assetState.selectedAsset}
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
                        {assetState.selectedAsset.title || "Untitled"}
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
                {#if assetState.selectedAsset.type === "image" && assetState.selectedAsset.localPath}
                    <img
                        src={`/assets/${assetState.selectedAsset.localPath}`}
                        alt={assetState.selectedAsset.title || "Image"}
                        class="max-h-full max-w-full object-contain"
                    />
                {:else if assetState.selectedAsset.type === "video" && assetState.selectedAsset.localPath}
                    <video
                        src={`/assets/${assetState.selectedAsset.localPath}`}
                        controls
                        autoplay
                        class="max-h-full max-w-full"
                    >
                        <track kind="captions" />
                        Your browser does not support the video tag.
                    </video>
                {:else if assetState.selectedAsset.type === "audio" && assetState.selectedAsset.localPath}
                    <div class="bg-base-800 p-6 rounded-lg w-full max-w-2xl">
                        <div class="mb-4 flex justify-center">
                            <div
                                class="w-32 h-32 bg-base-700 rounded-full flex items-center justify-center text-4xl"
                            >
                                🔊
                            </div>
                        </div>
                        <h3 class="text-xl font-medium text-center mb-4">
                            {assetState.selectedAsset.title || "Audio File"}
                        </h3>
                        <audio
                            src={`/assets/${assetState.selectedAsset.localPath}`}
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
                                {#if assetState.selectedAsset.type === "document"}
                                    📄
                                {:else}
                                    ❓
                                {/if}
                            </div>
                            <h3 class="text-xl font-medium mb-2">
                                {assetState.selectedAsset.title || "File"}
                            </h3>
                            <p class="text-dark-400 mb-4">
                                {formatFileSize(assetState.selectedAsset.size)}
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
                        <p>{assetState.selectedAsset.title || "Untitled"}</p>
                    </div>
                    {#if assetState.selectedAsset.description}
                        <div>
                            <h4 class="text-sm font-medium text-dark-300 mb-1">
                                Description
                            </h4>
                            <p class="text-sm">{assetState.selectedAsset.description}</p>
                        </div>
                    {/if}
                    <div>
                        <h4 class="text-sm font-medium text-dark-300 mb-1">
                            Type
                        </h4>
                        <p>{assetState.selectedAsset.type}</p>
                    </div>
                    <div>
                        <h4 class="text-sm font-medium text-dark-300 mb-1">
                            Size
                        </h4>
                        <p>{formatFileSize(assetState.selectedAsset.size)}</p>
                    </div>
                    {#if assetState.selectedAsset.date}
                        <div>
                            <h4 class="text-sm font-medium text-dark-300 mb-1">
                                Date
                            </h4>
                            <p>{formatDate(assetState.selectedAsset.date)}</p>
                        </div>
                    {/if}
                    {#if assetState.selectedAsset.url}
                        <div>
                            <h4 class="text-sm font-medium text-dark-300 mb-1">
                                Source URL
                            </h4>
                            <p class="text-sm break-all">
                                <a
                                    href={assetState.selectedAsset.url}
                                    target="_blank"
                                    rel="noopener noreferrer"
                                    class="text-primary-400 hover:text-primary-300"
                                >
                                    {assetState.selectedAsset.url}
                                </a>
                            </p>
                        </div>
                    {/if}
                    {#if assetState.selectedAsset.metadata && Object.keys(assetState.selectedAsset.metadata).length > 0}
                        <div>
                            <h4 class="text-sm font-medium text-dark-300 mb-2">
                                Metadata
                            </h4>
                            <div class="bg-base-900 rounded-md p-3 text-sm">
                                {#each Object.entries(assetState.selectedAsset.metadata) as [key, value]}
                                    <div
                                        class="mb-1 pb-1 border-b border-dark-800 last:border-b-0 last:mb-0 last:pb-0"
                                    >
                                        <span class="text-dark-400">{key}:</span>
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
