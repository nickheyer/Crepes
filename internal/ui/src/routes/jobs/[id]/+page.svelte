<script>
    import { page } from "$app/state";
    import { buttonClasses, cardClasses, formatDate, formatSize, getAssetIcon } from '$lib/components';

    // STATE
    let job = $state({});
    let notFound = $state(false);
    
    let loading = $state(false);
    let currentTab = $state("assets");
    let selectedAssets = $state({});
    let selectAll = $state(false);
    let mediaViewerOpen = $state(false);
    let mediaViewerAsset = $state(null);
    let pollingPaused = $state(false);
    const jobId = $derived(page.params.id);

    async function pollJob() {
        if (pollingPaused || !job) return;
        
        try {
            const response = await fetch(`/api/jobs/${job.id}`);
            if (!response.ok) return; // Silently fail for polling
            
            const data = await response.json();
            
            // Only update if status or asset count changed
            if (job.status !== data.status || 
                (job.assets?.length || 0) !== (data.assets?.length || 0)) {
                
                // Keep the current tab selected when updating
                const prevTab = currentTab;
                job = data;
                currentTab = prevTab;
                
                // Fetch assets separately to avoid server-side pagination issues
                await fetchAssets();
            }
        } catch (error) {
            console.error("Error polling job:", error);
            // Don't show errors for background polling
        }
    }

    async function fetchAssets() {
        if (!job) return;
        
        try {
            const response = await fetch(`/api/jobs/${job.id}/assets`);
            if (!response.ok) {
                throw new Error("Assets not found");
            }

            const assets = await response.json();
            if (!job.assets || job.assets.length !== assets.length) {
                job.assets = assets || [];
            }
        } catch (error) {
            console.error("Error fetching assets:", error);
            window.showToast?.("Failed to fetch assets", "error");
        }
    }

    async function executeAction(actionFn, successMsg) {
        pollingPaused = true;

        try {
            window.showToast?.(successMsg + "...", "info");
            await actionFn();

            // Direct fetch after action completes
            const response = await fetch(`/api/jobs/${jobId}`);
            if (response.ok) {
                job = await response.json();
                await fetchAssets();
            }

            window.showToast?.(successMsg, "success");
        } catch (error) {
            console.error(`Error: ${error.message}`);
            window.showToast?.(`Error: ${error.message}`, "error");
        } finally {
            pollingPaused = false;
        }
    }

    function toggleSelectAll() {
        selectedAssets = {};
        if (selectAll && job?.assets) {
            job.assets.forEach((asset) => {
                selectedAssets[asset.id] = true;
            });
        }
    }

    function toggleAssetSelection(assetId, isChecked) {
        if (isChecked) {
            selectedAssets[assetId] = true;
        } else {
            const newSelectedAssets = { ...selectedAssets };
            delete newSelectedAssets[assetId];
            selectedAssets = newSelectedAssets;
        }

        // UPDATE SELECT ALL CHECKBOX
        const assets = job?.assets || [];
        selectAll =
            assets.length > 0 &&
            Object.keys(selectedAssets).length === assets.length;
    }

    function openMediaViewer(asset) {
        if (!asset.localPath) {
            window.showToast?.("Media not available", "error");
            return;
        }
        mediaViewerAsset = asset;
        mediaViewerOpen = true;
    }

    function closeMediaViewer() {
        mediaViewerOpen = false;
        setTimeout(() => {
            mediaViewerAsset = null;
        }, 300);
    }

    function navigateMedia(direction) {
        if (!mediaViewerAsset || (job?.assets || []).length <= 1) return;

        const assets = job?.assets || [];
        const currentIndex = assets.findIndex(
            (a) => a.id === mediaViewerAsset.id,
        );
        if (currentIndex === -1) return;

        let nextIndex = currentIndex + direction;

        // WRAP AROUND
        if (nextIndex < 0) nextIndex = assets.length - 1;
        if (nextIndex >= assets.length) nextIndex = 0;

        mediaViewerAsset = assets[nextIndex];
    }

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

    const selectedCount = $derived(Object.keys(selectedAssets).length);

    async function startJob() {
        executeAction(async () => {
            const resp = await fetch(`/api/jobs/${jobId}/start`, {
                method: "POST",
            });
            if (!resp.ok) throw new Error("Failed to start job");
        }, "Job started");
    }

    async function stopJob() {
        executeAction(async () => {
            const resp = await fetch(`/api/jobs/${jobId}/stop`, {
                method: "POST",
            });
            if (!resp.ok) throw new Error("Failed to stop job");
        }, "Job stopped");
    }

    async function deleteJob() {
        if (
            !confirm(
                "Are you sure you want to delete this job? This action cannot be undone.",
            )
        ) {
            return;
        }

        try {
            pollingPaused = true;
            window.showToast?.("Deleting job...", "info");
            await fetch(`/api/jobs/${jobId}`, { method: "DELETE" });
            window.showToast?.("Job deleted", "success");
            setTimeout(() => {
                window.location.href = "/";
            }, 1000);
        } catch (error) {
            console.error("Error deleting job:", error);
            window.showToast?.("Error deleting job", "error");
            pollingPaused = false;
        }
    }

    async function deleteSelectedAssets() {
        const selectedIds = Object.keys(selectedAssets);
        if (selectedIds.length === 0) return;

        if (
            !confirm(
                `Are you sure you want to delete ${selectedIds.length} selected assets? This action cannot be undone.`,
            )
        ) {
            return;
        }

        executeAction(async () => {
            // Delete assets one by one
            const deletePromises = selectedIds.map((id) =>
                fetch(`/api/assets/${id}`, { method: "DELETE" }).then(
                    (response) => {
                        if (!response.ok)
                            throw new Error(`Failed to delete asset ${id}`);
                        return response.json();
                    },
                ),
            );

            await Promise.all(deletePromises);
            selectedAssets = {};
            selectAll = false;
        }, "Selected assets deleted");
    }

    async function regenerateThumbnail(assetId) {
        executeAction(async () => {
            const response = await fetch(
                `/api/assets/${assetId}/regenerate-thumbnail`,
                { method: "POST" },
            );

            if (!response.ok) {
                throw new Error("Failed to regenerate thumbnail");
            }
        }, "Thumbnail regenerated");
    }

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

    function getStatusColor(status) {
        const colors = {
            idle: "bg-gray-500",
            running: "bg-green-500",
            completed: "bg-blue-500",
            failed: "bg-red-500",
            stopped: "bg-yellow-500",
        };
        return colors[status] || "bg-gray-500";
    }

    let refreshInterval;

    $effect(async () => {
        refreshInterval = setInterval(() => {
            if (!pollingPaused && !mediaViewerOpen) {
                pollJob();
            }
        }, 5000);

        try {
            const jobResponse = await fetch(`/api/jobs/${jobId}`);
            if (!jobResponse.ok) {
                job = null;
                notFound = true;
                return;
            }

            job = await jobResponse.json();
            notFound = false;
            const assetsResponse = await fetch(`/api/jobs/${jobId}/assets`);
            if (assetsResponse.ok) {
                job.assets = await assetsResponse.json();
            } else {
                job.assets = [];
            }
            
        } catch (error) {
            console.error("Error loading job:", error);
            job = null;
            notFound = true;
            return;
        }
        
        return () => {
            if (refreshInterval) clearInterval(refreshInterval);
        };
    });
