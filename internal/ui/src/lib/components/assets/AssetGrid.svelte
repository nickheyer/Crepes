<script>
    import { fade } from "svelte/transition";
    import {
        state as assetState,
        filteredAssets,
        viewAsset,
    } from "$lib/stores/assetStore.svelte";
    import AssetCard from "./AssetCard.svelte";
    import { formatFileSize, formatDate } from "$lib/utils/formatters";
    
    // PROPS
    let { view = "grid" } = $props(); // 'grid' OR 'list'
    
    // SORTING AND FILTERING
    let sortBy = $state("date");
    let sortDir = $state("desc");
    
    function toggleSort(field) {
        if (sortBy === field) {
            sortDir = sortDir === "asc" ? "desc" : "asc";
        } else {
            sortBy = field;
            sortDir = "desc";
        }
    }
    
    // HANDLE VIEW ASSET
    function handleViewAsset(asset) {
        viewAsset(asset);
    }
</script>

<div>
    <!-- View toggle -->
    <div class="flex justify-between items-center mb-4">
        <div>
            <span class="text-dark-300 text-sm">
                {filteredAssets.length} assets found
            </span>
        </div>
        <div class="flex">
            <button
                class={`px-3 py-1.5 text-sm rounded-l-lg focus:outline-none ${view === "grid" ? "bg-primary-600 text-white" : "bg-base-700 text-dark-300 hover:bg-base-600"}`}
                onclick={() => (view = "grid")}
                aria-label="Grid view"
            >
                <svg
                    xmlns="http://www.w3.org/2000/svg"
                    class="h-5 w-5"
                    viewBox="0 0 20 20"
                    fill="currentColor"
                >
                    <path
                        d="M5 3a2 2 0 00-2 2v2a2 2 0 002 2h2a2 2 0 002-2V5a2 2 0 00-2-2H5zM5 11a2 2 0 00-2 2v2a2 2 0 002 2h2a2 2 0 002-2v-2a2 2 0 00-2-2H5zM11 5a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V5zM11 13a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z"
                    />
                </svg>
            </button>
            <button
                class={`px-3 py-1.5 text-sm rounded-r-lg focus:outline-none ${view === "list" ? "bg-primary-600 text-white" : "bg-base-700 text-dark-300 hover:bg-base-600"}`}
                onclick={() => (view = "list")}
                aria-label="List view"
            >
                <svg
                    xmlns="http://www.w3.org/2000/svg"
                    class="h-5 w-5"
                    viewBox="0 0 20 20"
                    fill="currentColor"
                >
                    <path
                        fill-rule="evenodd"
                        d="M3 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm0 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm0 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm0 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1z"
                        clip-rule="evenodd"
                    />
                </svg>
            </button>
        </div>
    </div>

    {#if !filteredAssets || filteredAssets.length === 0}
        <div class="bg-base-800 rounded-lg p-8 text-center">
            <svg
                xmlns="http://www.w3.org/2000/svg"
                class="h-16 w-16 mx-auto text-dark-500 mb-4"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
            >
                <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"
                />
            </svg>
            <h3 class="text-lg font-medium mb-2">No assets found</h3>
            <p class="text-dark-400">
                Try adjusting your filters or run a job to collect some assets.
            </p>
        </div>
    {:else if view === "grid"}
        <div
            class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-6"
        >
            {#each filteredAssets as asset (asset.id)}
                <div in:fade={{ duration: 150 }}>
                    <AssetCard {asset} on:view={() => handleViewAsset(asset)} />
                </div>
            {/each}
        </div>
    {:else}
        <div class="bg-base-800 rounded-lg overflow-hidden">
            <table class="min-w-full divide-y divide-dark-700">
                <thead>
                    <tr>
                        <th
                            scope="col"
                            class="px-4 py-3 text-left text-xs font-medium text-dark-300 uppercase tracking-wider"
                        >
                            <button
                                class="flex items-center focus:outline-none"
                                onclick={() => toggleSort("type")}
                            >
                                Type
                                {#if sortBy === "type"}
                                    <svg
                                        xmlns="http://www.w3.org/2000/svg"
                                        class="h-4 w-4 ml-1"
                                        viewBox="0 0 20 20"
                                        fill="currentColor"
                                    >
                                        {#if sortDir === "asc"}
                                            <path
                                                fill-rule="evenodd"
                                                d="M14.707 12.707a1 1 0 01-1.414 0L10 9.414l-3.293 3.293a1 1 0 01-1.414-1.414l4-4a1 1 0 011.414 0l4 4a1 1 0 010 1.414z"
                                                clip-rule="evenodd"
                                            />
                                        {:else}
                                            <path
                                                fill-rule="evenodd"
                                                d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z"
                                                clip-rule="evenodd"
                                            />
                                        {/if}
                                    </svg>
                                {/if}
                            </button>
                        </th>
                        <th
                            scope="col"
                            class="px-4 py-3 text-left text-xs font-medium text-dark-300 uppercase tracking-wider"
                        >
                            <button
                                class="flex items-center focus:outline-none"
                                onclick={() => toggleSort("title")}
                            >
                                Title
                                {#if sortBy === "title"}
                                    <svg
                                        xmlns="http://www.w3.org/2000/svg"
                                        class="h-4 w-4 ml-1"
                                        viewBox="0 0 20 20"
                                        fill="currentColor"
                                    >
                                        {#if sortDir === "asc"}
                                            <path
                                                fill-rule="evenodd"
                                                d="M14.707 12.707a1 1 0 01-1.414 0L10 9.414l-3.293 3.293a1 1 0 01-1.414-1.414l4-4a1 1 0 011.414 0l4 4a1 1 0 010 1.414z"
                                                clip-rule="evenodd"
                                            />
                                        {:else}
                                            <path
                                                fill-rule="evenodd"
                                                d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z"
                                                clip-rule="evenodd"
                                            />
                                        {/if}
                                    </svg>
                                {/if}
                            </button>
                        </th>
                        <th
                            scope="col"
                            class="px-4 py-3 text-left text-xs font-medium text-dark-300 uppercase tracking-wider"
                        >
                            <button
                                class="flex items-center focus:outline-none"
                                onclick={() => toggleSort("date")}
                            >
                                Date
                                {#if sortBy === "date"}
                                    <svg
                                        xmlns="http://www.w3.org/2000/svg"
                                        class="h-4 w-4 ml-1"
                                        viewBox="0 0 20 20"
                                        fill="currentColor"
                                    >
                                        {#if sortDir === "asc"}
                                            <path
                                                fill-rule="evenodd"
                                                d="M14.707 12.707a1 1 0 01-1.414 0L10 9.414l-3.293 3.293a1 1 0 01-1.414-1.414l4-4a1 1 0 011.414 0l4 4a1 1 0 010 1.414z"
                                                clip-rule="evenodd"
                                            />
                                        {:else}
                                            <path
                                                fill-rule="evenodd"
                                                d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z"
                                                clip-rule="evenodd"
                                            />
                                        {/if}
                                    </svg>
                                {/if}
                            </button>
                        </th>
                        <th
                            scope="col"
                            class="px-4 py-3 text-left text-xs font-medium text-dark-300 uppercase tracking-wider"
                        >
                            <button
                                class="flex items-center focus:outline-none"
                                onclick={() => toggleSort("size")}
                            >
                                Size
                                {#if sortBy === "size"}
                                    <svg
                                        xmlns="http://www.w3.org/2000/svg"
                                        class="h-4 w-4 ml-1"
                                        viewBox="0 0 20 20"
                                        fill="currentColor"
                                    >
                                        {#if sortDir === "asc"}
                                            <path
                                                fill-rule="evenodd"
                                                d="M14.707 12.707a1 1 0 01-1.414 0L10 9.414l-3.293 3.293a1 1 0 01-1.414-1.414l4-4a1 1 0 011.414 0l4 4a1 1 0 010 1.414z"
                                                clip-rule="evenodd"
                                            />
                                        {:else}
                                            <path
                                                fill-rule="evenodd"
                                                d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z"
                                                clip-rule="evenodd"
                                            />
                                        {/if}
                                    </svg>
                                {/if}
                            </button>
                        </th>
                        <th
                            scope="col"
                            class="px-4 py-3 text-right text-xs font-medium text-dark-300 uppercase tracking-wider"
                        >
                            Actions
                        </th>
                    </tr>
                </thead>
                <tbody class="divide-y divide-dark-700">
                    {#each filteredAssets as asset (asset.id)}
                        <tr
                            class="hover:bg-base-750"
                            in:fade={{ duration: 150 }}
                        >
                            <td class="px-4 py-3 whitespace-nowrap">
                                <div class="flex items-center">
                                    <div
                                        class="w-8 h-8 flex-shrink-0 mr-2 bg-base-700 rounded overflow-hidden"
                                    >
                                        {#if asset.thumbnailPath}
                                            <img
                                                src={`/api/thumbnails/${asset.thumbnailPath}`}
                                                alt=""
                                                class="w-full h-full object-cover"
                                            />
                                        {:else}
                                            <div
                                                class="w-full h-full flex items-center justify-center text-dark-400"
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
                                    </div>
                                    <div>
                                        <div class="text-sm font-medium">
                                            {asset.type}
                                        </div>
                                    </div>
                                </div>
                            </td>
                            <td class="px-4 py-3">
                                <div class="text-sm truncate max-w-xs">
                                    {asset.title || "Untitled"}
                                </div>
                                <div
                                    class="text-xs text-dark-400 truncate max-w-xs"
                                >
                                    {asset.description || "No description"}
                                </div>
                            </td>
                            <td class="px-4 py-3 whitespace-nowrap text-sm">
                                {asset.date
                                    ? formatDate(asset.date)
                                    : "Unknown"}
                            </td>
                            <td class="px-4 py-3 whitespace-nowrap text-sm">
                                {formatFileSize(asset.size)}
                            </td>
                            <td
                                class="px-4 py-3 whitespace-nowrap text-right text-sm font-medium"
                            >
                                <button
                                    onclick={() => handleViewAsset(asset)}
                                    class="text-primary-400 hover:text-primary-300 mr-3"
                                >
                                    View
                                </button>
                                <button
                                    class="text-danger-400 hover:text-danger-300"
                                >
                                    Delete
                                </button>
                            </td>
                        </tr>
                    {/each}
                </tbody>
            </table>
        </div>
    {/if}
</div>
