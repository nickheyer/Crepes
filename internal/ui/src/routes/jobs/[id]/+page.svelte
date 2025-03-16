<script>
    import { onMount } from "svelte";
    import { page } from "$app/stores";
    import Card from "$lib/components/common/Card.svelte";
    import Button from "$lib/components/common/Button.svelte";
    import Tabs from "$lib/components/common/Tabs.svelte";
    import JobBuilder from "$lib/components/jobs/JobBuilder.svelte";
    import {
        state as jobState,
        loadJobs,
        startJobById,
        stopJobById,
        removeJob
    } from "$lib/stores/jobStore.svelte.js";
    import Loading from "$lib/components/common/Loading.svelte";
    import { 
        state as assetState,
        loadAssets, 
        updateFilters,
        filteredAssets
    } from "$lib/stores/assetStore.svelte.js";
    import AssetGrid from "$lib/components/assets/AssetGrid.svelte";
    import AssetViewer from "$lib/components/assets/AssetViewer.svelte";
    //import { fetchJobStatistics, updateJob } from "$lib/utils/api.js";
    import { jobsApi } from '$lib/utils/api.js';
    import { formatDate, formatJobStatus, formatProgress } from "$lib/utils/formatters";
    import { addToast } from "$lib/stores/uiStore.svelte.js";
    import {
        Play, 
        StopCircle, 
        RefreshCcw, 
        ChevronLeft,
        Trash,
        CopyPlus,
        Clock,
        Edit,
        Workflow,
        Settings,
        Save,
        Blocks,
        Code
    } from 'lucide-svelte';
    
    // JOB ID FROM ROUTE
    const jobId = $page.params.id;
    
    // LOCAL STATE
    let job = $state(null);
    let loading = $state(true);
    let assetsLoading = $state(true);
    let savingPipeline = $state(false);
    let confirmDelete = $state(false);
    let pipelineEditorOpen = $state(false);
    let editBasicInfoOpen = $state(false);
    let editingJob = $state(null);
    let statistics = $state({
        totalAssets: 0,
        assetTypes: {},
        progress: {
            completedTasks: 0,
            totalTasks: 0
        },
        duration: "0s"
    });
    let refreshInterval;
    let activeTab = $state('overview');
    
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
                const response = await jobsApi.getStatistics(jobId);
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
    
    // OPEN PIPELINE EDITOR
    function openPipelineEditor() {
        pipelineEditorOpen = true;
    }
    
    // OPEN BASIC INFO EDITOR
    function openBasicInfoEditor() {
        editingJob = {...job};
        editBasicInfoOpen = true;
    }
    
    // SAVE PIPELINE CHANGES
    async function handleSavePipeline(event) {
        const { pipeline, jobConfig } = event.detail;
        
        try {
            savingPipeline = true;
            
            // PREPARE UPDATED JOB DATA
            const updatedJob = {
                ...job,
                data: {
                    ...job.data,
                    pipeline: JSON.stringify(pipeline),
                    jobConfig: JSON.stringify(jobConfig)
                }
            };
            
            // SAVE CHANGES
            await jobsApi.update(jobId, updatedJob);
            addToast("Pipeline saved successfully", "success");
            
            // RELOAD JOB DATA
            await loadJobData();
            
            // CLOSE EDITOR
            pipelineEditorOpen = false;
        } catch (error) {
            console.error("ERROR SAVING PIPELINE:", error);
            addToast(`Failed to save pipeline: ${error.message}`, "error");
        } finally {
            savingPipeline = false;
        }
    }
    
    // SAVE BASIC INFO CHANGES
    async function handleSaveBasicInfo() {
        if (!editingJob.name || !editingJob.baseUrl) {
            addToast("Name and Base URL are required", "error");
            return;
        }
        
        try {
            // PRESERVE DATA PROPERTIES
            const updatedJob = {
                ...editingJob,
                data: job.data 
            };
            
            // SAVE CHANGES
            await jobsApi.update(jobId, updatedJob);
            addToast("Job information updated successfully", "success");
            
            // RELOAD JOB DATA
            await loadJobData();
            
            // CLOSE EDITOR
            editBasicInfoOpen = false;
        } catch (error) {
            console.error("ERROR UPDATING JOB:", error);
            addToast(`Failed to update job: ${error.message}`, "error");
        }
    }
    
    // HELPER FUNCTION TO PARSE PIPELINE DATA
    function getPipelineStats() {
        if (!job?.data?.pipeline) return { stages: 0, tasks: 0 };
        
        try {
            const pipelineData = JSON.parse(job.data.pipeline);
            return {
                stages: pipelineData.length,
                tasks: pipelineData.reduce((total, stage) => total + stage.tasks.length, 0)
            };
        } catch {
            return { stages: 0, tasks: 0 };
        }
    }
    
    // CALCULATE PROGRESS PERCENTAGE
    function getProgressPercentage() {
        if (!statistics?.progress) return 0;
        const { completedTasks, totalTasks } = statistics.progress;
        if (totalTasks === 0) return 0;
        return Math.round((completedTasks / totalTasks) * 100);
    }
