<script>
    import { onMount } from "svelte";
    import AssetGrid from "$lib/components/assets/AssetGrid.svelte";
    import FilterPanel from "$lib/components/assets/FilterPanel.svelte";
    import AssetViewer from "$lib/components/assets/AssetViewer.svelte";
    import {
        loadAssets,
        assetViewerOpen,
        assetCounts,
        selectedAsset,
    } from "$lib/stores/assetStore";

    // State
    let loading = $state(true);
    let view = $state("grid"); // 'grid' or 'list'

    onMount(async () => {
        try {
            await loadAssets();
        } catch (error) {
            console.error("Error loading assets:", error);
        } finally {
            loading = false;
        }
    });
</script>

<svelte:head>
    <title>Asset Gallery | Crepes</title>
</svelte:head>

<section>
    <div class="mb-4">
        <h1 class="text-2xl font-bold mb-2">Asset Gallery</h1>
        <p class="text-dark-300">Browse and manage your scraped assets</p>
    </div>

    <div class="mb-6">
        <FilterPanel />
    </div>

    {#if loading}
        <div class="py-20 flex justify-center">
            <div
                class="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-primary-500"
            ></div>
        </div>
    {:else}
        <AssetGrid {view} />
    {/if}
</section>

<!-- Asset Viewer Modal -->
{#if $assetViewerOpen && $selectedAsset}
    <AssetViewer />
{/if}
