<script>
    // STATE
    let assets = $state([]);
    let jobs = $state([]);
    let loading = $state(true);
    let selectedAsset = $state(null);
    let currentView = $state("grid");
    let currentSort = $state({ field: "date", direction: "desc" });
    let mediaViewerOpen = $state(false);
    let mediaViewerAsset = $state(null);

    // FILTERS
    let filters = $state({
        type: "",
        jobId: "",
        search: "",
    });

    // FETCH JOBS
    async function fetchJobs() {
        try {
            const response = await fetch("/api/jobs");
            if (!response.ok) throw new Error("Failed to fetch jobs");
            jobs = (await response.json()) || [];
        } catch (error) {
            console.error("Error fetching jobs:", error);
            window.showToast?.("Error fetching jobs", "error");
        }
    }

    // FETCH ASSETS
    async function fetchAssets() {
        try {
            loading = true;

            // BUILD QUERY PARAMS
            const params = new URLSearchParams();
            if (filters.type) params.append("type", filters.type);
            if (filters.jobId) params.append("jobId", filters.jobId);
            if (filters.search) params.append("search", filters.search);

            const response = await fetch("/api/assets?" + params.toString());
            if (!response.ok) throw new Error("Failed to fetch assets");

            assets = (await response.json()) || [];
        } catch (error) {
            console.error("Error fetching assets:", error);
            window.showToast?.("Error fetching assets", "error");
        } finally {
            loading = false;
        }
    }

    // APPLY FILTERS
    function applyFilters() {
        fetchAssets();
    }

    // RESET FILTERS
    function resetFilters() {
        filters = {
            type: "",
            jobId: "",
            search: "",
        };
        fetchAssets();
    }

    // SET VIEW MODE
    function setView(view) {
        currentView = view;
    }

    // SET SORTING
    function setSorting(field) {
        if (currentSort.field === field) {
            currentSort = {
                ...currentSort,
                direction: currentSort.direction === "asc" ? "desc" : "asc",
            };
        } else {
            currentSort = { field, direction: "desc" };
        }
    }

    // SHOW ASSET DETAIL
    function showAssetDetail(asset) {
        selectedAsset = asset;
    }

    // OPEN MEDIA VIEWER
    function openMediaViewer(asset) {
        if (!asset.localPath) {
            window.showToast?.("Media not available", "error");
            return;
        }
        mediaViewerAsset = asset;
        mediaViewerOpen = true;
    }

    // CLOSE MEDIA VIEWER
    function closeMediaViewer() {
        mediaViewerOpen = false;
        setTimeout(() => {
            mediaViewerAsset = null;
        }, 300);
    }

    // CLOSE MODAL
    function closeAssetDetail() {
        selectedAsset = null;
    }

    // DELETE ASSET
    async function deleteAsset(assetId) {
        if (!confirm("Are you sure you want to delete this asset? This action cannot be undone.")) {
            return;
        }
        
        try {
            window.showToast?.("Deleting asset...", "info");

            const response = await fetch(`/api/assets/${assetId}`, {
                method: "DELETE",
            });

            if (!response.ok) {
                throw new Error("Failed to delete asset");
            }

            await response.json();

            // CLOSE MODALS IF OPEN
            if (selectedAsset && selectedAsset.id === assetId) {
                closeAssetDetail();
            }
            
            if (mediaViewerAsset && mediaViewerAsset.id === assetId) {
                closeMediaViewer();
            }

            // REFRESH ASSETS
            fetchAssets();
            window.showToast?.("Asset deleted", "success");
        } catch (error) {
            console.error("Error deleting asset:", error);
            window.showToast?.("Failed to delete asset", "error");
        }
    }

    // REGENERATE THUMBNAIL
    async function regenerateThumbnail(assetId) {
        try {
            window.showToast?.("Regenerating thumbnail...", "info");

            const response = await fetch(
                `/api/assets/${assetId}/regenerate-thumbnail`,
                { method: "POST" }
            );

            if (!response.ok) {
                throw new Error("Failed to regenerate thumbnail");
            }

            const data = await response.json();

            // UPDATE ASSET IN LIST
            assets = assets.map((item) => {
                if (item.id === assetId) {
                    return { ...item, thumbnailPath: data.thumbnailPath };
                }
                return item;
            });

            // UPDATE SELECTED ASSET IF OPEN
            if (selectedAsset && selectedAsset.id === assetId) {
                selectedAsset = {
                    ...selectedAsset,
                    thumbnailPath: data.thumbnailPath,
                };
            }

            // UPDATE MEDIA VIEWER ASSET IF OPEN
            if (mediaViewerAsset && mediaViewerAsset.id === assetId) {
                mediaViewerAsset = {
                    ...mediaViewerAsset,
                    thumbnailPath: data.thumbnailPath,
                };
            }

            window.showToast?.("Thumbnail regenerated", "success");
        } catch (error) {
            console.error("Error regenerating thumbnail:", error);
            window.showToast?.("Failed to regenerate thumbnail", "error");
        }
    }

    // UTILITY FUNCTIONS
    function getJobName(jobId) {
        const job = jobs.find((j) => j.id === jobId);
        return job ? job.baseUrl : "Unknown Job";
    }

    function formatDate(dateString) {
        if (!dateString) return "Unknown";
        return new Date(dateString).toLocaleString();
    }

    function formatSize(bytes) {
        if (!bytes) return "Unknown";
        const sizes = ["B", "KB", "MB", "GB"];
        const i = Math.floor(Math.log(bytes) / Math.log(1024));
        return (bytes / Math.pow(1024, i)).toFixed(2) + " " + sizes[i];
    }

    function getAssetIcon(type) {
        const icons = {
            video: "🎬",
            image: "🖼️",
            audio: "🔊",
            document: "📄",
            unknown: "❓",
        };
        return icons[type] || icons["unknown"];
    }

    // DOWNLOAD ASSET
    function downloadAsset(asset) {
        if (!asset.localPath) {
            window.showToast?.("Asset file not available", "error");
            return;
        }

        const link = document.createElement("a");
        link.href = `/assets/${asset.localPath}`;
        link.download = asset.title || "download";
        link.click();
    }

    // SORTED ASSETS
    const sortedAssets = $derived(sortAssets(assets, currentSort));

    function sortAssets(assets, sort) {
        return [...assets].sort((a, b) => {
            let aValue, bValue;

            if (sort.field === "date") {
                aValue = a.date ? new Date(a.date).getTime() : 0;
                bValue = b.date ? new Date(b.date).getTime() : 0;
            } else if (sort.field === "title") {
                aValue = (a.title || "").toLowerCase();
                bValue = (b.title || "").toLowerCase();
            } else if (sort.field === "type") {
                aValue = (a.type || "").toLowerCase();
                bValue = (b.type || "").toLowerCase();
            } else if (sort.field === "size") {
                aValue = a.size || 0;
                bValue = b.size || 0;
            } else {
                return 0;
            }

            if (aValue < bValue) return sort.direction === "asc" ? -1 : 1;
            if (aValue > bValue) return sort.direction === "asc" ? 1 : -1;
            return 0;
        });
    }

    // KEYBOARD NAVIGATION FOR MEDIA VIEWER
    function handleKeydown(event) {
        if (!mediaViewerOpen) return;
        
        if (event.key === "Escape") {
            closeMediaViewer();
        } else if (event.key === "ArrowRight") {
            navigateMedia(1);
        } else if (event.key === "ArrowLeft") {
            navigateMedia(-1);
        }
    }

    // NAVIGATE BETWEEN MEDIA IN VIEWER
    function navigateMedia(direction) {
        if (!mediaViewerAsset || sortedAssets.length <= 1) return;
        
        const currentIndex = sortedAssets.findIndex(a => a.id === mediaViewerAsset.id);
        if (currentIndex === -1) return;
        
        let nextIndex = currentIndex + direction;
        
        // WRAP AROUND
        if (nextIndex < 0) nextIndex = sortedAssets.length - 1;
        if (nextIndex >= sortedAssets.length) nextIndex = 0;
        
        mediaViewerAsset = sortedAssets[nextIndex];
    }

    // LIFECYCLE
    $effect(() => {
        // ON MOUNT
        fetchJobs().then(fetchAssets);
        
        // ADD KEYBOARD EVENT LISTENER
        if (typeof window !== 'undefined') {
            window.addEventListener('keydown', handleKeydown);
            return () => window.removeEventListener('keydown', handleKeydown);
        }
    });
