<script>
    import { onMount } from "svelte";
    import {
        state as jobState,
        runningJobs,
        completedJobs,
        failedJobs,
        loadJobs,
        startJobById,
        stopJobById
    } from "$lib/stores/jobStore.svelte.js";
    import { 
        state as assetState
    } from "$lib/stores/assetStore.svelte.js";
    import { loadAssets } from "$lib/stores/assetStore.svelte.js";
    import Card from "$lib/components/common/Card.svelte";
    import Button from "$lib/components/common/Button.svelte";
    import StatCard from "$lib/components/dashboard/StatCard.svelte";
    import ActivityFeed from "$lib/components/dashboard/ActivityFeed.svelte";
    import QuickActions from "$lib/components/dashboard/QuickActions.svelte";
    import JobBuilderModal from "$lib/components/jobs/JobBuilderModal.svelte";
    import { formatDate, formatRelativeTime } from "$lib/utils/formatters";
    import { addToast } from "$lib/stores/uiStore.svelte.js";
    
    let loading = $state(true);
    let newJobModalOpen = $state(false);
    
    onMount(async () => {
        try {
            await loadAssets();
            await loadJobs();
        } catch (error) {
            console.error("Error loading dashboard data:", error);
        } finally {
            loading = false;
        }
    });
    
    function openNewJobModal() {
        jobState.createJobModal = true;
    }
</script>

<svelte:head>
    <title>Dashboard | Crepes</title>
</svelte:head>

