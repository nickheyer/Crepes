<script>
    import { 
        Plus, 
        Image, 
        Copy, 
        RefreshCw, 
        HelpCircle 
    } from "lucide-svelte";
    import { state as jobState } from "$lib/stores/jobStore.svelte.js";
    import { addToast } from "$lib/stores/uiStore.svelte.js";

    // ACTION FUNCTIONS
    function openNewJobModal() {
        jobState.createJobModal = true;
    }
    
    function goToAssets() {
        window.location.href = "/assets";
    }
    
    function refreshJobs() {
        addToast("Refreshing jobs...", "info");
        // TODO: ACTUALLY REFRESH JOBS
        setTimeout(() => {
            addToast("Jobs refreshed successfully", "success");
        }, 500);
    }
    
    function viewDocumentation() {
        window.open("https://github.com/nickheyer/Crepes", "_blank");
    }
</script>

<div class="flex flex-col gap-3">
    <button 
        class="btn btn-primary w-full flex justify-start gap-3 px-4 shadow-md hover:shadow-lg" 
        onclick={openNewJobModal}
    >
        <Plus size={20} strokeWidth={2.5} />
        <span class="font-medium">Create New Job</span>
    </button>
    <button 
        class="btn btn-secondary w-full flex justify-start gap-3 px-4 shadow-md hover:shadow-lg" 
        onclick={goToAssets}
    >
        <Image size={20} strokeWidth={2.5} />
        <span class="font-medium">View Asset Gallery</span>
    </button>
    <button 
        class="btn btn-info w-full flex justify-start gap-3 px-4 shadow-md hover:shadow-lg" 
        onclick={refreshJobs}
    >
        <RefreshCw size={20} strokeWidth={2.5} />
        <span class="font-medium">Refresh Jobs</span>
    </button>
    <button 
        class="btn btn-neutral w-full flex justify-start gap-3 px-4 shadow-md hover:shadow-lg" 
        onclick={viewDocumentation}
    >
        <HelpCircle size={20} strokeWidth={2.5} />
        <span class="font-medium">Documentation</span>
    </button>
</div>
