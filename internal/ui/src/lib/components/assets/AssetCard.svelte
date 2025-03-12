<script>
    import { createEventDispatcher } from "svelte";
    import { formatFileSize, formatDate } from "$lib/utils/formatters";
    import {
        removeAsset,
        regenerateAssetThumbnail,
    } from "$lib/stores/assetStore.svelte";
    import { addToast } from "$lib/stores/uiStore.svelte";
    
    // PROPS USING SVELTE 5 $PROPS RUNE
    let { asset = {} } = $props();
    
    // LOCAL STATE USING $STATE RUNE
    let isMenuOpen = $state(false);
    let loading = $state(false);
    
    // CREATE DISPATCH FUNCTION
    const dispatch = createEventDispatcher();
    
    function toggleMenu() {
        isMenuOpen = !isMenuOpen;
    }
    
    function closeMenu() {
        isMenuOpen = false;
    }
    
    function viewAsset() {
        dispatch("view");
    }
    
    async function handleDelete() {
        if (!confirm("Are you sure you want to delete this asset?")) {
            return;
        }
        
        try {
            loading = true;
            await removeAsset(asset.id);
            addToast("Asset deleted", "success");
        } catch (error) {
            addToast(`Failed to delete asset: ${error.message}`, "error");
        } finally {
            loading = false;
            closeMenu();
        }
    }
    
    async function handleRegenerate() {
        try {
            loading = true;
            await regenerateAssetThumbnail(asset.id);
            addToast("Thumbnail regenerated", "success");
        } catch (error) {
            addToast(
                `Failed to regenerate thumbnail: ${error.message}`,
                "error",
            );
        } finally {
            loading = false;
            closeMenu();
        }
    }
    
    function downloadAsset() {
        if (!asset.localPath) {
            addToast("This asset has no local file", "error");
            return;
        }
        
        const link = document.createElement("a");
        link.href = `/api/assets/${asset.localPath}`;
        link.download = asset.title || "download";
        link.click();
        closeMenu();
    }
</script>

<div
    class="bg-base-800 rounded-lg overflow-hidden shadow hover:shadow-lg transition-shadow duration-200"
>
    <!-- THUMBNAIL/PREVIEW -->
    <button
        class="w-full aspect-square bg-base-700 overflow-hidden group relative"
        onclick={viewAsset}
    >
        {#if asset.thumbnailPath}
            <img
                src={`/api/thumbnails/${asset.thumbnailPath}`}
                alt={asset.title || "Asset"}
                class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-200"
            />
        {:else}
            <div
                class="w-full h-full flex items-center justify-center text-4xl bg-base-750"
            >
                {#if asset.type === "image"}
                    🖼️
                {:else if asset.type === "video"}
                    🎬
                {:else if asset.type === "audio"}
                    🔊
                {:else if asset.type === "document"}
                    📄
                {:else}
                    ❓
                {/if}
            </div>
        {/if}
        
        <!-- OVERLAY WITH VIEW BUTTON ON HOVER -->
        <div
            class="absolute inset-0 bg-black bg-opacity-0 group-hover:bg-opacity-30 transition-all duration-200 flex items-center justify-center opacity-0 group-hover:opacity-100"
        >
            <span
                class="bg-primary-600 text-white rounded-full px-3 py-1 text-sm font-medium"
            >
                VIEW
            </span>
        </div>
        
        <!-- TYPE BADGE -->
        <div
            class="absolute top-2 right-2 px-2 py-0.5 rounded-full bg-base-900 bg-opacity-80 text-xs font-medium text-white"
        >
            {asset.type}
        </div>
    </button>
    
    <!-- CONTENT -->
    <div class="p-3">
        <div class="flex items-start justify-between">
            <h3
                class="text-sm font-medium truncate"
                title={asset.title || "Untitled"}
            >
                {asset.title || "Untitled"}
            </h3>
            
            <!-- MENU BUTTON -->
            <div class="relative ml-2">
                <button
                    class="p-1 rounded-full hover:bg-base-700 text-dark-300 hover:text-white focus:outline-none"
                    onclick={toggleMenu}
                    aria-label="Asset options"
                >
                    <svg
                        xmlns="http://www.w3.org/2000/svg"
                        class="h-4 w-4"
                        viewBox="0 0 20 20"
                        fill="currentColor"
                    >
                        <path
                            d="M10 6a2 2 0 110-4 2 2 0 010 4zM10 12a2 2 0 110-4 2 2 0 010 4zM10 18a2 2 0 110-4 2 2 0 010 4z"
                        />
                    </svg>
                </button>
                
                {#if isMenuOpen}
                    <div
                        class="absolute right-0 mt-1 w-48 rounded-md shadow-lg py-1 bg-base-700 ring-1 ring-black ring-opacity-5 z-10"
                        onmousedown={(e) => e.stopPropagation()}
                        transition:fade={{ duration: 200 }}
                        onkeydown={() => {}}
                        role="button"
                        aria-label="Close modal"
                        tabindex="0"
                    >
                        <button
                            class="block w-full text-left px-4 py-2 text-sm text-white hover:bg-base-600"
                            onclick={viewAsset}
                        >
                            View Details
                        </button>
                        <button
                            class="block w-full text-left px-4 py-2 text-sm text-white hover:bg-base-600"
                            onclick={downloadAsset}
                        >
                            Download
                        </button>
                        <button
                            class="block w-full text-left px-4 py-2 text-sm text-white hover:bg-base-600"
                            onclick={handleRegenerate}
                            disabled={loading}
                        >
                            Regenerate Thumbnail
                        </button>
                        <button
                            class="block w-full text-left px-4 py-2 text-sm text-danger-400 hover:bg-base-600"
                            onclick={handleDelete}
                            disabled={loading}
                        >
                            DElete
                        </button>
                    </div>
                {/if}
            </div>
        </div>
        
        {#if asset.description}
            <p
                class="mt-1 text-xs text-dark-300 line-clamp-2"
                title={asset.description}
            >
                {asset.description}
            </p>
        {/if}
        
        <div
            class="mt-2 flex justify-between items-center text-xs text-dark-400"
        >
            <span title={asset.date ? formatDate(asset.date) : "Unknown date"}>
                {asset.date
                    ? formatDate(asset.date, "MMM D, YYYY")
                    : "Unknown date"}
            </span>
            <span>{formatFileSize(asset.size)}</span>
        </div>
    </div>
</div>

<!-- CLOSE MENU WHEN CLICKING OUTSIDE -->
{#if isMenuOpen}
    <div
        class="fixed inset-0 z-0"
        onclick={closeMenu}
        onkeydown={(e) => e.key === "Escape" && closeMenu()}
        transition:fade={{ duration: 200 }}
        role="button"
        aria-label="Close menu"
        tabindex="0"
    ></div>
{/if}
