<script>
    import { onMount } from "svelte";
    import {
        jobs,
        runningJobs,
        completedJobs,
        failedJobs,
        loadJobs,
        startJobById,
        stopJobById,
    } from "$lib/stores/jobStore";
    import { assetCounts, loadAssets } from "$lib/stores/assetStore";
    import { createJobModal } from "$lib/stores/jobStore";
    import Button from "$lib/components/common/Button.svelte";
    import StatCard from "$lib/components/dashboard/StatCard.svelte";
    import ActivityFeed from "$lib/components/dashboard/ActivityFeed.svelte";
    import QuickActions from "$lib/components/dashboard/QuickActions.svelte";
    import JobWizard from "$lib/components/jobs/JobWizard.svelte";
    import { formatDate, formatRelativeTime } from "$lib/utils/formatters";
    
    import { Play, StopCircle, Eye } from "lucide-svelte";
    
    // Local state
    let loading = $state(true);
    
    // Load data on mount
    onMount(async () => {
        try {
            await Promise.all([loadJobs(), loadAssets()]);
        } catch (error) {
            console.error("Error loading dashboard data:", error);
        } finally {
            loading = false;
        }
    });
    
    // Create new job
    function openNewJobModal() {
        createJobModal.set(true);
    }
</script>

<svelte:head>
    <title>Dashboard | Crepes</title>
</svelte:head>