</script>

<svelte:head>
    <title>Crepes - Asset Gallery</title>
</svelte:head>

<header class="bg-gray-800 shadow">
    <div class="max-w-7xl mx-auto py-6 px-4 sm:px-6 lg:px-8 flex justify-between items-center">
        <div class="flex items-center">
            <a href="/" class="text-indigo-400 hover:text-indigo-300 mr-4 flex items-center">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 mr-1" viewBox="0 0 20 20" fill="currentColor">
                    <path fill-rule="evenodd" d="M12.707 5.293a1 1 0 010 1.414L9.414 10l3.293 3.293a1 1 0 01-1.414 1.414l-4-4a1 1 0 010-1.414l4-4a1 1 0 011.414 0z" clip-rule="evenodd" />
                </svg>
                Back
            </a>
            <h1 class="text-3xl font-bold">Asset Gallery</h1>
        </div>
        <div class="flex space-x-3">
            <button
                class={`px-3 py-1 rounded-md transition ${currentView === "grid" ? "bg-indigo-600" : "bg-gray-700 hover:bg-gray-600"}`}
                onclick={() => setView("grid")}
                aria-label="setviewgrid"
            >
                <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                    <path d="M5 3a2 2 0 00-2 2v2a2 2 0 002 2h2a2 2 0 002-2V5a2 2 0 00-2-2H5zM5 11a2 2 0 00-2 2v2a2 2 0 002 2h2a2 2 0 002-2v-2a2 2 0 00-2-2H5zM11 5a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V5zM11 13a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z" />
                </svg>
            </button>
            <button
                class={`px-3 py-1 rounded-md transition ${currentView === "table" ? "bg-indigo-600" : "bg-gray-700 hover:bg-gray-600"}`}
                onclick={() => setView("table")}
                aria-label="setviewtable"
            >
                <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                    <path fill-rule="evenodd" d="M5 4a3 3 0 00-3 3v6a3 3 0 003 3h10a3 3 0 003-3V7a3 3 0 00-3-3H5zm-1 9v-1h5v2H5a1 1 0 01-1-1zm7 1h4a1 1 0 001-1v-1h-5v2zm0-4h5V8h-5v2zM9 8H4v2h5V8z" clip-rule="evenodd" />
                </svg>
            </button>
        </div>
    </div>
