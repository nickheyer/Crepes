<script>
    import { onMount } from "svelte";
    import { page } from "$app/stores";
    import Card from "$lib/components/common/Card.svelte";
    import Button from "$lib/components/common/Button.svelte";
    import { 
        jobs, 
        loadJobs, 
        startJobById, 
        stopJobById,
        removeJob,
        jobWizardState
    } from "$lib/stores/jobStore";
    import { 
        loadAssets, 
        assetViewerOpen,
        selectedAsset,
        assetFilters,
        updateFilters
    } from "$lib/stores/assetStore";
    import AssetGrid from "$lib/components/assets/AssetGrid.svelte";
    import AssetViewer from "$lib/components/assets/AssetViewer.svelte";
    import { formatDate, formatJobStatus, formatProgress } from "$lib/utils/formatters";
    import { addToast } from "$lib/stores/uiStore";
    import {
        Play, 
        StopCircle, 
        RefreshCcw, 
        ChevronLeft,
        Trash,
        CopyPlus,
        Clock,
    } from 'lucide-svelte';

    // Job ID from route
    const jobId = $page.params.id;
    
    // Local state
    let job = $state(null);
    let loading = $state(true);
    let assets = $state([]);
    let assetsLoading = $state(true);
    let confirmDelete = $state(false);
    let statistics = $state({
        totalAssets: 0,
        assetTypes: {},
        progress: 0,
        duration: "0s"
    });
    let refreshInterval;

    onMount(async () => {
        await loadJobData();
        
        // Start auto-refresh if job is running
        if (job && job.status === "running") {
            startRefreshInterval();
        }

        return () => {
            if (refreshInterval) clearInterval(refreshInterval);
        };
    });

    async function loadJobData() {
        try {
            // Load jobs if not already loaded
            if ($jobs.length === 0) {
                await loadJobs();
            }
            
            // Find job by ID
            job = $jobs.find(j => j.id === jobId);
            
            if (!job) {
                addToast("Job not found", "error");
                return;
            }
            
            // Load job statistics 
            try {
                const response = await fetch(`/api/jobs/${jobId}/statistics`);
                if (response.ok) {
                    const data = await response.json();
                    if (data.success) {
                        statistics = data.data;
                    }
                }
            } catch (error) {
                console.error("Error loading job statistics:", error);
            }
            
            // Set asset filter for this job
            updateFilters({ jobId });
            
            // Load assets for this job
            await loadAssets({ jobId });
            
        } catch (error) {
            console.error("Error loading job data:", error);
            addToast(`Failed to load job data: ${error.message}`, "error");
        } finally {
            loading = false;
            assetsLoading = false;
        }
    }

    function startRefreshInterval() {
        // Refresh job data every 5 seconds if job is running
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
            addToast("Job started successfully", "success");
            await loadJobData();
            startRefreshInterval();
        } catch (error) {
            addToast(`Failed to start job: ${error.message}`, "error");
        }
    }

    async function handleStopJob() {
        try {
            await stopJobById(jobId);
            addToast("Job stopped successfully", "success");
            await loadJobData();
            if (refreshInterval) clearInterval(refreshInterval);
        } catch (error) {
            addToast(`Failed to stop job: ${error.message}`, "error");
        }
    }

    async function handleDeleteJob() {
        try {
            await removeJob(jobId);
            addToast("Job deleted successfully", "success");
            window.location.href = "/jobs";
        } catch (error) {
            addToast(`Failed to delete job: ${error.message}`, "error");
            confirmDelete = false;
        }
    }

    function createTemplate() {
        // Clone job to template and redirect to template creation
        if (job) {
            jobWizardState.set({
                step: 1,
                data: { ...job },
                isTemplate: true
            });
            window.location.href = "/templates?fromJob=" + jobId;
        }
    }
</script>

<svelte:head>
    <title>{job ? (job.name || 'Job Details') : 'Job Details'} | Crepes</title>
</svelte:head>