<section class="space-y-6">
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard
            title="Total Jobs"
            value={$jobs.length}
            icon="briefcase"
            trend={null}
            href="/jobs"
        />
        <StatCard
            title="Running Jobs"
            value={$runningJobs.length}
            icon="play"
            color="success"
            href="/jobs"
        />
        <StatCard
            title="Total Assets"
            value={$assetCounts.total || 0}
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
    
    <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <!-- Recent Jobs -->
        <div class="card bg-base-100 shadow-xl lg:col-span-2">
            <div class="card-body">
                <h2 class="card-title">Recent Jobs</h2>
                
                {#if loading}
                    <div class="py-20 flex justify-center">
                        <span class="loading loading-spinner loading-lg text-primary"></span>
                    </div>
                {:else if $jobs.length === 0}
                    <div class="py-12 text-center">
                        <div class="text-6xl opacity-20 mx-auto mb-4">📦</div>
                        <h3 class="text-lg font-medium mb-2">No jobs found</h3>
                        <p class="text-base-content/60 mb-4">Start by creating your first job</p>
                        <button class="btn btn-primary" onclick={openNewJobModal}>Create New Job</button>
                    </div>
                {:else}
                    <div class="overflow-x-auto">
                        <table class="table table-zebra">
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
                                {#each $jobs.slice(0, 5) as job}
                                    <tr>
                                        <td>
                                            <a
                                                href={`/jobs/${job.id}`}
                                                class="font-medium hover:text-primary"
                                            >
                                                {job.name || "Unnamed Job"}
                                            </a>
                                            <div class="text-xs opacity-60 truncate max-w-xs">
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
                                                    "badge-ghost"
                                                }`}
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
                                                <button
                                                    class="btn btn-warning btn-xs"
                                                    onclick={() => stopJobById(job.id)}
                                                >
                                                    <StopCircle size={14} />
                                                    Stop
                                                </button>
                                            {:else}
                                                <button
                                                    class="btn btn-success btn-xs"
                                                    onclick={() => startJobById(job.id)}
                                                >
                                                    <Play size={14} />
                                                    Start
                                                </button>
                                            {/if}
                                            <a
                                                href={`/jobs/${job.id}`}
                                                class="btn btn-ghost btn-xs"
                                            >
                                                <Eye size={14} />
                                                View
                                            </a>
                                        </td>
                                    </tr>
                                {/each}
                            </tbody>
                        </table>
                    </div>
                    <div class="card-actions justify-end mt-3">
                        <a
                            href="/jobs"
                            class="link link-primary link-hover"
                        >
                            View All Jobs →
                        </a>
                    </div>
                {/if}
            </div>
        </div>
        
        <!-- Quick Actions -->
        <div class="card bg-base-100 shadow-xl">
            <div class="card-body">
                <h2 class="card-title">Quick Actions</h2>
                <QuickActions />
            </div>
        </div>
        
        <!-- Activity Feed -->
        <div class="card bg-base-100 shadow-xl lg:col-span-2">
            <div class="card-body">
                <h2 class="card-title">Recent Activity</h2>
                <ActivityFeed />
            </div>
        </div>
        
        <!-- Asset Stats -->
        <div class="card bg-base-100 shadow-xl">
            <div class="card-body">
                <h2 class="card-title">Asset Distribution</h2>
                
                {#if loading}
                    <div class="py-10 flex justify-center">
                        <span class="loading loading-spinner loading-md text-primary"></span>
                    </div>
                {:else}
                    <div class="space-y-3">
                        <!-- Images -->
                        <div>
                            <div class="flex justify-between items-center mb-1">
                                <span class="text-sm font-medium">Images</span>
                                <span class="text-sm opacity-70">{$assetCounts.image || 0}</span>
                            </div>
                            <progress 
                                class="progress progress-info w-full" 
                                value={$assetCounts.total > 0 ? (($assetCounts.image || 0) / $assetCounts.total) * 100 : 0} 
                                max="100"
                            ></progress>
                        </div>
                        
                        <!-- Videos -->
                        <div>
                            <div class="flex justify-between items-center mb-1">
                                <span class="text-sm font-medium">Videos</span>
                                <span class="text-sm opacity-70">{$assetCounts.video || 0}</span>
                            </div>
                            <progress 
                                class="progress progress-error w-full" 
                                value={$assetCounts.total > 0 ? (($assetCounts.video || 0) / $assetCounts.total) * 100 : 0} 
                                max="100"
                            ></progress>
                        </div>
                        
                        <!-- Audio -->
                        <div>
                            <div class="flex justify-between items-center mb-1">
                                <span class="text-sm font-medium">Audio</span>
                                <span class="text-sm opacity-70">{$assetCounts.audio || 0}</span>
                            </div>
                            <progress 
                                class="progress progress-success w-full" 
                                value={$assetCounts.total > 0 ? (($assetCounts.audio || 0) / $assetCounts.total) * 100 : 0} 
                                max="100"
                            ></progress>
                        </div>
                        
                        <!-- Documents -->
                        <div>
                            <div class="flex justify-between items-center mb-1">
                                <span class="text-sm font-medium">Documents</span>
                                <span class="text-sm opacity-70">{$assetCounts.document || 0}</span>
                            </div>
                            <progress 
                                class="progress progress-warning w-full" 
                                value={$assetCounts.total > 0 ? (($assetCounts.document || 0) / $assetCounts.total) * 100 : 0} 
                                max="100"
                            ></progress>
                        </div>
                        
                        <!-- Other -->
                        <div>
                            <div class="flex justify-between items-center mb-1">
                                <span class="text-sm font-medium">Other</span>
                                <span class="text-sm opacity-70">{$assetCounts.unknown || 0}</span>
                            </div>
                            <progress 
                                class="progress progress-secondary w-full" 
                                value={$assetCounts.total > 0 ? (($assetCounts.unknown || 0) / $assetCounts.total) * 100 : 0} 
                                max="100"
                            ></progress>
                        </div>
                    </div>
                    
                    <div class="card-actions justify-end mt-4">
                        <a
                            href="/assets"
                            class="link link-primary link-hover"
                        >
                            View All Assets →
                        </a>
                    </div>
                {/if}
            </div>
        </div>
    </div>
</section>

<!-- Job Creation Modal -->
{#if $createJobModal}
    <div class="modal modal-open">
        <div class="modal-box max-w-5xl">
            <JobWizard
                onclick={() => createJobModal.set(false)}
                oncancel={() => createJobModal.set(false)}
            />
        </div>
        <div 
            class="modal-backdrop" 
            role="button" 
            tabindex="0" 
            onclick={() => createJobModal.set(false)}
            onkeydown={(e) => e.key === 'Escape' && createJobModal.set(false)}
            aria-label="Close modal"
        ></div>
    </div>
{/if}
