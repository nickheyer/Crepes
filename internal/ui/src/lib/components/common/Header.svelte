<script>
    import {
        Menu,
        Plus,
        Search,
    } from "lucide-svelte";
    import { state as uiState } from "$lib/stores/uiStore.svelte";
    import { state as jobState } from "$lib/stores/jobStore.svelte";
    
    // PROPS
    let { currentPage = "dashboard" } = $props();
    
    const pageTitles = {
        dashboard: "Dashboard",
        jobs: "Jobs",
        assets: "Assets Gallery",
        settings: "Settings",
        help: "Help & Documentation",
    };
    
    function toggleSidebar() {
        uiState.isSidebarOpen = !uiState.isSidebarOpen;
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
</div>