<div class="container mx-auto px-4">
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
        <StatCard
            title="Total Jobs"
            value={jobState.jobs.length}
            icon="briefcase"
            trend={null}
            href="/jobs"
        />
        <StatCard
            title="Running Jobs"
            value={runningJobs().length}
            icon="play"
            color="success"
            href="/jobs"
        />
        <StatCard
            title="Total Assets"
            value={assetState.assetCounts.total || 0}
            icon="photo"
            color="primary"
            trend={+15}
            href="/assets"
        />
        <StatCard
            title="Storage Used"
            value="1.2 GB"
            icon="database"
            trend={+5.4}
            href="/settings"
        />
    </div>
    <div class="grid grid-cols-1 lg:grid-cols-3 gap-4">
        <!-- RECENT JOBS -->
        <Card title="Recent Jobs" class="lg:col-span-2">
            {#if loading}
                <div class="flex justify-center items-center py-20">
                    <span class="loading loading-spinner loading-lg text-primary"></span>
                </div>
            {:else if jobState.jobs.length === 0}
                <div class="py-12 text-center">
                    <div class="flex justify-center mb-4">
                        <svg
                            xmlns="http://www.w3.org/2000/svg"
                            class="h-16 w-16 text-base-content opacity-40"
                            fill="none"
                            viewBox="0 0 24 24"
                            stroke="currentColor"
                        >
                            <path
                                stroke-linecap="round"
                                stroke-linejoin="round"
                                stroke-width="2"
                                d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"
                            />
                        </svg>
                    </div>
                    <h3 class="text-lg font-medium mb-2">No jobs found</h3>
                    <p class="text-base-content opacity-60 mb-4">
                        Start by creating your first job
                    </p>
                    <Button variant="primary" onclick={openNewJobModal}>Create New Job</Button>
                </div>
            {:else}
                <div class="overflow-x-auto">
                    <table class="table table-zebra w-full">
                        <thead>
                            <tr>
                                <th>Name</th>
                                <th>Status</th>
                                <th>Last Run</th>
                                <th>Assets</th>
                                <th class="text-right">Actions</th>
                            </tr>
                        </thead>
                        <tbody>
                            {#each jobState.jobs.slice(0, 5) as job}
                                <tr>
                                    <td>
                                        <a
                                            href={`/jobs/${job.id}`}
                                            class="font-medium hover:text-primary"
                                        >
                                            {job.name || "Unnamed Job"}
                                        </a>
                                        <div
                                            class="text-xs opacity-70 truncate max-w-xs"
                                        >
                                            {job.baseUrl}
                                        </div>
                                    </td>
                                    <td>
                                        <span
                                            class={`badge ${
                                                job.status === "running" ? "badge-success" : 
                                                job.status === "completed" ? "badge-info" : 
                                                job.status === "failed" ? "badge-error" : 
                                                job.status === "stopped" ? "badge-warning" : 
                                                "badge-ghost"}
                                            `}
                                        >
                                            {job.status || "idle"}
                                        </span>
                                    </td>
                                    <td>
                                        {job.lastRun
                                            ? formatRelativeTime(job.lastRun)
                                            : "Never"}
                                    </td>
                                    <td>
                                        {job.assets?.length || 0}
                                    </td>
                                    <td class="text-right">
                                        {#if job.status === "running"}
                                            <Button
                                                variant="warning"
                                                size="sm"
                                                onclick={() => stopJobById(job.id)}
                                            >
                                                Stop
                                            </Button>
                                        {:else}
                                            <Button
                                                variant="success"
                                                size="sm"
                                                onclick={() => startJobById(job.id)}
                                            >
                                                Start
                                            </Button>
                                        {/if}
                                        <a
                                            href={`/jobs/${job.id}`}
                                            class="btn btn-sm btn-ghost ml-2"
                                        >
                                            View
                                        </a>
                                    </td>
                                </tr>
                            {/each}
                        </tbody>
                    </table>
                </div>
                <div class="flex justify-end mt-4">
                    <a
                        href="/jobs"
                        class="link link-primary flex items-center"
                    >
                        View All Jobs →
                    </a>
                </div>
            {/if}
        </Card>
        <!-- QUICK ACTIONS -->
        <Card title="Quick Actions">
            <QuickActions />
        </Card>
        <!-- ACTIVITY FEED -->
        <Card title="Recent Activity" class="lg:col-span-2">
            <ActivityFeed />
        </Card>
        <!-- ASSET STATS -->
        <Card title="Asset Distribution">
            {#if loading}
                <div class="flex justify-center items-center py-20">
                    <span class="loading loading-spinner loading-lg text-primary"></span>
                </div>
            {:else}
                <div class="space-y-3">
                    <!-- Images -->
                    <div>
                        <div class="flex justify-between items-center mb-1">
                            <span class="font-medium">Images</span>
                            <span class="opacity-70"
                                >{assetState.assetCounts.image || 0}</span
                            >
                        </div>
                        <progress 
                            class="progress progress-info w-full" 
                            value={assetState.assetCounts.total > 0 ? ((assetState.assetCounts.image || 0) / assetState.assetCounts.total) * 100 : 0} 
                            max="100"
                        ></progress>
                    </div>
                    <!-- Videos -->
                    <div>
                        <div class="flex justify-between items-center mb-1">
                            <span class="font-medium">Videos</span>
                            <span class="opacity-70"
                                >{assetState.assetCounts.video || 0}</span
                            >
                        </div>
                        <progress 
                            class="progress progress-error w-full" 
                            value={assetState.assetCounts.total > 0 ? ((assetState.assetCounts.video || 0) / assetState.assetCounts.total) * 100 : 0} 
                            max="100"
                        ></progress>
                    </div>
                    <!-- Audio -->
                    <div>
                        <div class="flex justify-between items-center mb-1">
                            <span class="font-medium">Audio</span>
                            <span class="opacity-70"
                                >{assetState.assetCounts.audio || 0}</span
                            >
                        </div>
                        <progress 
                            class="progress progress-success w-full" 
                            value={assetState.assetCounts.total > 0 ? ((assetState.assetCounts.audio || 0) / assetState.assetCounts.total) * 100 : 0} 
                            max="100"
                        ></progress>
                    </div>
                    <!-- Documents -->
                    <div>
                        <div class="flex justify-between items-center mb-1">
                            <span class="font-medium">Documents</span>
                            <span class="opacity-70"
                                >{assetState.assetCounts.document || 0}</span
                            >
                        </div>
                        <progress 
                            class="progress progress-warning w-full" 
                            value={assetState.assetCounts.total > 0 ? ((assetState.assetCounts.document || 0) / assetState.assetCounts.total) * 100 : 0} 
                            max="100"
                        ></progress>
                    </div>
                    <!-- Other -->
                    <div>
                        <div class="flex justify-between items-center mb-1">
                            <span class="font-medium">Other</span>
                            <span class="opacity-70"
                                >{assetState.assetCounts.unknown || 0}</span
                            >
                        </div>
                        <progress 
                            class="progress progress-secondary w-full" 
                            value={assetState.assetCounts.total > 0 ? ((assetState.assetCounts.unknown || 0) / assetState.assetCounts.total) * 100 : 0} 
                            max="100"
                        ></progress>
                    </div>
                </div>
                <div class="flex justify-end mt-4 pt-3 border-t border-base-300">
                    <a
                        href="/assets"
                        class="link link-primary flex items-center"
                    >
                        View All Assets →
                    </a>
                </div>
            {/if}
        </Card>
    </div>
</div>

<!-- CREATE NEW JOB MODAL -->
<JobBuilderModal 
  isOpen={jobState.createJobModal} 
  onClose={() => jobState.createJobModal = false}
  onJobCreated={async () => {
    jobState.createJobModal = false;
    await loadJobs();
  }}
/>