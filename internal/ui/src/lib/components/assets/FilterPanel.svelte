<script>
    import { onMount } from "svelte";
    import {
        state as assetState,
        updateFilters,
        resetFilters,
    } from "$lib/stores/assetStore.svelte";
    import { state as jobState, loadJobs } from "$lib/stores/jobStore.svelte";
    import Button from "../common/Button.svelte";
    import Card from "../common/Card.svelte";
    import {
        Search,
        Menu,
        RefreshCcw
    } from "lucide-svelte";
    
    // LOCAL STATE
    let expanded = $state(false);
    let jobsLoaded = $state(false);
    
    // LOCAL FILTER STATE TO AVOID TOO MANY UPDATES
    let filterState = $state({
        type: assetState.assetFilters.type || "",
        jobId: assetState.assetFilters.jobId || "",
        search: assetState.assetFilters.search || "",
        dateRange: {
            from: assetState.assetFilters.dateRange?.from || null,
            to: assetState.assetFilters.dateRange?.to || null,
        },
        sortBy: assetState.assetFilters.sortBy || "date",
        sortDirection: assetState.assetFilters.sortDirection || "desc",
    });
    
    // ASSET TYPES
    const assetTypes = [
        { id: "", label: "All Types" },
        { id: "image", label: "Images" },
        { id: "video", label: "Videos" },
        { id: "audio", label: "Audio" },
        { id: "document", label: "Documents" },
    ];
    
    // SORT OPTIONS
    const sortOptions = [
        { id: "date", label: "Date" },
        { id: "title", label: "Title" },
        { id: "type", label: "Type" },
        { id: "size", label: "Size" },
    ];
    
    onMount(async () => {
        if (!jobState.jobs || jobState.jobs.length === 0) {
            await loadJobs();
        }
        jobsLoaded = true;
    });
    
    function toggleExpanded() {
        expanded = !expanded;
    }
    
    function applyFilters() {
        updateFilters(filterState);
    }
    
    function handleReset() {
        resetFilters();
        filterState = {
            type: "",
            jobId: "",
            search: "",
            dateRange: {
                from: null,
                to: null,
            },
            sortBy: "date",
            sortDirection: "desc",
        };
    }
    
    // AUTO-APPLY FILTERS WHEN SEARCH CHANGES
    $effect(() => {
        if (filterState.search !== assetState.assetFilters.search) {
            // DEBOUNCE SEARCH FILTER APPLICATION
            const timeout = setTimeout(() => {
                updateFilters({ search: filterState.search });
            }, 300);
            return () => clearTimeout(timeout);
        }
    });
</script>

<Card>
    <div
        class="flex flex-col md:flex-row items-start md:items-center justify-between space-y-3 md:space-y-0"
    >
        <!-- SEARCH BAR -->
        <div class="w-full md:w-2/3 relative">
            <div
                class="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none"
            >
                <Search class="h-5 w-5 text-dark-400" />
            </div>
            <input
                type="text"
                placeholder="Search assets..."
                bind:value={filterState.search}
                class="pl-10 pr-4 py-2 w-full rounded-md bg-base-700 border border-dark-600 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
            />
        </div>
        
        <!-- FILTER TOGGLE AND APPLY BUTTON -->
        <div class="flex w-full md:w-auto space-x-2">
            <Button
                variant="outline"
                onclick={toggleExpanded}
                class="flex-1 md:flex-none"
            >
                <Menu class="h-5 w-5 mr-1" />
                Filters
                <span
                    class="ml-1.5 flex h-5 w-5 items-center justify-center rounded-full bg-primary-600 text-xs font-semibold text-white"
                >
                    {Object.values(filterState).filter(
                        (v) =>
                            (v && typeof v === "string" && v !== "") ||
                            (v &&
                                typeof v === "object" &&
                                Object.values(v).some((sv) => sv)),
                    ).length}
                </span>
            </Button>
            <Button
                variant="outline"
                onclick={handleReset}
                class="flex-none"
                aria-label="Reset filters"
            >
                <RefreshCcw class="h-5 w-5" />
            </Button>
        </div>
    </div>
    
    {#if expanded}
        <div
            class="mt-4 pt-4 border-t border-dark-700 grid grid-cols-1 md:grid-cols-3 gap-4"
        >
            <!-- ASSET TYPE FILTER -->
            <div>
                <label
                    for="asset-type"
                    class="block text-sm font-medium text-dark-300 mb-1"
                >
                    Asset Type
                </label>
                <div class="relative">
                    <select
                        id="asset-type"
                        bind:value={filterState.type}
                        class="select select-bordered w-full"
                    >
                        {#each assetTypes as type}
                            <option value={type.id}
                                >{type.label}
                                {type.id
                                    ? `(${assetState.assetCounts[type.id] || 0})`
                                    : ""}</option
                            >
                        {/each}
                    </select>
                </div>
            </div>
            
            <!-- JOB FILTER -->
            <div>
                <label
                    for="job-filter"
                    class="block text-sm font-medium text-dark-300 mb-1"
                >
                    Source Job
                </label>
                <div class="relative">
                    <select
                        id="job-filter"
                        bind:value={filterState.jobId}
                        class="select select-bordered w-full"
                    >
                        <option value="">All Jobs</option>
                        {#if jobsLoaded}
                            {#each jobState.jobs as job}
                                <option value={job.id}
                                    >{job.name || job.baseUrl}</option
                                >
                            {/each}
                        {/if}
                    </select>
                </div>
            </div>
            
            <!-- DATE RANGE -->
            <div>
                <legend class="block text-sm font-medium text-dark-300 mb-1">
                    Date Range
                </legend>
                <div class="flex space-x-2">
                    <input
                        type="date"
                        bind:value={filterState.dateRange.from}
                        placeholder="From"
                        class="w-1/2 rounded-md bg-base-700 border border-dark-600 py-2 px-3 text-white focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                    />
                    <input
                        type="date"
                        bind:value={filterState.dateRange.to}
                        placeholder="To"
                        class="w-1/2 rounded-md bg-base-700 border border-dark-600 py-2 px-3 text-white focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                    />
                </div>
            </div>
            
            <!-- SORT OPTIONS -->
            <div>
                <label
                    for="sort-by"
                    class="block text-sm font-medium text-dark-300 mb-1"
                >
                    Sort By
                </label>
                <div class="flex space-x-2">
                    <select
                        id="sort-by"
                        bind:value={filterState.sortBy}
                        class="select select-bordered w-2/3"
                    >
                        {#each sortOptions as option}
                            <option value={option.id}>{option.label}</option>
                        {/each}
                    </select>
                    <select
                        id="sort-direction"
                        bind:value={filterState.sortDirection}
                        class="select select-bordered w-1/3"
                    >
                        <option value="desc">Descending</option>
                        <option value="asc">Ascending</option>
                    </select>
                </div>
            </div>
            
            <!-- APPLY BUTTON -->
            <div class="md:col-span-3 flex justify-end">
                <Button variant="primary" onclick={applyFilters}>
                    Apply Filters
                </Button>
            </div>
        </div>
    {/if}
</Card>
