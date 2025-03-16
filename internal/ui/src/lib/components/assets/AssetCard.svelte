<script>
    import { createEventDispatcher } from "svelte";
    import { fade } from "svelte/transition";
    import { formatFileSize, formatDate } from "$lib/utils/formatters";
    import { removeAsset, regenerateAssetThumbnail } from "$lib/stores/assetStore.svelte";
    import { addToast } from "$lib/stores/uiStore.svelte";
    import Button from "$lib/components/common/Button.svelte";
    
    // PROPS USING SVELTE 5 RUNES
    let { 
        asset = {},
        showActions = true,
        size = "md",  // sm, md, lg
        className = "",
        onClick = null
    } = $props();
    
    // LOCAL STATE USING RUNES
    let isMenuOpen = $state(false);
    let loading = $state(false);
    
    // CREATE EVENT DISPATCHER
    const dispatch = createEventDispatcher();
    
    // SIZE CLASSES
    const sizeClasses = {
        sm: "max-w-xs",
        md: "",
        lg: "min-w-80"
    };
    
    // FUNCTIONS
    function toggleMenu(event) {
        event.stopPropagation();
        isMenuOpen = !isMenuOpen;
    }
    
    function closeMenu() {
        isMenuOpen = false;
    }
    
    function viewAsset() {
        if (onClick) {
            onClick(asset);
        } else {
            dispatch("view", asset);
        }
    }
    
    async function handleDelete(event) {
        event.stopPropagation();
        
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
    
    async function handleRegenerate(event) {
        event.stopPropagation();
        
        try {
            loading = true;
            await regenerateAssetThumbnail(asset.id);
            addToast("Thumbnail regenerated", "success");
        } catch (error) {
            addToast(`Failed to regenerate thumbnail: ${error.message}`, "error");
        } finally {
            loading = false;
            closeMenu();
        }
    }
    
    function downloadAsset(event) {
        event.stopPropagation();
        
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
    
    // GET ASSET TYPE ICON/EMOJI
    function getAssetTypeIcon(type) {
        switch(type) {
            case "image": return "🖼️";
            case "video": return "🎬";
            case "audio": return "🔊";
            case "document": return "📄";
            default: return "❓";
        }
    }
</script>

<div
    class={`card bg-base-200 shadow-lg hover:shadow-xl transition-all duration-200 ${sizeClasses[size] || ''} ${className}`}
    role={onClick ? "button" : ""}
    tabindex="-1"
    onclick={onClick ? viewAsset : null}
    onkeydown={onClick ? (e) => e.key === "Enter" && viewAsset() : null}
>
    <!-- THUMBNAIL/PREVIEW -->
    <figure class="relative aspect-square bg-base-300 overflow-hidden">
        {#if asset.thumbnailPath}
            <img
                src={`/api/thumbnails/${asset.thumbnailPath}`}
                alt={asset.title || "Asset"}
                class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-200"
            />
        {:else}
            <div class="w-full h-full flex items-center justify-center text-4xl bg-base-100">
                {getAssetTypeIcon(asset.type)}
            </div>
        {/if}
        
        <!-- TYPE BADGE -->
        <div class="absolute top-2 right-2 badge badge-sm">{asset.type}</div>
        
        <!-- OVERLAY WITH VIEW BUTTON -->
        <div class="absolute inset-0 bg-black opacity-0 hover:opacity-30 transition-opacity flex items-center justify-center">
            <span class="badge badge-primary">VIEW</span>
        </div>
    </figure>
    
    <!-- CONTENT -->
    <div class="card-body p-3">
        <div class="flex items-start justify-between">
            <h3 class="card-title text-sm truncate" title={asset.title || "Untitled"}>
                {asset.title || "Untitled"}
            </h3>
            
            {#if showActions}
                <!-- MENU BUTTON -->
                <div class="relative ml-2">
                    <button
                        class="btn btn-ghost btn-circle btn-xs"
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
                            class="dropdown-content menu bg-base-200 rounded-box p-2 shadow w-48 absolute right-0 z-50"
                            transition:fade={{ duration: 200 }}
                        >
                            <button
                                class="menu-item py-2 px-4 rounded-btn hover:bg-base-300 w-full text-left"
                                onclick={viewAsset}
                                aria-label="View asset details"
                            >
                                View Details
                            </button>
                            <button
                                class="menu-item py-2 px-4 rounded-btn hover:bg-base-300 w-full text-left"
                                onclick={downloadAsset}
                                aria-label="Download asset"
                            >
                                Download
                            </button>
                            <button
                                class="menu-item py-2 px-4 rounded-btn hover:bg-base-300 w-full text-left"
                                onclick={handleRegenerate}
                                disabled={loading}
                                aria-label="Regenerate thumbnail"
                            >
                                Regenerate Thumbnail
                            </button>
                            <button
                                class="menu-item py-2 px-4 rounded-btn hover:bg-base-300 w-full text-left text-error"
                                onclick={handleDelete}
                                disabled={loading}
                                aria-label="Delete asset"
                            >
                                Delete
                            </button>
                        </div>
                    {/if}
                </div>
            {/if}
        </div>
        
        {#if asset.description}
            <p class="text-xs text-gray-500 line-clamp-2" title={asset.description}>
                {asset.description}
            </p>
        {/if}
        
        <div class="flex justify-between items-center text-xs opacity-70 mt-2">
            <span title={asset.date ? formatDate(asset.date) : "Unknown date"}>
                {asset.date ? formatDate(asset.date, "MMM D, YYYY") : "Unknown date"}
            </span>
            <span>{formatFileSize(asset.size)}</span>
        </div>
    </div>
</div>

<!-- CLOSE MENU WHEN CLICKING OUTSIDE -->
{#if isMenuOpen}
    <div
        class="fixed inset-0 z-40"
        onclick={closeMenu}
        onkeydown={(e) => e.key === "Escape" && closeMenu()}
        transition:fade={{ duration: 200 }}
        role="button"
        aria-label="Close menu"
        tabindex="0"
    ></div>
{/if}
