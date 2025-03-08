<script>
    import {
        Menu,
        Plus,
        Search,
    } from "lucide-svelte";
    import { isSidebarOpen } from "$lib/stores/uiStore";
    import { createJobModal } from "$lib/stores/jobStore";
    
    // PROPS
    let { currentPage = "dashboard" } = $props();
    
    const pageTitles = {
        dashboard: "Dashboard",
        jobs: "Jobs",
        assets: "Assets Gallery",
        templates: "Templates",
        settings: "Settings",
        help: "Help & Documentation",
    };
    
    function toggleSidebar() {
        isSidebarOpen.update((open) => !open);
    }
    
    function openNewJobModal() {
        createJobModal.set(true);
    }
</script>
<div class="navbar bg-base-300">
    <div class="navbar-start">
        <label for="main-drawer" class="btn btn-ghost btn-circle drawer-button lg:hidden">
            <Menu size={20} />
        </label>
        <h1 class="text-xl font-semibold ml-2">
            {pageTitles[currentPage] || "Crepes"}
        </h1>
    </div>
    <div class="navbar-end">
        <!-- NEW JOB BUTTON - KEEPING THIS FUNCTIONAL -->
        <button class="btn btn-primary btn-sm" onclick={openNewJobModal}>
            <Plus size={16} />
            New Job
        </button>
    </div>
</div>