</header>

<main class="max-w-7xl mx-auto py-6 px-4 sm:px-6 lg:px-8">
    <!-- FILTERS -->
    <div class="bg-gray-800 shadow rounded-lg p-6 mb-6">
        <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
            <div>
                <label for="filter-type-select" class="block text-sm font-medium mb-1">Asset Type</label>
                <select
                    id="filter-type-select"
                    bind:value={filters.type}
                    class="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-md"
                >
                    <option value="">All Types</option>
                    <option value="image">Images</option>
                    <option value="video">Videos</option>
                    <option value="audio">Audio</option>
                    <option value="document">Documents</option>
                </select>
            </div>
            <div>
                <label for="filter-jobid-select" class="block text-sm font-medium mb-1">Job</label>
                <select
                    id="filter-jobid-select"
                    bind:value={filters.jobId}
                    class="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-md"
                >
                    <option value="">All Jobs</option>
                    {#each jobs as job}
                        <option value={job.id}>{job.baseUrl}</option>
                    {/each}
                </select>
            </div>
            <div>
                <label for="filter-search-input" class="block text-sm font-medium mb-1">Search</label>
                <input
                    id="filter-search-input"
                    type="text"
                    bind:value={filters.search}
                    placeholder="Search by title or description"
                    class="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-md"
                />
            </div>
            <div class="flex items-end">
                <div class="flex space-x-2">
                    <button
                        class="px-4 py-2 bg-indigo-600 rounded-md hover:bg-indigo-700 transition"
                        onclick={applyFilters}
                    >
                        Apply
                    </button>
                    <button
                        class="px-4 py-2 bg-gray-600 rounded-md hover:bg-gray-700 transition"
                        onclick={resetFilters}
                    >
                        Reset
                    </button>
                </div>
            </div>
        </div>
    </div>

    {#if loading}
        <div class="flex justify-center my-12">
            <div
                class="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-indigo-500"
            ></div>
        </div>
    {:else}
        <!-- ASSETS INFO -->
        {#if sortedAssets.length > 0}
            <div class="flex justify-between items-center mb-4">
                <h2 class="text-xl font-semibold">
                    Assets ({sortedAssets.length})
                </h2>
                <div class="flex items-center space-x-2">
                    <span class="text-sm">Sort by:</span>
                    <button
                        class={`px-2 py-1 text-sm rounded transition ${currentSort.field === "date" ? "bg-indigo-600" : "bg-gray-700 hover:bg-gray-600"}`}
                        onclick={() => setSorting("date")}
                    >
                        Date {currentSort.field === "date"
                            ? currentSort.direction === "asc"
                                ? "↑"
                                : "↓"
                            : ""}
                    </button>
                    <button
                        class={`px-2 py-1 text-sm rounded transition ${currentSort.field === "title" ? "bg-indigo-600" : "bg-gray-700 hover:bg-gray-600"}`}
                        onclick={() => setSorting("title")}
                    >
                        Title {currentSort.field === "title"
                            ? currentSort.direction === "asc"
                                ? "↑"
                                : "↓"
                            : ""}
                    </button>
                    <button
                        class={`px-2 py-1 text-sm rounded transition ${currentSort.field === "type" ? "bg-indigo-600" : "bg-gray-700 hover:bg-gray-600"}`}
                        onclick={() => setSorting("type")}
                    >
                        Type {currentSort.field === "type"
                            ? currentSort.direction === "asc"
                                ? "↑"
                                : "↓"
                            : ""}
                    </button>
                    <button
                        class={`px-2 py-1 text-sm rounded transition ${currentSort.field === "size" ? "bg-indigo-600" : "bg-gray-700 hover:bg-gray-600"}`}
                        onclick={() => setSorting("size")}
                    >
                        Size {currentSort.field === "size"
                            ? currentSort.direction === "asc"
                                ? "↑"
                                : "↓"
                            : ""}
                    </button>
                </div>
            </div>

            <!-- GRID VIEW -->
            {#if currentView === "grid"}
                <div
                    class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-6"
                >
                    {#each sortedAssets as asset (asset.id)}
                        <div
                            class="bg-gray-800 rounded-lg shadow overflow-hidden hover:shadow-lg transition transform hover:-translate-y-1"
                        >
                            <button
                                class="h-48 bg-gray-700 overflow-hidden relative cursor-pointer w-full group"
                                onclick={() => openMediaViewer(asset)}
                            >
                                <img
                                    src={asset.thumbnailPath
                                        ? `/thumbnails/${asset.thumbnailPath}`
                                        : "/static/icons/generic.jpg"}
                                    class="w-full h-full object-cover transition-transform group-hover:scale-105"
                                    alt={asset.title || "Asset"}
                                />
                                <span
                                    class="absolute top-2 right-2 px-2 py-1 rounded-full text-xs font-bold bg-gray-900 text-white"
                                >
                                    {getAssetIcon(asset.type)}
                                    {asset.type}
                                </span>
                                
                                <!-- OVERLAY WITH PREVIEW BUTTON -->
                                <div class="absolute inset-0 bg-black bg-opacity-50 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center">
                                    <div class="bg-indigo-600 rounded-full p-2">
                                        {#if asset.type === 'image'}
                                            <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                                            </svg>
                                        {:else if asset.type === 'video'}
                                            <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
                                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                                            </svg>
                                        {:else if asset.type === 'audio'}
                                            <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.536 8.464a5 5 0 010 7.072m2.828-9.9a9 9 0 010 12.728M5.586 15.536a5 5 0 001.414 1.414m2.828-9.9a9 9 0 012.828-2.828" />
                                            </svg>
                                        {:else}
                                            <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                                            </svg>
                                        {/if}
                                    </div>
                                </div>
                            </button>
                            <div class="p-4">
                                <h3 class="font-medium text-lg truncate">
                                    {asset.title || "Untitled"}
                                </h3>
                                <p class="text-gray-400 text-sm truncate">
                                    {asset.description || "No description"}
                                </p>
                                <div class="mt-2 text-sm text-gray-400">
                                    From: {getJobName(asset.jobId)}
                                </div>
                                <div class="mt-3 flex justify-between text-sm">
                                    <span class="text-gray-400">{formatSize(asset.size)}</span>
                                    <div class="flex space-x-2">
                                        <button
                                            class="text-indigo-400 hover:text-indigo-300 transition"
                                            onclick={() => showAssetDetail(asset)}
                                        >
                                            Details
                                        </button>
                                        <button
                                            class="text-yellow-400 hover:text-yellow-300 transition"
                                            onclick={() => regenerateThumbnail(asset.id)}
                                        >
                                            Regen
                                        </button>
                                        <button
                                            class="text-red-400 hover:text-red-300 transition"
                                            onclick={() => deleteAsset(asset.id)}
                                        >
                                            Delete
                                        </button>
                                    </div>
                                </div>
                            </div>
                        </div>
                    {/each}
                </div>

                <!-- TABLE VIEW -->
            {:else}
                <div class="bg-gray-800 rounded-lg overflow-hidden shadow">
                    <div class="overflow-x-auto">
                        <table class="min-w-full divide-y divide-gray-700">
                            <thead>
                                <tr>
                                    <th
                                        class="px-6 py-3 bg-gray-700 text-left text-xs font-medium uppercase tracking-wider cursor-pointer hover:bg-gray-600 transition"
                                        onclick={() => setSorting("type")}
                                    >
                                        Type {currentSort.field === "type"
                                            ? currentSort.direction === "asc"
                                                ? "↑"
                                                : "↓"
                                            : ""}
                                    </th>
                                    <th
                                        class="px-6 py-3 bg-gray-700 text-left text-xs font-medium uppercase tracking-wider cursor-pointer hover:bg-gray-600 transition"
                                        onclick={() => setSorting("title")}
                                    >
                                        Title {currentSort.field === "title"
                                            ? currentSort.direction === "asc"
                                                ? "↑"
                                                : "↓"
                                            : ""}
                                    </th>
                                    <th
                                        class="px-6 py-3 bg-gray-700 text-left text-xs font-medium uppercase tracking-wider cursor-pointer hover:bg-gray-600 transition"
                                        onclick={() => setSorting("date")}
                                    >
                                        Date {currentSort.field === "date"
                                            ? currentSort.direction === "asc"
                                                ? "↑"
                                                : "↓"
                                            : ""}
                                    </th>
                                    <th
                                    class="px-6 py-3 bg-gray-700 text-left text-xs font-medium uppercase tracking-wider cursor-pointer hover:bg-gray-600 transition"
                                    onclick={() => setSorting("size")}
                                >
                                    Size {currentSort.field === "size"
                                        ? currentSort.direction === "asc"
                                            ? "↑"
                                            : "↓"
                                        : ""}
                                </th>
                                <th
                                    class="px-6 py-3 bg-gray-700 text-left text-xs font-medium uppercase tracking-wider"
                                >
                                    Source
                                </th>
                                <th
                                    class="px-6 py-3 bg-gray-700 text-right text-xs font-medium uppercase tracking-wider"
                                >
                                    Actions
                                </th>
                            </tr>
                        </thead>
                        <tbody class="divide-y divide-gray-700">
                            {#each sortedAssets as asset (asset.id)}
                                <tr class="hover:bg-gray-700 transition">
                                    <td class="px-6 py-4 whitespace-nowrap">
                                        <div class="flex items-center">
                                            <span class="mr-2"
                                                >{getAssetIcon(
                                                    asset.type,
                                                )}</span
                                            >
                                            <span>{asset.type}</span>
                                        </div>
                                    </td>
                                    <td class="px-6 py-4">
                                        <div class="line-clamp-2">
                                            {asset.title || "Untitled"}
                                        </div>
                                    </td>
                                    <td class="px-6 py-4 whitespace-nowrap"
                                        >{asset.date
                                            ? formatDate(asset.date)
                                            : "Unknown"}</td
                                    >
                                    <td class="px-6 py-4 whitespace-nowrap"
                                        >{formatSize(asset.size)}</td
                                    >
                                    <td class="px-6 py-4 whitespace-nowrap"
                                        >{getJobName(asset.jobId)}</td
                                    >
                                    <td
                                        class="px-6 py-4 whitespace-nowrap text-right text-sm font-medium space-x-3"
                                    >
                                        <button
                                            class="text-indigo-400 hover:text-indigo-300 transition"
                                            onclick={() => openMediaViewer(asset)}
                                        >
                                            View
                                        </button>
                                        <button
                                            class="text-indigo-400 hover:text-indigo-300 transition"
                                            onclick={() => showAssetDetail(asset)}
                                        >
                                            Details
                                        </button>
                                        <button
                                            class="text-red-400 hover:text-red-300 transition"
                                            onclick={() => deleteAsset(asset.id)}
                                        >
                                            Delete
                                        </button>
                                    </td>
                                </tr>
                            {/each}
                        </tbody>
                    </table>
                </div>
            </div>
        {/if}
    {:else}
        <div
            class="bg-gray-800 rounded-lg shadow p-6 text-center text-gray-400"
        >
            <svg xmlns="http://www.w3.org/2000/svg" class="h-16 w-16 mx-auto text-gray-600 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
            </svg>
            <p class="text-lg">No assets found matching the current filters.</p>
            <p class="mt-2 text-gray-500">Try adjusting your filters or creating a new job.</p>
            <a href="/" class="mt-4 inline-block px-4 py-2 bg-indigo-600 rounded-md hover:bg-indigo-700 transition">
                Go to Jobs
            </a>
        </div>
    {/if}
{/if}
</main>

<!-- ASSET DETAILS MODAL -->
{#if selectedAsset}
<div
    class="fixed inset-0 bg-black bg-opacity-75 flex items-center justify-center z-50 p-4 overflow-y-auto"
>
    <!-- MODAL CONTENT -->
    <div
        class="bg-gray-800 rounded-lg max-w-4xl w-full max-h-screen overflow-y-auto mx-4"
    >
        <div
            class="px-6 py-4 border-b border-gray-700 flex justify-between items-center sticky top-0 bg-gray-800 z-10"
        >
            <h3 class="text-lg font-medium">Asset Details</h3>
            <button
                class="text-gray-400 hover:text-white transition text-2xl"
                onclick={closeAssetDetail}
            >
                &times;
            </button>
        </div>

        <div class="p-6">
            <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div>
                    <div
                        class="bg-gray-700 rounded-lg overflow-hidden mb-4 relative group"
                    >
                        <img
                            src={selectedAsset.thumbnailPath
                                ? `/thumbnails/${selectedAsset.thumbnailPath}`
                                : "/static/icons/generic.jpg"}
                            class="w-full h-64 object-contain"
                            alt={selectedAsset.title || "Asset"}
                        />
                        
                        <!-- VIEW BUTTON OVERLAY -->
                        <div class="absolute inset-0 bg-black bg-opacity-50 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center">
                            <button 
                                onclick={() => openMediaViewer(selectedAsset)}
                                class="bg-indigo-600 hover:bg-indigo-700 transition text-white px-4 py-2 rounded-full flex items-center space-x-2"
                            >
                                <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                                    <path d="M10 12a2 2 0 100-4 2 2 0 000 4z" />
                                    <path fill-rule="evenodd" d="M.458 10C1.732 5.943 5.522 3 10 3s8.268 2.943 9.542 7c-1.274 4.057-5.064 7-9.542 7S1.732 14.057.458 10zM14 10a4 4 0 11-8 0 4 4 0 018 0z" clip-rule="evenodd" />
                                </svg>
                                <span>View Full</span>
                            </button>
                        </div>
                    </div>
                    <div class="flex space-x-2 mb-4">
                        <button
                            class="px-4 py-2 bg-indigo-600 rounded-md hover:bg-indigo-700 transition flex-1 flex justify-center items-center space-x-1"
                            onclick={() => openMediaViewer(selectedAsset)}
                        >
                            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                                <path d="M10 12a2 2 0 100-4 2 2 0 000 4z" />
                                <path fill-rule="evenodd" d="M.458 10C1.732 5.943 5.522 3 10 3s8.268 2.943 9.542 7c-1.274 4.057-5.064 7-9.542 7S1.732 14.057.458 10zM14 10a4 4 0 11-8 0 4 4 0 018 0z" clip-rule="evenodd" />
                            </svg>
                            <span>View</span>
                        </button>
                        <button
                            class="px-4 py-2 bg-blue-600 rounded-md hover:bg-blue-700 transition flex-1 flex justify-center items-center space-x-1"
                            onclick={() => downloadAsset(selectedAsset)}
                        >
                            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                                <path fill-rule="evenodd" d="M3 17a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm3.293-7.707a1 1 0 011.414 0L9 10.586V3a1 1 0 112 0v7.586l1.293-1.293a1 1 0 111.414 1.414l-3 3a1 1 0 01-1.414 0l-3-3a1 1 0 010-1.414z" clip-rule="evenodd" />
                            </svg>
                            <span>Download</span>
                        </button>
                    </div>
                    <div class="flex space-x-2">
                        <button
                            class="px-4 py-2 bg-yellow-600 rounded-md hover:bg-yellow-700 transition flex-1 flex justify-center items-center space-x-1"
                            onclick={() => regenerateThumbnail(selectedAsset.id)}
                        >
                            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                                <path fill-rule="evenodd" d="M4 2a1 1 0 011 1v2.101a7.002 7.002 0 0111.601 2.566 1 1 0 11-1.885.666A5.002 5.002 0 005.999 7H9a1 1 0 010 2H4a1 1 0 01-1-1V3a1 1 0 011-1zm.008 9.057a1 1 0 011.276.61A5.002 5.002 0 0014.001 13H11a1 1 0 110-2h5a1 1 0 011 1v5a1 1 0 11-2 0v-2.101a7.002 7.002 0 01-11.601-2.566 1 1 0 01.61-1.276z" clip-rule="evenodd" />
                            </svg>
                            <span>Regenerate</span>
                        </button>
                        <button
                            class="px-4 py-2 bg-red-600 rounded-md hover:bg-red-700 transition flex-1 flex justify-center items-center space-x-1"
                            onclick={() => {
                                deleteAsset(selectedAsset.id);
                            }}
                        >
                            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                                <path fill-rule="evenodd" d="M9 2a1 1 0 00-.894.553L7.382 4H4a1 1 0 000 2v10a2 2 0 002 2h8a2 2 0 002-2V6a1 1 0 100-2h-3.382l-.724-1.447A1 1 0 0011 2H9zM7 8a1 1 0 012 0v6a1 1 0 11-2 0V8zm5-1a1 1 0 00-1 1v6a1 1 0 102 0V8a1 1 0 00-1-1z" clip-rule="evenodd" />
                            </svg>
                            <span>Delete</span>
                        </button>
                    </div>
                </div>

                <div class="space-y-4">
                    <div>
                        <h4 class="text-sm font-medium text-gray-400">
                            Title
                        </h4>
                        <p class="mt-1">
                            {selectedAsset.title || "Untitled"}
                        </p>
                    </div>
                    <div>
                        <h4 class="text-sm font-medium text-gray-400">
                            Description
                        </h4>
                        <p class="mt-1">
                            {selectedAsset.description || "No description"}
                        </p>
                    </div>
                    <div>
                        <h4 class="text-sm font-medium text-gray-400">
                            Type
                        </h4>
                        <p class="mt-1">{selectedAsset.type}</p>
                    </div>
                    <div>
                        <h4 class="text-sm font-medium text-gray-400">
                            Size
                        </h4>
                        <p class="mt-1">{formatSize(selectedAsset.size)}</p>
                    </div>
                    <div>
                        <h4 class="text-sm font-medium text-gray-400">
                            Source URL
                        </h4>
                        <p class="mt-1 break-all">
                            <a
                                href={selectedAsset.url}
                                target="_blank"
                                class="text-indigo-400 hover:text-indigo-300 transition"
                            >
                                {selectedAsset.url}
                            </a>
                        </p>
                    </div>
                    <div>
                        <h4 class="text-sm font-medium text-gray-400">
                            From Job
                        </h4>
                        <p class="mt-1">
                            <a 
                              href={`/jobs/${selectedAsset.jobId}`}
                              class="text-indigo-400 hover:text-indigo-300 transition"
                            >
                              {getJobName(selectedAsset.jobId)}
                            </a>
                        </p>
                    </div>
                    <div>
                        <h4 class="text-sm font-medium text-gray-400">
                            Author
                        </h4>
                        <p class="mt-1">
                            {selectedAsset.author || "Unknown"}
                        </p>
                    </div>
                    <div>
                        <h4 class="text-sm font-medium text-gray-400">
                            Date
                        </h4>
                        <p class="mt-1">{formatDate(selectedAsset.date)}</p>
                    </div>
                </div>
            </div>
        </div>
    </div>
</div>
{/if}

<!-- MEDIA VIEWER MODAL -->
{#if mediaViewerOpen}
<div 
    class="fixed inset-0 bg-black z-50 flex flex-col"
>
    <!-- HEADER -->
    <div class="p-4 flex justify-between items-center bg-gray-900 bg-opacity-80">
        <div class="text-lg font-medium truncate flex-1">
            {mediaViewerAsset?.title || "Media Viewer"}
        </div>
        <div class="flex space-x-4">
            <button 
                onclick={() => downloadAsset(mediaViewerAsset)}
                class="text-white hover:text-indigo-300 transition flex items-center space-x-1"
            >
                <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                    <path fill-rule="evenodd" d="M3 17a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm3.293-7.707a1 1 0 011.414 0L9 10.586V3a1 1 0 112 0v7.586l1.293-1.293a1 1 0 111.414 1.414l-3 3a1 1 0 01-1.414 0l-3-3a1 1 0 010-1.414z" clip-rule="evenodd" />
                </svg>
                <span>Download</span>
            </button>
            <button 
                onclick={() => closeMediaViewer()}
                class="text-white hover:text-red-300 transition"
                aria-label="closemediaview"
            >
                <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                </svg>
            </button>
        </div>
    </div>
    
    <!-- MAIN CONTENT -->
    <div class="flex-1 flex items-center justify-center p-4 relative">
        <!-- NAVIGATION ARROWS -->
        <button 
            onclick={() => navigateMedia(-1)}
            class="absolute left-4 bg-gray-800 bg-opacity-50 hover:bg-opacity-80 transition p-2 rounded-full text-white z-50"
            aria-label="navmediaviewback"
        >
            <svg xmlns="http://www.w3.org/2000/svg" class="h-8 w-8" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
            </svg>
        </button>
        
        <button 
            onclick={() => navigateMedia(1)}
            class="absolute right-4 bg-gray-800 bg-opacity-50 hover:bg-opacity-80 transition p-2 rounded-full text-white z-50"
            aria-label="navmediaviewplus"
        >
            <svg xmlns="http://www.w3.org/2000/svg" class="h-8 w-8" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
            </svg>
        </button>
        
        <!-- MEDIA CONTENT -->
        {#if mediaViewerAsset?.type === 'image' && mediaViewerAsset?.localPath}
            <img 
                src={`/assets/${mediaViewerAsset.localPath}`} 
                alt={mediaViewerAsset.title || "Image"} 
                class="max-h-full max-w-full object-contain"
            />
        {:else if mediaViewerAsset?.type === 'video' && mediaViewerAsset?.localPath}
            <video 
                src={`/assets/${mediaViewerAsset.localPath}`} 
                controls 
                autoplay 
                class="max-h-full max-w-full"
            >
                <track kind="captions">
                Your browser does not support the video tag.
            </video>
        {:else if mediaViewerAsset?.type === 'audio' && mediaViewerAsset?.localPath}
            <div class="bg-gray-800 p-6 rounded-lg w-full max-w-2xl">
                <div class="mb-4 flex justify-center">
                    <div class="w-32 h-32 bg-gray-700 rounded-full flex items-center justify-center text-4xl">
                        🔊
                    </div>
                </div>
                <h3 class="text-xl font-medium text-center mb-4">{mediaViewerAsset.title || "Audio File"}</h3>
                <audio 
                    src={`/assets/${mediaViewerAsset.localPath}`} 
                    controls 
                    class="w-full" 
                    autoplay
                >
                    Your browser does not support the audio element.
                </audio>
            </div>
        {:else}
            <div class="bg-gray-800 p-6 rounded-lg">
                <div class="flex flex-col items-center">
                    <div class="text-6xl mb-4">
                        {getAssetIcon(mediaViewerAsset?.type || 'unknown')}
                    </div>
                    <h3 class="text-xl font-medium mb-2">{mediaViewerAsset?.title || "File"}</h3>
                    <p class="text-gray-400 mb-4">{formatSize(mediaViewerAsset?.size || 0)}</p>
                    <button
                        onclick={() => downloadAsset(mediaViewerAsset)}
                        class="px-4 py-2 bg-indigo-600 rounded-md hover:bg-indigo-700 transition flex items-center space-x-2"
                    >
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                            <path fill-rule="evenodd" d="M3 17a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm3.293-7.707a1 1 0 011.414 0L9 10.586V3a1 1 0 112 0v7.586l1.293-1.293a1 1 0 111.414 1.414l-3 3a1 1 0 01-1.414 0l-3-3a1 1 0 010-1.414z" clip-rule="evenodd" />
                        </svg>
                        <span>Download File</span>
                    </button>
                </div>
            </div>
        {/if}
    </div>
    
    <!-- FOOTER WITH METADATA -->
    <div class="p-4 bg-gray-900 bg-opacity-80">
        <div class="flex justify-between items-center">
            <div>
                <p class="text-sm text-gray-400">
                    {mediaViewerAsset?.type || "Unknown"} • {formatSize(mediaViewerAsset?.size || 0)}
                </p>
            </div>
            <div class="text-sm text-gray-400">
                {mediaViewerAsset ? sortedAssets.findIndex(a => a.id === mediaViewerAsset.id) + 1 : 0} of {sortedAssets.length}
            </div>
        </div>
    </div>
</div>
{/if}
