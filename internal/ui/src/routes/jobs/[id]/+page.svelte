<script>
    import { onMount } from "svelte";
    import { page } from "$app/stores";
    import Card from "$lib/components/common/Card.svelte";
    import Button from "$lib/components/common/Button.svelte";
    import {
        state as jobState,
        loadJobs,
        startJobById,
        stopJobById,
        removeJob,
        resetJobWizard,
        updateJobWizardStep
    } from "$lib/stores/jobStore.svelte";

    import { 
        state as assetState,
        loadAssets, 
        updateFilters,
        filteredAssets
    } from "$lib/stores/assetStore.svelte";
    import AssetGrid from "$lib/components/assets/AssetGrid.svelte";
    import AssetViewer from "$lib/components/assets/AssetViewer.svelte";
    import JobWizard from "$lib/components/jobs/JobWizard.svelte";
    import { fetchJobStatistics } from "$lib/utils/api";
    import { formatDate, formatJobStatus, formatProgress } from "$lib/utils/formatters";
    import { addToast } from "$lib/stores/uiStore.svelte";
    import {
        Play, 
        StopCircle, 
        RefreshCcw, 
        ChevronLeft,
        Trash,
        CopyPlus,
        Clock,
        Edit
    } from 'lucide-svelte';
    
    // JOB ID FROM ROUTE
    const jobId = $page.params.id;
    
    // LOCAL STATE
    let job = $state(null);
    let loading = $state(true);
    let assetsLoading = $state(true);
    let confirmDelete = $state(false);
    let statistics = $state({
        totalAssets: 0,
        assetTypes: {},
        progress: 0,
        duration: "0s"
    });
    let refreshInterval;
    
    // INITIALIZE EDIT MODAL STATE 
    jobState.editJobModal = false;
    
    onMount(async () => {
        await loadJobData();
        // START AUTO-REFRESH IF JOB IS RUNNING
        if (job && job.status === "running") {
            startRefreshInterval();
        }
        
        return () => {
            if (refreshInterval) clearInterval(refreshInterval);
        };
    });
    
    async function loadJobData() {
        try {
            // LOAD JOBS IF NOT ALREADY LOADED
            if (jobState.jobs.length === 0) {
                await loadJobs();
            }
            
            // FIND JOB BY ID
            job = jobState.jobs.find(j => j.id === jobId);
            if (!job) {
                addToast("Job not found", "error");
                return;
            }
            
            // LOAD JOB STATISTICS 
            try {
                const response = await fetchJobStatistics(jobId);
                if (response.success) {
                    statistics = response.data;
                }
            } catch (error) {
                console.error("ERROR LOADING JOB STATISTICS:", error);
            }
            
            // SET ASSET FILTER FOR THIS JOB
            updateFilters({ jobId });
            
            // LOAD ASSETS FOR THIS JOB
            await loadAssets({ jobId });
        } catch (error) {
            console.error("ERROR LOADING JOB DATA:", error);
            addToast(`Failed to load job data: ${error.message}`, "error");
        } finally {
            loading = false;
            assetsLoading = false;
        }
    }
    
    function startRefreshInterval() {
        // REFRESH JOB DATA EVERY 5 SECONDS IF JOB IS RUNNING
        refreshInterval = setInterval(async () => {
            if (job && job.status === "running") {
                await loadJobData();
            } else {
                clearInterval(refreshInterval);
            }
        }, 5000);
    }
    
    async function handleStartJob() {
        try {
            await startJobById(jobId);
            addToast("Job Started", "success");
            await loadJobData();
            startRefreshInterval();
        } catch (error) {
            addToast(`Failed to start job: ${error.message}`, "error");
        }
    }
    
    async function handleStopJob() {
        try {
            await stopJobById(jobId);
            addToast("Job Stopped", "success");
            await loadJobData();
            if (refreshInterval) clearInterval(refreshInterval);
        } catch (error) {
            addToast(`Failed to stop job: ${error.message}`, "error");
        }
    }
    
    async function handleDeleteJob() {
        try {
            await removeJob(jobId);
            addToast("Job Deleted", "success");
            window.location.href = "/jobs";
        } catch (error) {
            addToast(`Failed to delete job: ${error.message}`, "error");
            confirmDelete = false;
        }
    }
    
    function createTemplate() {
        // CLONE JOB TO TEMPLATE AND REDIRECT TO TEMPLATE CREATION
        if (job) {
            // Reset the wizard before setting new data
            resetJobWizard();
            updateJobWizardStep(1, job);
            window.location.href = "/templates?fromJob=" + jobId;
        }
    }
    
    // OPEN EDIT JOB MODAL
    function openEditJobModal() {
        if (job) {
            // Reset the wizard and populate it with the current job data
            resetJobWizard();
            updateJobWizardStep(1, job);
            jobState.editJobModal = true;
        }
    }
    
    // HANDLE EDIT JOB SUCCESS
    function handleEditSuccess() {
        jobState.editJobModal = false;
        addToast("JOB UPDATED SUCCESSFULLY", "success");
        loadJobData();
    }
</script>

<svelte:head>
    <title>{job ? job.name || "Job Details" : "Job Details"} | Crepes</title>
