<script>
    import { onMount } from "svelte";
    import AssetGrid from "$lib/components/assets/AssetGrid.svelte";
    import FilterPanel from "$lib/components/assets/FilterPanel.svelte";
    import AssetViewer from "$lib/components/assets/AssetViewer.svelte";
    import Loading from "$lib/components/common/Loading.svelte";
    import {
        state as assetState,
        loadAssets
    } from "$lib/stores/assetStore.svelte";

    let loading = $state(true);
    let view = $state("grid");

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
        <Loading size="lg" />
    {:else}
        <AssetGrid {view} />
    {/if}
</section>

<!-- Asset Viewer Modal -->
{#if assetState.assetViewerOpen && assetState.selectedAsset}
    <AssetViewer />
{/if}