<section>
    {#if loading}
        <div class="py-20 flex justify-center">
            <div class="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-primary-500"></div>
        </div>
    {:else if !job}
        <div class="text-center py-12">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-16 w-16 mx-auto text-dark-500 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <h3 class="text-lg font-medium mb-2">Job not found</h3>
            <p class="text-dark-400 mb-4">The job you're looking for doesn't exist</p>
            <Button variant="primary" onclick={() => window.location.href = '/jobs'}>
                <ChevronLeft class="h-5 w-5 mr-1" />
                Back to Jobs
            </Button>
        </div>
    {:else}
        <!-- Job header -->
        <div class="mb-6">
            <div class="flex items-center mb-2">
                <a href="/jobs" class="text-primary-400 hover:text-primary-300 flex items-center mr-2">
                    <ChevronLeft class="h-5 w-5" />
                </a>
                <h1 class="text-2xl font-bold">{job.name || 'Unnamed Job'}</h1>
                <span class={`ml-3 px-2 py-1 text-xs font-medium rounded-full 
                    ${job.status === "running" ? "bg-green-600 text-green-100" : 
                      job.status === "completed" ? "bg-blue-600 text-blue-100" : 
                      job.status === "failed" ? "bg-red-600 text-red-100" : 
                      job.status === "stopped" ? "bg-yellow-600 text-yellow-100" : 
                      "bg-gray-600 text-gray-100"}`}
                >
                    {job.status || "idle"}
                </span>
            </div>
            <p class="text-dark-300">{job.baseUrl}</p>
        </div>

        <!-- Job actions -->
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
            <Button variant="outline" onclick={createTemplate}>
                <CopyPlus class="h-5 w-5 mr-1" />
                Create Template
            </Button>
            {#if confirmDelete}
                <div class="flex items-center space-x-2">
                    <span class="text-sm text-danger-400">Confirm Delete?</span>
                    <Button variant="danger" size="sm" onclick={handleDeleteJob}>
                        Yes
                    </Button>
                    <Button variant="outline" size="sm" onclick={() => confirmDelete = false}>
                        No
                    </Button>
                </div>
            {:else}
                <Button variant="outline" onclick={() => confirmDelete = true}>
                    <Trash class="h-5 w-5 mr-1 text-danger-400" />
                    Delete Job
                </Button>
            {/if}
        </div>

        <!-- Job statistics -->
        <div class="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
            <Card>
                <div class="flex flex-col items-center p-3">
                    <h3 class="text-sm font-medium text-dark-300 mb-1">Total Assets</h3>
                    <p class="text-2xl font-semibold">{statistics.totalAssets}</p>
                </div>
            </Card>
            <Card>
                <div class="flex flex-col items-center p-3">
                    <h3 class="text-sm font-medium text-dark-300 mb-1">Progress</h3>
                    <p class="text-2xl font-semibold">{formatProgress(statistics.progress || 0)}</p>
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
                    <h3 class="text-sm font-medium text-dark-300 mb-1">Duration</h3>
                    <p class="text-2xl font-semibold">{statistics.duration || "0s"}</p>
                </div>
            </Card>
            <Card>
                <div class="flex flex-col items-center p-3">
                    <h3 class="text-sm font-medium text-dark-300 mb-1">Schedule</h3>
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

        <!-- Job assets -->
        <Card title="Assets" class="mb-6">
            {#if assetsLoading}
                <div class="py-20 flex justify-center">
                    <div class="animate-spin rounded-full h-10 w-10 border-t-2 border-b-2 border-primary-500"></div>
                </div>
            {:else}
                <AssetGrid />
            {/if}
        </Card>

        <!-- Job configuration -->
        <Card title="Job Configuration" class="mb-6">
            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                    <h3 class="text-sm font-medium text-dark-300 mb-2">Base URL</h3>
                    <p class="text-sm break-all bg-base-900 p-2 rounded">{job.baseUrl}</p>
                </div>
                <div>
                    <h3 class="text-sm font-medium text-dark-300 mb-2">Status</h3>
                    <p class="text-sm">{formatJobStatus(job.status)}</p>
                </div>
                <div>
                    <h3 class="text-sm font-medium text-dark-300 mb-2">Last Run</h3>
                    <p class="text-sm">{job.lastRun ? formatDate(job.lastRun) : "Never"}</p>
                </div>
                <div>
                    <h3 class="text-sm font-medium text-dark-300 mb-2">Schedule</h3>
                    <p class="text-sm">{job.schedule || "Not scheduled"}</p>
                </div>
            </div>

            {#if job.selectors && job.selectors.length > 0}
                <div class="mt-4 pt-4 border-t border-dark-700">
                    <h3 class="text-sm font-medium text-dark-300 mb-2">Selectors</h3>
                    <div class="bg-base-900 overflow-hidden rounded-lg">
                        <table class="min-w-full divide-y divide-dark-700">
                            <thead class="bg-base-800">
                                <tr>
                                    <th scope="col" class="px-4 py-3 text-left text-xs font-medium text-dark-300 uppercase tracking-wider">Name</th>
                                    <th scope="col" class="px-4 py-3 text-left text-xs font-medium text-dark-300 uppercase tracking-wider">Purpose</th>
                                    <th scope="col" class="px-4 py-3 text-left text-xs font-medium text-dark-300 uppercase tracking-wider">Type</th>
                                    <th scope="col" class="px-4 py-3 text-left text-xs font-medium text-dark-300 uppercase tracking-wider">Value</th>
                                </tr>
                            </thead>
                            <tbody class="bg-base-900 divide-y divide-dark-700">
                                {#each job.selectors as selector}
                                    <tr>
                                        <td class="px-4 py-2 whitespace-nowrap text-sm">{selector.name}</td>
                                        <td class="px-4 py-2 whitespace-nowrap">
                                            <span class={`px-2 py-0.5 inline-flex text-xs leading-5 font-medium rounded-full 
                                                ${selector.purpose === "assets" ? "bg-blue-500 text-blue-100" : 
                                                  selector.purpose === "links" ? "bg-green-500 text-green-100" : 
                                                  selector.purpose === "pagination" ? "bg-yellow-500 text-yellow-100" : 
                                                  "bg-purple-500 text-purple-100"}`}
                                            >
                                                {selector.purpose}
                                            </span>
                                        </td>
                                        <td class="px-4 py-2 whitespace-nowrap text-sm">{selector.type}</td>
                                        <td class="px-4 py-2 text-sm font-mono">{selector.value}</td>
                                    </tr>
                                {/each}
                            </tbody>
                        </table>
                    </div>
                </div>
            {/if}

            {#if job.rules}
                <div class="mt-4 pt-4 border-t border-dark-700">
                    <h3 class="text-sm font-medium text-dark-300 mb-2">Rules</h3>
                    <div class="grid grid-cols-1 md:grid-cols-2 gap-4 bg-base-900 p-4 rounded-lg">
                        <div>
                            <p class="text-xs text-dark-400">Max Depth</p>
                            <p class="text-sm">{job.rules.maxDepth || "Unlimited"}</p>
                        </div>
                        <div>
                            <p class="text-xs text-dark-400">Max Pages</p>
                            <p class="text-sm">{job.rules.maxPages || "Unlimited"}</p>
                        </div>
                        <div>
                            <p class="text-xs text-dark-400">Max Assets</p>
                            <p class="text-sm">{job.rules.maxAssets || "Unlimited"}</p>
                        </div>
                        <div>
                            <p class="text-xs text-dark-400">Concurrent Connections</p>
                            <p class="text-sm">{job.rules.maxConcurrent || 5}</p>
                        </div>
                        {#if job.rules.includeUrlPattern}
                            <div>
                                <p class="text-xs text-dark-400">Include URL Pattern</p>
                                <p class="text-sm font-mono">{job.rules.includeUrlPattern}</p>
                            </div>
                        {/if}
                        {#if job.rules.excludeUrlPattern}
                            <div>
                                <p class="text-xs text-dark-400">Exclude URL Pattern</p>
                                <p class="text-sm font-mono">{job.rules.excludeUrlPattern}</p>
                            </div>
                        {/if}
                    </div>
                </div>
            {/if}
        </Card>
    {/if}
</section>

<!-- Asset Viewer Modal -->
{#if $assetViewerOpen && $selectedAsset}
    <AssetViewer />
{/if}