</script>

<svelte:head>
    <title>{job ? job.name || "Job Details" : "Job Details"} | Crepes</title>
</svelte:head>

<section>
    {#if loading}
        <Loading size="lg" />
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

            <Button variant="primary" onclick={openBasicInfoEditor}>
                <Edit class="h-5 w-5 mr-1" />
                Edit Details
            </Button>
            
            <Button variant="primary" onclick={openPipelineEditor}>
                <Workflow class="h-5 w-5 mr-1" />
                Edit Pipeline
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

        <!-- TABS NAVIGATION -->
        <Tabs
            tabs={[
                { id: 'overview', label: 'Overview', icon: Settings },
                { id: 'assets', label: 'Assets', icon: CopyPlus },
                { id: 'pipeline', label: 'Pipeline', icon: Workflow }
            ]}
            bind:activeTab={activeTab}
        />

        <!-- TAB CONTENT -->
        <div class="mt-6">
            <!-- OVERVIEW TAB -->
            {#if activeTab === 'overview'}
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
                                {getProgressPercentage()}%
                            </p>
                            {#if job.status === "running"}
                                <div class="w-full bg-base-700 h-2 rounded-full mt-2">
                                    <div
                                        class="bg-primary-600 h-2 rounded-full"
                                        style={`width: ${getProgressPercentage()}%`}
                                    ></div>
                                </div>
                                <div class="text-xs text-dark-400 mt-1">
                                    {statistics.progress.completedTasks} / {statistics.progress.totalTasks} tasks
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
                        <div>
                            <h3 class="text-sm font-medium text-dark-300 mb-2">
                                Pipeline
                            </h3>
                            <p class="text-sm">
                                {#if job.data?.pipeline}
                                    {@const stats = getPipelineStats()}
                                    {stats.stages} stages with {stats.tasks} tasks
                                {:else}
                                    No pipeline defined
                                {/if}
                            </p>
                        </div>
                    </div>
                </Card>
                
                <!-- ASSET TYPE DISTRIBUTION -->
                {#if statistics.assetTypes && Object.keys(statistics.assetTypes).length > 0}
                    <Card title="Asset Types" class="mb-6">
                        <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
                            {#each Object.entries(statistics.assetTypes) as [type, count]}
                                <div class="bg-base-900 p-3 rounded-lg">
                                    <p class="text-xs text-dark-400">{type}</p>
                                    <p class="text-lg font-medium">{count}</p>
                                </div>
                            {/each}
                        </div>
                    </Card>
                {/if}
            {/if}

            <!-- ASSETS TAB -->
            {#if activeTab === 'assets'}
                <Card title="Assets">
                    {#if assetsLoading}
                        <Loading size="lg" />
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
            {/if}

            <!-- PIPELINE TAB -->
            {#if activeTab === 'pipeline'}
                <Card>
                    <div class="flex justify-between items-center mb-4">
                        <h2 class="text-lg font-medium">Pipeline Definition</h2>
                        <Button variant="primary" onclick={openPipelineEditor}>
                            <Edit class="h-5 w-5 mr-1" />
                            Edit Pipeline
                        </Button>
                    </div>
                    
                    {#if !job.data?.pipeline}
                        <div class="text-center py-12">
                            <Blocks class="h-16 w-16 mx-auto text-dark-500 mb-4" />
                            <h3 class="text-lg font-medium mb-2">No pipeline defined</h3>
                            <p class="text-dark-400 mb-4">
                                Create a pipeline to define how this job will scrape data
                            </p>
                            <Button variant="primary" onclick={openPipelineEditor}>
                                <Workflow class="h-5 w-5 mr-1" />
                                Create Pipeline
                            </Button>
                        </div>
                    {:else}
                        <div class="mb-4">
                            {#if job.data.pipeline}
                                {@const pipelineData = JSON.parse(job.data.pipeline)}
                                <div class="bg-base-900 p-4 rounded-lg mb-4">
                                    <h3 class="text-sm font-medium mb-2">Pipeline Overview</h3>
                                    <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
                                        <div>
                                            <p class="text-xs text-dark-400">Stages</p>
                                            <p class="text-lg font-medium">{pipelineData.length}</p>
                                        </div>
                                        <div>
                                            <p class="text-xs text-dark-400">Tasks</p>
                                            <p class="text-lg font-medium">
                                                {pipelineData.reduce((total, stage) => total + stage.tasks.length, 0)}
                                            </p>
                                        </div>
                                    </div>
                                </div>
                                
                                <div class="space-y-4">
                                    {#each pipelineData as stage, stageIndex}
                                        <div class="bg-base-800 border border-base-700 rounded-lg overflow-hidden">
                                            <div class="bg-base-700 p-3">
                                                <div class="flex items-center">
                                                    <span class="bg-base-900 text-xs px-2 py-1 rounded-full mr-2">
                                                        {stageIndex + 1}
                                                    </span>
                                                    <h3 class="font-medium">{stage.name}</h3>
                                                    
                                                    <span class="ml-3 px-2 py-0.5 text-xs rounded-full bg-base-900 flex items-center">
                                                        {#if stage.parallelism.mode === 'sequential'}
                                                            Sequential
                                                        {:else if stage.parallelism.mode === 'parallel'}
                                                            Parallel ({stage.parallelism.maxWorkers})
                                                        {:else}
                                                            Worker per item ({stage.parallelism.maxWorkers})
                                                        {/if}
                                                    </span>
                                                </div>
                                                {#if stage.description}
                                                    <p class="text-xs text-dark-400 mt-1">{stage.description}</p>
                                                {/if}
                                            </div>
                                            
                                            <div class="p-3">
                                                <p class="text-xs text-dark-400 mb-2">Tasks ({stage.tasks.length})</p>
                                                {#if stage.tasks.length === 0}
                                                    <p class="text-xs text-dark-500 italic">No tasks defined in this stage</p>
                                                {:else}
                                                    <div class="space-y-1">
                                                        {#each stage.tasks as task, taskIndex}
                                                            <div class="bg-base-900 p-2 rounded text-sm flex items-center">
                                                                <span class="text-xs bg-base-800 rounded-full px-2 py-0.5 mr-2">
                                                                    {taskIndex + 1}
                                                                </span>
                                                                <span>{task.name}</span>
                                                                <span class="text-xs text-dark-400 ml-2">({task.type})</span>
                                                            </div>
                                                        {/each}
                                                    </div>
                                                {/if}
                                            </div>
                                        </div>
                                    {/each}
                                </div>
                                
                                <!-- VIEW PIPELINE JSON BUTTON -->
                                <div class="mt-4 flex justify-end">
                                    <Button variant="outline" size="sm" onclick={() => {
                                        // THIS WOULD OPEN A MODAL WITH THE JSON
                                        // FOR SIMPLICITY, WE'LL USE CONSOLE.LOG FOR NOW
                                        console.log(job.data.pipeline);
                                        addToast('Pipeline JSON logged to console', 'info');
                                    }}>
                                        <Code class="h-4 w-4 mr-1" />
                                        View Pipeline JSON
                                    </Button>
                                </div>
                            {:else}
                                <div class="bg-base-900 p-4 rounded-lg text-center">
                                    <p class="text-danger-400">Error parsing pipeline: {error.message}</p>
                                    <Button variant="primary" class="mt-2" onclick={openPipelineEditor}>
                                        Edit Pipeline
                                    </Button>
                                </div>
                            {/if}
                        </div>
                    {/if}
                </Card>
            {/if}
        </div>
    {/if}
</section>

<!-- ASSET VIEWER MODAL -->
{#if assetState.assetViewerOpen && assetState.selectedAsset}
    <AssetViewer />
{/if}

<!-- PIPELINE EDITOR MODAL -->
{#if pipelineEditorOpen && job}
    <div class="modal modal-open">
        <div class="modal-box max-w-7xl w-full h-[90vh] overflow-y-auto">
            <div class="bg-base-800 py-2 z-10 flex justify-between items-center">
                <h3 class="font-bold text-lg">Pipeline Builder - {job.name}</h3>
                <button 
                    onclick={() => pipelineEditorOpen = false} 
                    class="bg-base-700 hover:bg-base-600 p-2 rounded-full"
                    aria-label="pipelineeditormodal"
                >
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                    </svg>
                </button>
            </div>
            
            <div class="mt-4 mb-6">
                <JobBuilder 
                    on:save={handleSavePipeline}
                    initialPipeline={job.data?.pipeline}
                    initialConfig={job.data?.jobConfig}
                />
            </div>
            
            <div class="bg-base-800 py-3 px-4 border-t border-base-700 flex justify-end">
                <Button 
                    variant="outline" 
                    onclick={() => pipelineEditorOpen = false}
                    class="mr-2"
                    disabled={savingPipeline}
                >
                    Cancel
                </Button>
                <Button 
                    variant="primary" 
                    class="save-button"
                    onclick={() => {
                        // TRIGGER SAVE IN THE JOBBUILDER COMPONENT
                        const saveButton = document.querySelector('.job-builder .save-button');
                        if (saveButton) saveButton.click();
                    }}
                    disabled={savingPipeline}
                >
                    {#if savingPipeline}
                        <Loading size="lg" text="Saving..."/>
                    {:else}
                        <Save class="h-5 w-5 mr-1" />
                        Save Pipeline
                    {/if}
                </Button>
            </div>
        </div>
    </div>
{/if}

<!-- EDIT BASIC INFO MODAL -->
{#if editBasicInfoOpen && job && editingJob}
    <div class="modal modal-open">
        <div class="modal-box max-w-xl">
            <h3 class="font-bold text-lg mb-4">Edit Job Details</h3>
            
            <div class="space-y-4">
                <div>
                    <label for="edit-job-name" class="block text-sm font-medium text-dark-300 mb-1">
                        Job Name <span class="text-danger-500">*</span>
                    </label>
                    <input
                        id="edit-job-name"
                        type="text"
                        bind:value={editingJob.name}
                        placeholder="E.g., Product Scraper"
                        class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                    />
                </div>
                
                <div>
                    <label for="edit-base-url" class="block text-sm font-medium text-dark-300 mb-1">
                        Base URL <span class="text-danger-500">*</span>
                    </label>
                    <input
                        id="edit-base-url"
                        type="text"
                        bind:value={editingJob.baseUrl}
                        placeholder="https://example.com"
                        class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                    />
                </div>
                
                <div>
                    <label for="edit-description" class="block text-sm font-medium text-dark-300 mb-1">
                        Description
                    </label>
                    <textarea
                        id="edit-description"
                        bind:value={editingJob.description}
                        placeholder="Describe the purpose of this job..."
                        rows="3"
                        class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                    ></textarea>
                </div>
                
                <div>
                    <label for="edit-schedule" class="block text-sm font-medium text-dark-300 mb-1">
                        Schedule (CRON Expression)
                    </label>
                    <input
                        id="edit-schedule"
                        type="text"
                        bind:value={editingJob.schedule}
                        placeholder="E.g., 0 0 * * * (daily at midnight)"
                        class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                    />
                    <p class="text-xs text-dark-400 mt-1">
                        Leave empty for manual execution only
                    </p>
                </div>
            </div>
            
            <div class="flex justify-end space-x-3 mt-6">
                <Button variant="outline" onclick={() => editBasicInfoOpen = false}>
                    Cancel
                </Button>
                <Button variant="primary" onclick={handleSaveBasicInfo}>
                    Save Changes
                </Button>
            </div>
        </div>
        <div
            class="modal-backdrop"
            onclick={() => editBasicInfoOpen = false}
            onkeydown={() => {}}
            role="button"
            aria-label="Close modal"
            tabindex="0"
        >
            <button>close</button>
        </div>
    </div>
{/if}