</svelte:head>

<section>
    {#if loading}
        <div class="py-20 flex justify-center">
            <div
                class="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-primary-500"
            ></div>
        </div>
    {:else if !job}
        <div class="text-center py-12">
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
                    d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                />
            </svg>
            <h3 class="text-lg font-medium mb-2">Job not found</h3>
            <p class="text-dark-400 mb-4">
                The job you're looking for doesn't exist
            </p>
            <Button
                variant="primary"
                onclick={() => (window.location.href = "/jobs")}
            >
                <ChevronLeft class="h-5 w-5 mr-1" />
                Back to Jobs
            </Button>
        </div>
    {:else}
        <!-- JOB HEADER -->
        <div class="mb-6">
            <div class="flex items-center mb-2">
                <a
                    href="/jobs"
                    class="text-primary-400 hover:text-primary-300 flex items-center mr-2"
                >
                    <ChevronLeft class="h-5 w-5" />
                </a>
                <h1 class="text-2xl font-bold">{job.name || "Unnamed Job"}</h1>
                <span
                    class={`ml-3 px-2 py-1 text-xs font-medium rounded-full 
                    ${
                        job.status === "running"
                            ? "bg-green-600 text-green-100"
                            : job.status === "completed"
                              ? "bg-blue-600 text-blue-100"
                              : job.status === "failed"
                                ? "bg-red-600 text-red-100"
                                : job.status === "stopped"
                                  ? "bg-yellow-600 text-yellow-100"
                                  : "bg-gray-600 text-gray-100"
                    }`}
                >
                    {job.status || "idle"}
                </span>
            </div>
            <p class="text-dark-300">{job.baseUrl}</p>
        </div>

        <!-- JOB ACTIONS -->
        <div class="flex flex-wrap gap-2 mb-6">
            {#if job.status === "running"}
                <Button variant="warning" onclick={handleStopJob}>
                    <StopCircle class="h-5 w-5 mr-1" />
                    Stop Job
                </Button>
            {:else}
                <Button variant="success" onclick={handleStartJob}>
                    <Play class="h-5 w-5 mr-1" />
                    Start Job
                </Button>
            {/if}

            <Button variant="primary" onclick={loadJobData}>
                <RefreshCcw class="h-5 w-5 mr-1" />
                Refresh
            </Button>

            <!-- ADD EDIT BUTTON -->
            <Button variant="primary" onclick={openEditJobModal}>
                <Edit class="h-5 w-5 mr-1" />
                Edit Job
            </Button>

            <Button variant="outline" onclick={createTemplate}>
                <CopyPlus class="h-5 w-5 mr-1" />
                Create Template
            </Button>

            <div class="flex items-center space-x-2"></div>
            {#if confirmDelete}
                <span class="text-sm text-danger-400">Confirm Delete?</span>
                <Button variant="danger" size="sm" onclick={handleDeleteJob}>
                    Yes
                </Button>
                <Button
                    variant="outline"
                    size="sm"
                    onclick={() => (confirmDelete = false)}
                >
                    No
                </Button>
            {:else}
                <Button
                    variant="outline"
                    onclick={() => (confirmDelete = true)}
                >
                    <Trash class="h-5 w-5 mr-1 text-danger-400" />
                    Delete Job
                </Button>
            {/if}
        </div>

        <!-- JOB STATISTICS -->
        <div class="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
            <Card>
                <div class="flex flex-col items-center p-3">
                    <h3 class="text-sm font-medium text-dark-300 mb-1">
                        Total Assets
                    </h3>
                    <p class="text-2xl font-semibold">
                        {statistics.totalAssets}
                    </p>
                </div>
            </Card>
            <Card>
                <div class="flex flex-col items-center p-3">
                    <h3 class="text-sm font-medium text-dark-300 mb-1">
                        Progress
                    </h3>
                    <p class="text-2xl font-semibold">
                        {formatProgress(statistics.progress || 0)}
                    </p>
                    {#if job.status === "running"}
                        <div class="w-full bg-base-700 h-2 rounded-full mt-2">
                            <div
                                class="bg-primary-600 h-2 rounded-full"
                                style={`width: ${statistics.progress || 0}%`}
                            ></div>
                        </div>
                    {/if}
                </div>
            </Card>
            <Card>
                <div class="flex flex-col items-center p-3">
                    <h3 class="text-sm font-medium text-dark-300 mb-1">
                        Duration
                    </h3>
                    <p class="text-2xl font-semibold">
                        {statistics.duration || "0s"}
                    </p>
                </div>
            </Card>
            <Card>
                <div class="flex flex-col items-center p-3">
                    <h3 class="text-sm font-medium text-dark-300 mb-1">
                        Schedule
                    </h3>
                    <div class="flex items-center">
                        <Clock class="h-5 w-5 mr-1 text-dark-300" />
                        <p class="text-sm">{job.schedule || "Not scheduled"}</p>
                    </div>
                    {#if job.nextRun}
                        <p class="text-xs text-dark-400 mt-1">
                            Next run: {formatDate(job.nextRun)}
                        </p>
                    {/if}
                </div>
            </Card>
        </div>

        <!-- JOB ASSETS -->
        <Card title="Assets" class="mb-6">
            {#if assetsLoading}
                <div class="py-20 flex justify-center">
                    <div
                        class="animate-spin rounded-full h-10 w-10 border-t-2 border-b-2 border-primary-500"
                    ></div>
                </div>
            {:else if filteredAssets && filteredAssets.length === 0}
                <div class="text-center py-12">
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
                    <p class="text-dark-400 mb-4">
                        Start the job to begin collecting assets
                    </p>
                    {#if job.status !== "running"}
                        <Button variant="success" onclick={handleStartJob}>
                            <Play class="h-5 w-5 mr-1" />
                            Start Job
                        </Button>
                    {/if}
                </div>
            {:else}
                <AssetGrid />
            {/if}
        </Card>

        <!-- JOB CONFIGURATION -->
        <Card title="Job Configuration" class="mb-6">
            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                    <h3 class="text-sm font-medium text-dark-300 mb-2">
                        Base URL
                    </h3>
                    <p class="text-sm break-all bg-base-900 p-2 rounded">
                        {job.baseUrl}
                    </p>
                </div>
                <div>
                    <h3 class="text-sm font-medium text-dark-300 mb-2">
                        Status
                    </h3>
                    <p class="text-sm">{formatJobStatus(job.status)}</p>
                </div>
                <div>
                    <h3 class="text-sm font-medium text-dark-300 mb-2">
                        Last Run
                    </h3>
                    <p class="text-sm">
                        {job.lastRun ? formatDate(job.lastRun) : "Never"}
                    </p>
                </div>
                <div>
                    <h3 class="text-sm font-medium text-dark-300 mb-2">
                        Schedule
                    </h3>
                    <p class="text-sm">{job.schedule || "Not scheduled"}</p>
                </div>
            </div>

            {#if job.selectors && job.selectors.length > 0}
                <div class="mt-4 pt-4 border-t border-dark-700">
                    <h3 class="text-sm font-medium text-dark-300 mb-2">
                        Selectors
                    </h3>
                    <div class="bg-base-900 overflow-hidden rounded-lg">
                        <table class="min-w-full divide-y divide-dark-700">
                            <thead class="bg-base-800">
                                <tr>
                                    <th
                                        scope="col"
                                        class="px-4 py-3 text-left text-xs font-medium text-dark-300 uppercase tracking-wider"
                                        >Name</th
                                    >
                                    <th
                                        scope="col"
                                        class="px-4 py-3 text-left text-xs font-medium text-dark-300 uppercase tracking-wider"
                                        >Purpose</th
                                    >
                                    <th
                                        scope="col"
                                        class="px-4 py-3 text-left text-xs font-medium text-dark-300 uppercase tracking-wider"
                                        >Type</th
                                    >
                                    <th
                                        scope="col"
                                        class="px-4 py-3 text-left text-xs font-medium text-dark-300 uppercase tracking-wider"
                                        >Value</th
                                    >
                                </tr>
                            </thead>
                            <tbody class="bg-base-900 divide-y divide-dark-700">
                                {#each job.selectors as selector}
                                    <tr>
                                        <td
                                            class="px-4 py-2 whitespace-nowrap text-sm"
                                            >{selector.name}</td
                                        >
                                        <td class="px-4 py-2 whitespace-nowrap">
                                            <span
                                                class={`px-2 py-0.5 inline-flex text-xs leading-5 font-medium rounded-full 
                                                    ${
                                                        selector.purpose ===
                                                        "assets"
                                                            ? "bg-blue-500 text-blue-100"
                                                            : selector.purpose ===
                                                                "links"
                                                              ? "bg-green-500 text-green-100"
                                                              : selector.purpose ===
                                                                  "pagination"
                                                                ? "bg-yellow-500 text-yellow-100"
                                                                : "bg-purple-500 text-purple-100"
                                                    }`}
                                            >
                                                {selector.purpose}
                                            </span>
                                        </td>
                                        <td
                                            class="px-4 py-2 whitespace-nowrap text-sm"
                                            >{selector.type}</td
                                        >
                                        <td class="px-4 py-2 text-sm font-mono"
                                            >{selector.value}</td
                                        >
                                    </tr>
                                {/each}
                            </tbody>
                        </table>
                    </div>
                </div>
            {/if}
        </Card>
    {/if}
</section>

<!-- ASSET VIEWER MODAL -->
{#if assetState.assetViewerOpen && assetState.selectedAsset}
    <AssetViewer />
{/if}

{#if jobState.editJobModal}
    <dialog class="modal modal-open">
        <div class="modal-box max-w-5xl">
            <h2 class="text-xl font-bold mb-4">Edit Job: {job?.name || 'Unnamed Job'}</h2>
            <JobWizard
                isEditing={true}
                initialData={job}
                onSuccess={handleEditSuccess}
                onCancel={() => jobState.editJobModal = false}
            />
        </div>
        <form method="dialog" class="modal-backdrop">
            <button onclick={() => jobState.editJobModal = false}>close</button>
        </form>
    </dialog>
{/if}