</script>

<svelte:head>
    <title>Crepes - Job Details</title>
</svelte:head>

<svelte:window on:keydown={handleKeydown} />

<header class="bg-gray-800 shadow">
    <div
        class="max-w-7xl mx-auto py-6 px-4 sm:px-6 lg:px-8 flex justify-between items-center"
    >
        <div class="flex items-center">
            <a
                href="/"
                class="text-indigo-400 hover:text-indigo-300 transition mr-4 flex items-center"
                aria-label="Back to jobs list"
            >
                <svg
                    xmlns="http://www.w3.org/2000/svg"
                    class="h-5 w-5 mr-1"
                    viewBox="0 0 20 20"
                    fill="currentColor"
                    aria-hidden="true"
                >
                    <path
                        fill-rule="evenodd"
                        d="M12.707 5.293a1 1 0 010 1.414L9.414 10l3.293 3.293a1 1 0 01-1.414 1.414l-4-4a1 1 0 010-1.414l4-4a1 1 0 011.414 0z"
                        clip-rule="evenodd"
                    />
                </svg>
                Back
            </a>
            <h1 class="text-3xl font-bold">Job Details</h1>
        </div>
        {#if job}
            <div class="flex space-x-3">
                {#if job.status !== "running"}
                    <button
                        class={buttonClasses.success + " flex items-center"}
                        onclick={startJob}
                        aria-label="Start job"
                    >
                        <svg
                            xmlns="http://www.w3.org/2000/svg"
                            class="h-5 w-5 mr-1"
                            viewBox="0 0 20 20"
                            fill="currentColor"
                            aria-hidden="true"
                        >
                            <path
                                fill-rule="evenodd"
                                d="M10 18a8 8 0 100-16 8 8 0 000 16zM9.555 7.168A1 1 0 008 8v4a1 1 0 001.555.832l3-2a1 1 0 000-1.664l-3-2z"
                                clip-rule="evenodd"
                            />
                        </svg>
                        Start
                    </button>
                {:else}
                    <button
                        class={buttonClasses.warning + " flex items-center"}
                        onclick={stopJob}
                        aria-label="Stop job"
                    >
                        <svg
                            xmlns="http://www.w3.org/2000/svg"
                            class="h-5 w-5 mr-1"
                            viewBox="0 0 20 20"
                            fill="currentColor"
                            aria-hidden="true"
                        >
                            <path
                                fill-rule="evenodd"
                                d="M10 18a8 8 0 100-16 8 8 0 000 16zM8 7a1 1 0 00-1 1v4a1 1 0 001 1h4a1 1 0 001-1V8a1 1 0 00-1-1H8z"
                                clip-rule="evenodd"
                            />
                        </svg>
                        Stop
                    </button>
                {/if}

                <a
                    href="/gallery"
                    class={buttonClasses.primary + " flex items-center"}
                    aria-label="View gallery"
                >
                    <svg
                        xmlns="http://www.w3.org/2000/svg"
                        class="h-5 w-5 mr-1"
                        viewBox="0 0 20 20"
                        fill="currentColor"
                        aria-hidden="true"
                    >
                        <path
                            fill-rule="evenodd"
                            d="M4 3a2 2 0 00-2 2v10a2 2 0 002 2h12a2 2 0 002-2V5a2 2 0 00-2-2H4zm12 12H4l4-8 3 6 2-4 3 6z"
                            clip-rule="evenodd"
                        />
                    </svg>
                    Gallery
                </a>

                <button
                    class={buttonClasses.danger + " flex items-center"}
                    onclick={deleteJob}
                    aria-label="Delete job"
                >
                    <svg
                        xmlns="http://www.w3.org/2000/svg"
                        class="h-5 w-5 mr-1"
                        viewBox="0 0 20 20"
                        fill="currentColor"
                        aria-hidden="true"
                    >
                        <path
                            fill-rule="evenodd"
                            d="M9 2a1 1 0 00-.894.553L7.382 4H4a1 1 0 000 2v10a2 2 0 002 2h8a2 2 0 002-2V6a1 1 0 100-2h-3.382l-.724-1.447A1 1 0 0011 2H9zM7 8a1 1 0 012 0v6a1 1 0 11-2 0V8zm5-1a1 1 0 00-1 1v6a1 1 0 102 0V8a1 1 0 00-1-1z"
                            clip-rule="evenodd"
                        />
                    </svg>
                    Delete
                </button>
            </div>
        {/if}
    </div>
</header>

<main class="max-w-7xl mx-auto py-6 px-4 sm:px-6 lg:px-8 flex-grow">
    {#if loading}
        <div class="flex justify-center my-12">
            <div
                class="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-indigo-500"
                aria-label="Loading"
                role="status"
            ></div>
        </div>
    {:else if notFound}
        <div class={cardClasses + " mb-6 text-center"}>
            <svg
                xmlns="http://www.w3.org/2000/svg"
                class="h-16 w-16 mx-auto text-gray-600 mb-4"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                aria-hidden="true"
            >
                <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M9.172 16.172a4 4 0 015.656 0M9 10h.01M15 10h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                />
            </svg>
            <p class="text-xl text-gray-400 mb-4">
                Job not found or has been deleted.
            </p>
            <a
                href="/"
                class={buttonClasses.primary}
            >
                Return to Home
            </a>
        </div>
    {:else if job}
        <div class={cardClasses + " mb-6"}>
            <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div>
                    <h2 class="text-xl font-semibold mb-4">Job Information</h2>
                    <div class="space-y-3">
                        <div
                            class="bg-gray-750 p-3 rounded-md flex justify-between"
                        >
                            <span class="text-gray-400">ID:</span>
                            <span class="font-mono text-sm">{job.id}</span>
                        </div>
                        <div class="bg-gray-750 p-3 rounded-md flex flex-col">
                            <span class="text-gray-400 mb-1">Base URL:</span>
                            <a
                                href={job.baseURL}
                                target="_blank"
                                rel="noopener noreferrer"
                                class="text-indigo-400 hover:text-indigo-300 transition break-all"
                                >{job.baseURL}</a
                            >
                        </div>
                        <div
                            class="bg-gray-750 p-3 rounded-md flex justify-between items-center"
                        >
                            <span class="text-gray-400">Status:</span>
                            <span
                                class={`px-2 py-1 inline-flex text-xs leading-5 font-semibold rounded-full ${getStatusColor(job.status)}`}
                            >
                                {job.status}
                            </span>
                        </div>
                        {#if job.lastError}
                            <div class="bg-gray-750 p-3 rounded-md flex flex-col">
                                <span class="text-gray-400 mb-1">Last Error:</span>
                                <span class="text-red-400">{job.lastError}</span>
                            </div>
                        {/if}
                        <div
                            class="bg-gray-750 p-3 rounded-md flex justify-between"
                        >
                            <span class="text-gray-400">Last Run:</span>
                            <span>{formatDate(job.lastRun)}</span>
                        </div>
                        <div
                            class="bg-gray-750 p-3 rounded-md flex justify-between"
                        >
                            <span class="text-gray-400">Next Run:</span>
                            <span>{formatDate(job.nextRun)}</span>
                        </div>
                        <div
                            class="bg-gray-750 p-3 rounded-md flex justify-between"
                        >
                            <span class="text-gray-400">Current Page:</span>
                            <span>{job.currentPage || 1}</span>
                        </div>
                    </div>
                </div>
                <div>
                    <h2 class="text-xl font-semibold mb-4">
                        Job Configuration
                    </h2>
                    <div class="space-y-3">
                        <div
                            class="bg-gray-750 p-3 rounded-md flex justify-between"
                        >
                            <span class="text-gray-400">Schedule:</span>
                            <span>{job.schedule || "None (manual)"}</span>
                        </div>
                        <div
                            class="bg-gray-750 p-3 rounded-md flex justify-between"
                        >
                            <span class="text-gray-400">Max Depth:</span>
                            <span>{job.rules?.maxDepth || "Unlimited"}</span>
                        </div>
                        <div
                            class="bg-gray-750 p-3 rounded-md flex justify-between"
                        >
                            <span class="text-gray-400">Max Assets:</span>
                            <span>{job.rules?.maxAssets || "Unlimited"}</span>
                        </div>
                        <div
                            class="bg-gray-750 p-3 rounded-md flex justify-between"
                        >
                            <span class="text-gray-400">Request Delay:</span>
                            <span>
                                {job.rules?.requestDelay || 0}ms
                                {job.rules?.randomizeDelay
                                    ? "(randomized)"
                                    : ""}
                            </span>
                        </div>
                        <div
                            class="bg-gray-750 p-3 rounded-md flex justify-between"
                        >
                            <span class="text-gray-400">Timeout:</span>
                            <span>
                                {job.rules?.timeout
                                    ? `${job.rules.timeout / 1000000000}s`
                                    : "Default"}
                            </span>
                        </div>
                        <div
                            class="bg-gray-750 p-3 rounded-md flex justify-between"
                        >
                            <span class="text-gray-400">Total Assets:</span>
                            <span class="font-semibold"
                                >{job.assets?.length || 0}</span
                            >
                        </div>
                    </div>
                </div>
            </div>
        </div>

        <!-- TABS -->
        <div class="border-b border-gray-700 mb-6">
            <nav class="-mb-px flex space-x-8">
                <button
                    class={`py-4 px-1 border-b-2 transition ${currentTab === "assets" ? "border-indigo-400 text-indigo-400" : "border-transparent text-gray-400 hover:text-gray-300 hover:border-gray-300"} font-medium text-sm`}
                    onclick={() => (currentTab = "assets")}
                    role="tab"
                    aria-selected={currentTab === "assets"}
                    aria-controls="assets-panel"
                    id="assets-tab"
                >
                    <div class="flex items-center">
                        <svg
                            xmlns="http://www.w3.org/2000/svg"
                            class="h-5 w-5 mr-1"
                            viewBox="0 0 20 20"
                            fill="currentColor"
                            aria-hidden="true"
                        >
                            <path
                                fill-rule="evenodd"
                                d="M4 3a2 2 0 00-2 2v10a2 2 0 002 2h12a2 2 0 002-2V5a2 2 0 00-2-2H4zm12 12H4l4-8 3 6 2-4 3 6z"
                                clip-rule="evenodd"
                            />
                        </svg>
                        Assets ({job.assets?.length || 0})
                    </div>
                </button>
                <button
                    class={`py-4 px-1 border-b-2 transition ${currentTab === "selectors" ? "border-indigo-400 text-indigo-400" : "border-transparent text-gray-400 hover:text-gray-300 hover:border-gray-300"} font-medium text-sm`}
                    onclick={() => (currentTab = "selectors")}
                    role="tab"
                    aria-selected={currentTab === "selectors"}
                    aria-controls="selectors-panel"
                    id="selectors-tab"
                >
                    <div class="flex items-center">
                        <svg
                            xmlns="http://www.w3.org/2000/svg"
                            class="h-5 w-5 mr-1"
                            viewBox="0 0 20 20"
                            fill="currentColor"
                            aria-hidden="true"
                        >
                            <path
                                fill-rule="evenodd"
                                d="M12.316 3.051a1 1 0 01.633 1.265l-4 12a1 1 0 11-1.898-.632l4-12a1 1 0 011.265-.633zM5.707 6.293a1 1 0 010 1.414L3.414 10l2.293 2.293a1 1 0 11-1.414 1.414l-3-3a1 1 0 010-1.414l3-3a1 1 0 011.414 0zm8.586 0a1 1 0 011.414 0l3 3a1 1 0 010 1.414l-3 3a1 1 0 11-1.414-1.414L16.586 10l-2.293-2.293a1 1 0 010-1.414z"
                                clip-rule="evenodd"
                            />
                        </svg>
                        Selectors
                    </div>
                </button>
            </nav>
        </div>

        <!-- ASSETS TAB -->
        <div 
            role="tabpanel"
            id="assets-panel"
            aria-labelledby="assets-tab"
            hidden={currentTab !== "assets"}
        >
            {#if currentTab === "assets"}
                <!-- BULK ACTIONS -->
                <div class="flex justify-between items-center mb-4">
                    <div class="flex items-center space-x-2">
                        <input
                            type="checkbox"
                            id="selectAll"
                            checked={selectAll}
                            onchange={toggleSelectAll}
                            class="rounded bg-gray-700 border-gray-600 text-indigo-600"
                            aria-label="Select all assets"
                        />
                        <label for="selectAll" class="text-sm">Select All</label>
                    </div>
                    {#if selectedCount > 0}
                        <div class="flex items-center space-x-4">
                            <span class="text-gray-400"
                                >{selectedCount} selected</span
                            >
                            <button
                                class={buttonClasses.danger + " text-sm flex items-center"}
                                onclick={deleteSelectedAssets}
                                aria-label="Delete selected assets"
                            >
                                <svg
                                    xmlns="http://www.w3.org/2000/svg"
                                    class="h-4 w-4 mr-1"
                                    viewBox="0 0 20 20"
                                    fill="currentColor"
                                    aria-hidden="true"
                                >
                                    <path
                                        fill-rule="evenodd"
                                        d="M9 2a1 1 0 00-.894.553L7.382 4H4a1 1 0 000 2v10a2 2 0 002 2h8a2 2 0 002-2V6a1 1 0 100-2h-3.382l-.724-1.447A1 1 0 0011 2H9zM7 8a1 1 0 012 0v6a1 1 0 11-2 0V8zm5-1a1 1 0 00-1 1v6a1 1 0 102 0V8a1 1 0 00-1-1z"
                                        clip-rule="evenodd"
                                    />
                                </svg>
                                Delete Selected
                            </button>
                        </div>
                    {/if}
                </div>

                <!-- ASSETS GRID -->
                {#if job.assets?.length > 0}
                    <div
                        class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-6"
                    >
                        {#each job.assets as asset (asset.id)}
                            <div
                                class="bg-gray-800 rounded-lg shadow overflow-hidden hover:shadow-lg transition transform hover:-translate-y-1"
                            >
                                <div
                                    class="h-48 bg-gray-700 overflow-hidden relative group"
                                >
                                    <div class="absolute top-2 left-2 z-10">
                                        <input
                                            type="checkbox"
                                            checked={selectedAssets[asset.id] ||
                                                false}
                                            onchange={(e) =>
                                                toggleAssetSelection(
                                                    asset.id,
                                                    e.target.checked,
                                                )}
                                            class="rounded bg-gray-700 border-gray-600 text-indigo-600"
                                            aria-label={`Select ${asset.title || 'asset'}`}
                                        />
                                    </div>
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
                                    <button
                                        class="absolute inset-0 bg-black bg-opacity-50 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center cursor-pointer"
                                        onclick={() => openMediaViewer(asset)}
                                        aria-label={`View ${asset.title || 'asset'}`}
                                    >
                                        <div
                                            class="bg-indigo-600 rounded-full p-2"
                                        >
                                            {#if asset.type === "image"}
                                                <svg
                                                    xmlns="http://www.w3.org/2000/svg"
                                                    class="h-6 w-6"
                                                    fill="none"
                                                    viewBox="0 0 24 24"
                                                    stroke="currentColor"
                                                    aria-hidden="true"
                                                >
                                                    <path
                                                        stroke-linecap="round"
                                                        stroke-linejoin="round"
                                                        stroke-width="2"
                                                        d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
                                                    />
                                                </svg>
                                            {:else if asset.type === "video"}
                                                <svg
                                                    xmlns="http://www.w3.org/2000/svg"
                                                    class="h-6 w-6"
                                                    fill="none"
                                                    viewBox="0 0 24 24"
                                                    stroke="currentColor"
                                                    aria-hidden="true"
                                                >
                                                    <path
                                                        stroke-linecap="round"
                                                        stroke-linejoin="round"
                                                        stroke-width="2"
                                                        d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z"
                                                    />
                                                    <path
                                                        stroke-linecap="round"
                                                        stroke-linejoin="round"
                                                        stroke-width="2"
                                                        d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                                                    />
                                                </svg>
                                            {:else if asset.type === "audio"}
                                                <svg
                                                    xmlns="http://www.w3.org/2000/svg"
                                                    class="h-6 w-6"
                                                    fill="none"
                                                    viewBox="0 0 24 24"
                                                    stroke="currentColor"
                                                    aria-hidden="true"
                                                >
                                                    <path
                                                        stroke-linecap="round"
                                                        stroke-linejoin="round"
                                                        stroke-width="2"
                                                        d="M15.536 8.464a5 5 0 010 7.072m2.828-9.9a9 9 0 010 12.728M5.586 15.536a5 5 0 001.414 1.414m2.828-9.9a9 9 0 012.828-2.828"
                                                    />
                                                </svg>
                                            {:else}
                                                <svg
                                                    xmlns="http://www.w3.org/2000/svg"
                                                    class="h-6 w-6"
                                                    fill="none"
                                                    viewBox="0 0 24 24"
                                                    stroke="currentColor"
                                                    aria-hidden="true"
                                                >
                                                    <path
                                                        stroke-linecap="round"
                                                        stroke-linejoin="round"
                                                        stroke-width="2"
                                                        d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
                                                    />
                                                </svg>
                                            {/if}
                                        </div>
                                    </button>
                                </div>
                                <div class="p-4">
                                    <h3 class="font-medium text-lg truncate">
                                        {asset.title || "Untitled"}
                                    </h3>
                                    <p class="text-gray-400 text-sm truncate">
                                        {asset.description || "No description"}
                                    </p>
                                    <div
                                        class="mt-3 flex justify-between text-sm"
                                    >
                                        <span class="text-gray-400"
                                            >{formatSize(asset.size)}</span
                                        >
                                        <div class="flex space-x-2">
                                            <button
                                                class="text-indigo-400 hover:text-indigo-300 transition"
                                                onclick={() =>
                                                    openMediaViewer(asset)}
                                                aria-label={`View ${asset.title || 'asset'}`}
                                            >
                                                View
                                            </button>
                                            <button
                                                class="text-yellow-400 hover:text-yellow-300 transition"
                                                onclick={() =>
                                                    regenerateThumbnail(
                                                        asset.id,
                                                    )}
                                                aria-label="Regenerate thumbnail"
                                            >
                                                Regen
                                            </button>
                                            <button
                                                class="text-red-400 hover:text-red-300 transition"
                                                onclick={() => {
                                                    if (
                                                        confirm(
                                                            "Are you sure you want to delete this asset?",
                                                        )
                                                    ) {
                                                        executeAction(
                                                            async () => {
                                                                const resp =
                                                                    await fetch(
                                                                        `/api/assets/${asset.id}`,
                                                                        {
                                                                            method: "DELETE",
                                                                        },
                                                                    );
                                                                if (!resp.ok)
                                                                    throw new Error(
                                                                        "Failed to delete asset",
                                                                    );
                                                            },
                                                            "Asset deleted",
                                                        );
                                                    }
                                                }}
                                                aria-label={`Delete ${asset.title || 'asset'}`}
                                            >
                                                Delete
                                            </button>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        {/each}
                    </div>
                {:else}
                    <div
                        class="bg-gray-800 rounded-lg shadow p-6 text-center text-gray-400"
                    >
                        <svg
                            xmlns="http://www.w3.org/2000/svg"
                            class="h-16 w-16 mx-auto text-gray-600 mb-4"
                            fill="none"
                            viewBox="0 0 24 24"
                            stroke="currentColor"
                            aria-hidden="true"
                        >
                            <path
                                stroke-linecap="round"
                                stroke-linejoin="round"
                                stroke-width="2"
                                d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"
                            />
                        </svg>
                        <p class="text-lg">No assets found yet.</p>
                        <p class="mt-2 text-gray-500">
                            Start the job to begin scraping content.
                        </p>
                        {#if job.status !== "running"}
                            <button
                                onclick={startJob}
                                class={buttonClasses.success + " mt-4 flex items-center mx-auto"}
                                aria-label="Start job"
                            >
                                <svg
                                    xmlns="http://www.w3.org/2000/svg"
                                    class="h-5 w-5 mr-1"
                                    viewBox="0 0 20 20"
                                    fill="currentColor"
                                    aria-hidden="true"
                                >
                                    <path
                                        fill-rule="evenodd"
                                        d="M10 18a8 8 0 100-16 8 8 0 000 16zM9.555 7.168A1 1 0 008 8v4a1 1 0 001.555.832l3-2a1 1 0 000-1.664l-3-2z"
                                        clip-rule="evenodd"
                                    />
                                </svg>
                                Start Job
                            </button>
                        {/if}
                    </div>
                {/if}
            {/if}
        </div>

        <!-- SELECTORS TAB -->
        <div 
            role="tabpanel"
            id="selectors-panel" 
            aria-labelledby="selectors-tab"
            hidden={currentTab !== "selectors"}
        >
            {#if currentTab === "selectors"}
                <div class={cardClasses}>
                    <h3 class="text-lg font-medium mb-4">Job Selectors</h3>

                    {#if job.selectors && job.selectors.length > 0}
                        <div class="grid gap-4">
                            {#each job.selectors as selector, index}
                                <div class="bg-gray-750 p-4 rounded-md">
                                    <div class="flex justify-between mb-2">
                                        <div>
                                            <span
                                                class="px-2 py-1 bg-indigo-600 rounded-md text-xs font-medium"
                                            >
                                                {selector.type}
                                            </span>
                                            <span
                                                class="ml-2 px-2 py-1 bg-gray-700 rounded-md text-xs font-medium"
                                            >
                                                {selector.for}
                                            </span>
                                        </div>
                                        <span class="text-xs text-gray-400"
                                            >Selector {index + 1}</span
                                        >
                                    </div>
                                    <div
                                        class="font-mono text-sm bg-gray-800 p-3 rounded mt-2 overflow-x-auto"
                                    >
                                        {selector.value}
                                    </div>
                                    <div class="mt-2 text-xs text-gray-400">
                                        {#if selector.for === "links"}
                                            Selects links to follow for crawling
                                        {:else if selector.for === "assets"}
                                            Selects elements to download as assets
                                        {:else if selector.for === "title"}
                                            Extracts the title metadata
                                        {:else if selector.for === "description"}
                                            Extracts the description metadata
                                        {:else if selector.for === "author"}
                                            Extracts the author metadata
                                        {:else if selector.for === "date"}
                                            Extracts the date metadata
                                        {:else if selector.for === "pagination"}
                                            Identifies pagination links for
                                            multi-page content
                                        {/if}
                                    </div>
                                </div>
                            {/each}
                        </div>
                    {:else}
                        <div class="text-center py-8 bg-gray-750 rounded-md">
                            <p class="text-gray-400">
                                No selectors defined for this job.
                            </p>
                        </div>
                    {/if}
                </div>
            {/if}
        </div>
    {/if}
</main>

<!-- MEDIA VIEWER MODAL -->
{#if mediaViewerOpen}
    <div
        class="fixed inset-0 bg-black z-50 flex flex-col"
        role="dialog"
        aria-modal="true"
        aria-labelledby="media-viewer-title"
    >
        <!-- HEADER -->
        <div
            class="p-4 flex justify-between items-center bg-gray-900 bg-opacity-80"
        >
            <div id="media-viewer-title" class="text-lg font-medium truncate flex-1">
                {mediaViewerAsset?.title || "Media Viewer"}
            </div>
            <div class="flex space-x-4">
                <button
                    onclick={() => downloadAsset(mediaViewerAsset)}
                    class="text-white hover:text-indigo-300 transition flex items-center space-x-1"
                    aria-label="Download asset"
                >
                    <svg
                        xmlns="http://www.w3.org/2000/svg"
                        class="h-5 w-5"
                        viewBox="0 0 20 20"
                        fill="currentColor"
                        aria-hidden="true"
                    >
                        <path
                            fill-rule="evenodd"
                            d="M3 17a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm3.293-7.707a1 1 0 011.414 0L9 10.586V3a1 1 0 112 0v7.586l1.293-1.293a1 1 0 111.414 1.414l-3 3a1 1 0 01-1.414 0l-3-3a1 1 0 010-1.414z"
                            clip-rule="evenodd"
                        />
                    </svg>
                    <span>Download</span>
                </button>
                <button
                    onclick={() => closeMediaViewer()}
                    class="text-white hover:text-red-300 transition"
                    aria-label="Close media viewer"
                >
                    <svg
                        xmlns="http://www.w3.org/2000/svg"
                        class="h-6 w-6"
                        fill="none"
                        viewBox="0 0 24 24"
                        stroke="currentColor"
                        aria-hidden="true"
                    >
                        <path
                            stroke-linecap="round"
                            stroke-linejoin="round"
                            stroke-width="2"
                            d="M6 18L18 6M6 6l12 12"
                        />
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
                aria-label="Previous media"
            >
                <svg
                    xmlns="http://www.w3.org/2000/svg"
                    class="h-8 w-8"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                    aria-hidden="true"
                >
                    <path
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        stroke-width="2"
                        d="M15 19l-7-7 7-7"
                    />
                </svg>
            </button>

            <button
                onclick={() => navigateMedia(1)}
                class="absolute right-4 bg-gray-800 bg-opacity-50 hover:bg-opacity-80 transition p-2 rounded-full text-white z-50"
                aria-label="Next media"
            >
                <svg
                    xmlns="http://www.w3.org/2000/svg"
                    class="h-8 w-8"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                    aria-hidden="true"
                >
                    <path
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        stroke-width="2"
                        d="M9 5l7 7-7 7"
                    />
                </svg>
            </button>

            <!-- MEDIA CONTENT -->
            {#if mediaViewerAsset?.type === "image" && mediaViewerAsset?.localPath}
                <img
                    src={`/assets/${mediaViewerAsset.localPath}`}
                    alt={mediaViewerAsset.title || "Image"}
                    class="max-h-full max-w-full object-contain"
                />
            {:else if mediaViewerAsset?.type === "video" && mediaViewerAsset?.localPath}
                <video
                    src={`/assets/${mediaViewerAsset.localPath}`}
                    controls
                    autoplay
                    class="max-h-full max-w-full"
                >
                    <track kind="captions">
                    Your browser does not support the video tag.
                </video>
            {:else if mediaViewerAsset?.type === "audio" && mediaViewerAsset?.localPath}
                <div class="bg-gray-800 p-6 rounded-lg w-full max-w-2xl">
                    <div class="mb-4 flex justify-center">
                        <div
                            class="w-32 h-32 bg-gray-700 rounded-full flex items-center justify-center text-4xl"
                            aria-hidden="true"
                        >
                            🔊
                        </div>
                    </div>
                    <h3 class="text-xl font-medium text-center mb-4">
                        {mediaViewerAsset.title || "Audio File"}
                    </h3>
                    <audio 
                        src={`/assets/${mediaViewerAsset.localPath}`} 
                        controls 
                        class="w-full" 
                        autoplay
                    >
                        <track kind="captions">
                        Your browser does not support the audio element.
                    </audio>
                </div>
            {:else}
                <div class="bg-gray-800 p-6 rounded-lg">
                    <div class="flex flex-col items-center">
                        <div class="text-6xl mb-4" aria-hidden="true">
                            {getAssetIcon(mediaViewerAsset?.type || 'unknown')}
                        </div>
                        <h3 class="text-xl font-medium mb-2">{mediaViewerAsset?.title || "File"}</h3>
                        <p class="text-gray-400 mb-4">{formatSize(mediaViewerAsset?.size || 0)}</p>
                        <button
                            onclick={() => downloadAsset(mediaViewerAsset)}
                            class={buttonClasses.primary + " flex items-center space-x-2"}
                            aria-label="Download file"
                        >
                            <svg
                                xmlns="http://www.w3.org/2000/svg"
                                class="h-5 w-5 mr-2"
                                viewBox="0 0 20 20"
                                fill="currentColor"
                                aria-hidden="true"
                            >
                                <path
                                    fill-rule="evenodd"
                                    d="M3 17a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm3.293-7.707a1 1 0 011.414 0L9 10.586V3a1 1 0 112 0v7.586l1.293-1.293a1 1 0 111.414 1.414l-3 3a1 1 0 01-1.414 0l-3-3a1 1 0 010-1.414z"
                                    clip-rule="evenodd"
                                />
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
                    {mediaViewerAsset && job?.assets 
                        ? job.assets.findIndex(a => a.id === mediaViewerAsset.id) + 1 
                        : 0} of {job?.assets?.length || 0}
                </div>
            </div>
        </div>
    </div>
{/if}

