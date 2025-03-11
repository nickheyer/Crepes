<script>
    import { onMount } from "svelte";
    import { isValidRegex } from "$lib/utils/validation";
    import { state as jobState, setStepValidity } from "$lib/stores/jobStore.svelte";

    // INITIALIZE LOCAL STATE
    let filters = $state(jobState.formData.data.filters || []);
    let rules = $state(
        jobState.formData.data.rules || {
            maxDepth: 3,
            maxAssets: 0,
            maxPages: 0,
            maxConcurrent: 5,
            includeUrlPattern: "",
            excludeUrlPattern: "",
            requestDelay: 0,
            randomizeDelay: false,
        },
    );
    let editingFilter = $state({
        id: "",
        name: "",
        type: "url",
        pattern: "",
        action: "include",
        description: "",
    });
    let editingIndex = $state(-1);
    let isValid = $state(false);

    // FILTER TYPES
    const filterTypes = [
        {
            id: "url",
            label: "URL Pattern",
            description: "Filter based on page URL",
        },
        {
            id: "content",
            label: "Page Content",
            description: "Filter based on page content",
        },
        {
            id: "file",
            label: "File Type",
            description: "Filter based on file type/extension",
        },
        {
            id: "size",
            label: "File Size",
            description: "Filter based on file size",
        },
    ];

    // SETUP ON COMPONENT MOUNT
    onMount(() => {
        resetEditingFilter();
        validate();
    });

    // ADD OR UPDATE FILTER
    function addFilter() {
        if (!editingFilter.name || !editingFilter.pattern) return;

        if (editingIndex >= 0) {
            // Update existing filter
            filters[editingIndex] = { ...editingFilter };
        } else {
            // Add new filter
            filters = [...filters, { ...editingFilter }];
        }

        // Update the store
        jobState.formData.data.filters = [...filters];

        // Reset form
        resetEditingFilter();
        editingIndex = -1;
        validate();
    }

    // EDIT FILTER
    function editFilter(index) {
        editingFilter = { ...filters[index] };
        editingIndex = index;
    }

    // REMOVE FILTER
    function removeFilter(index) {
        filters = filters.filter((_, i) => i !== index);
        jobState.formData.data.filters = [...filters];
        
        if (editingIndex === index) {
            resetEditingFilter();
            editingIndex = -1;
        }
        validate();
    }

    // RESET EDITING FORM
    function resetEditingFilter() {
        editingFilter = {
            id: generateId(),
            name: "",
            type: "url",
            pattern: "",
            action: "include",
            description: "",
        };
        editingIndex = -1;
    }

    // GENERATE RANDOM ID
    function generateId() {
        return "filter_" + Math.random().toString(36).substring(2, 11);
    }

    // VALIDATE CONFIGURATION
    function validate() {
        // Check URL patterns if provided
        let valid = true;

        if (rules.includeUrlPattern && !isValidRegex(rules.includeUrlPattern)) {
            valid = false;
        }

        if (rules.excludeUrlPattern && !isValidRegex(rules.excludeUrlPattern)) {
            valid = false;
        }

        // Validate filters
        filters.forEach((filter) => {
            if (!isValidRegex(filter.pattern)) {
                valid = false;
            }
        });

        isValid = valid;
        setStepValidity(3, valid);
        return valid;
    }

    // UPDATE FORM WITH PROCESSED DATA
    function updateFormData() {
        // Create a new object to avoid reactivity issues
        const updatedRules = {
            ...rules,
            maxDepth: parseInt(rules.maxDepth) || 0,
            maxAssets: parseInt(rules.maxAssets) || 0,
            maxPages: parseInt(rules.maxPages) || 0,
            maxConcurrent: parseInt(rules.maxConcurrent) || 5,
            requestDelay: parseInt(rules.requestDelay) || 0,
        };
        
        // Only update if values have changed
        if (JSON.stringify(jobState.formData.data.rules) !== JSON.stringify(updatedRules)) {
            jobState.formData.data.rules = updatedRules;
        }
    }

    // FIX FOR INFINITE LOOP: ADD EXPLICIT DEPENDENCIES
    $effect(() => {
        // Explicitly track all rule properties that we need to watch
        const watchedRules = {
            maxDepth: rules.maxDepth,
            maxAssets: rules.maxAssets,
            maxPages: rules.maxPages,
            maxConcurrent: rules.maxConcurrent,
            requestDelay: rules.requestDelay,
            includeUrlPattern: rules.includeUrlPattern,
            excludeUrlPattern: rules.excludeUrlPattern,
            randomizeDelay: rules.randomizeDelay
        };
        
        // Now updateFormData will only run when these values change
        updateFormData();
    });
</script>

<div>
    <h2 class="text-xl font-semibold mb-4">Filtering & Limits</h2>
    <p class="text-dark-300 mb-6">
        Set limits and filter criteria for your scraping job
    </p>

    <!-- Crawl rules section -->
    <div class="bg-base-800 rounded-lg p-4 mb-6">
        <h3 class="text-sm font-medium mb-4">Crawling Limits</h3>

        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div>
                <label
                    for="max-depth"
                    class="block text-sm font-medium text-dark-300 mb-1"
                >
                    Maximum Depth
                </label>
                <input
                    id="max-depth"
                    type="number"
                    min="0"
                    bind:value={rules.maxDepth}
                    class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                />
                <p class="mt-1 text-xs text-dark-400">
                    How many links deep to follow (0 = unlimited)
                </p>
            </div>

            <div>
                <label
                    for="max-assets"
                    class="block text-sm font-medium text-dark-300 mb-1"
                >
                    Maximum Assets
                </label>
                <input
                    id="max-assets"
                    type="number"
                    min="0"
                    bind:value={rules.maxAssets}
                    class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                />
                <p class="mt-1 text-xs text-dark-400">
                    Maximum number of assets to download (0 = unlimited)
                </p>
            </div>

            <div>
                <label
                    for="max-pages"
                    class="block text-sm font-medium text-dark-300 mb-1"
                >
                    Maximum Pages
                </label>
                <input
                    id="max-pages"
                    type="number"
                    min="0"
                    bind:value={rules.maxPages}
                    class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                />
                <p class="mt-1 text-xs text-dark-400">
                    Maximum number of pages to visit (0 = unlimited)
                </p>
            </div>

            <div>
                <label
                    for="max-concurrent"
                    class="block text-sm font-medium text-dark-300 mb-1"
                >
                    Concurrent Connections
                </label>
                <input
                    id="max-concurrent"
                    type="number"
                    min="1"
                    max="20"
                    bind:value={rules.maxConcurrent}
                    class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                />
                <p class="mt-1 text-xs text-dark-400">
                    How many parallel connections to use (1-20)
                </p>
            </div>

            <div>
                <label
                    for="request-delay"
                    class="block text-sm font-medium text-dark-300 mb-1"
                >
                    Request Delay (ms)
                </label>
                <input
                    id="request-delay"
                    type="number"
                    min="0"
                    bind:value={rules.requestDelay}
                    class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                />
                <p class="mt-1 text-xs text-dark-400">
                    Delay between requests in milliseconds (0 = no delay)
                </p>
            </div>

            <div class="flex items-center">
                <input
                    id="randomize-delay"
                    type="checkbox"
                    bind:checked={rules.randomizeDelay}
                    class="h-4 w-4 text-primary-600 focus:ring-primary-500 border-dark-500 rounded"
                />
                <label
                    for="randomize-delay"
                    class="ml-2 block text-sm text-dark-300"
                >
                    Randomize delay (helps avoid rate limiting)
                </label>
            </div>
        </div>
    </div>

    <!-- URL patterns section -->
    <div class="bg-base-800 rounded-lg p-4 mb-6">
        <h3 class="text-sm font-medium mb-4">URL Patterns</h3>

        <div class="grid grid-cols-1 gap-4">
            <div>
                <label
                    for="include-pattern"
                    class="block text-sm font-medium text-dark-300 mb-1"
                >
                    Include URLs (regex)
                </label>
                <input
                    id="include-pattern"
                    type="text"
                    bind:value={rules.includeUrlPattern}
                    placeholder="E.g., /products/.* (leave empty to include all)"
                    class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                />
                <p class="mt-1 text-xs text-dark-400">
                    Only crawl URLs matching this pattern
                </p>
            </div>

            <div>
                <label
                    for="exclude-pattern"
                    class="block text-sm font-medium text-dark-300 mb-1"
                >
                    Exclude URLs (regex)
                </label>
                <input
                    id="exclude-pattern"
                    type="text"
                    bind:value={rules.excludeUrlPattern}
                    placeholder="E.g., .*\.pdf$ (leave empty to exclude none)"
                    class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                />
                <p class="mt-1 text-xs text-dark-400">
                    Skip URLs matching this pattern
                </p>
            </div>
        </div>
    </div>

    <!-- Custom filters section -->
    <div class="bg-base-800 rounded-lg p-4 mb-6">
        <h3 class="text-sm font-medium mb-4">Custom Filters</h3>

        {#if filters.length > 0}
            <div class="mb-6">
                <div class="bg-base-850 rounded-lg overflow-hidden">
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
                                    >Type</th
                                >
                                <th
                                    scope="col"
                                    class="px-4 py-3 text-left text-xs font-medium text-dark-300 uppercase tracking-wider"
                                    >Pattern</th
                                >
                                <th
                                    scope="col"
                                    class="px-4 py-3 text-left text-xs font-medium text-dark-300 uppercase tracking-wider"
                                    >Action</th
                                >
                                <th
                                    scope="col"
                                    class="px-4 py-3 text-right text-xs font-medium text-dark-300 uppercase tracking-wider"
                                    >Actions</th
                                >
                            </tr>
                        </thead>
                        <tbody class="divide-y divide-dark-700">
                            {#each filters as filter, i}
                                <tr class="hover:bg-base-750">
                                    <td class="px-4 py-3 whitespace-nowrap">
                                        <div class="text-sm font-medium">
                                            {filter.name}
                                        </div>
                                    </td>
                                    <td class="px-4 py-3 whitespace-nowrap">
                                        <span class="text-sm"
                                            >{filter.type}</span
                                        >
                                    </td>
                                    <td class="px-4 py-3">
                                        <div
                                            class="text-xs font-mono text-dark-300 break-all"
                                        >
                                            {filter.pattern}
                                        </div>
                                    </td>
                                    <td class="px-4 py-3 whitespace-nowrap">
                                        <span
                                            class={`px-2 py-1 inline-flex text-xs leading-5 font-semibold rounded-full 
                        ${filter.action === "include" ? "bg-green-500 text-green-100" : "bg-red-500 text-red-100"}`}
                                        >
                                            {filter.action}
                                        </span>
                                    </td>
                                    <td
                                        class="px-4 py-3 whitespace-nowrap text-right text-sm font-medium"
                                    >
                                        <button
                                            class="text-primary-400 hover:text-primary-300 mr-3"
                                            onclick={() => editFilter(i)}
                                        >
                                            Edit
                                        </button>
                                        <button
                                            class="text-danger-400 hover:text-danger-300"
                                            onclick={() => removeFilter(i)}
                                        >
                                            Delete
                                        </button>
                                    </td>
                                </tr>
                            {/each}
                        </tbody>
                    </table>
                </div>
            </div>
        {/if}

        <!-- Filter form -->
        <div class="bg-base-850 rounded-lg p-4">
            <h4 class="text-sm font-medium mb-3">
                {editingIndex >= 0 ? "Edit Filter" : "Add New Filter"}
            </h4>
            <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <div>
                    <label
                        for="filter-name"
                        class="block text-sm font-medium text-dark-300 mb-1"
                    >
                        Name <span class="text-danger-500">*</span>
                    </label>
                    <input
                        id="filter-name"
                        type="text"
                        bind:value={editingFilter.name}
                        placeholder="E.g., Skip PDFs"
                        class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                    />
                </div>

                <div>
                    <label
                        for="filter-type"
                        class="block text-sm font-medium text-dark-300 mb-1"
                    >
                        Filter Type
                    </label>
                    <select
                        id="filter-type"
                        bind:value={editingFilter.type}
                        class="select select-bordered w-full"
                    >
                        {#each filterTypes as type}
                            <option value={type.id}>{type.label}</option>
                        {/each}
                    </select>
                </div>

                <div class="sm:col-span-2">
                    <label
                        for="filter-pattern"
                        class="block text-sm font-medium text-dark-300 mb-1"
                    >
                        Pattern (regex) <span class="text-danger-500">*</span>
                    </label>
                    <input
                        id="filter-pattern"
                        type="text"
                        bind:value={editingFilter.pattern}
                        placeholder="E.g., \.pdf$"
                        class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                    />
                </div>

                <div>
                    <label
                        for="filter-action"
                        class="block text-sm font-medium text-dark-300 mb-1"
                    >
                        Action
                    </label>
                    <select
                        id="filter-action"
                        bind:value={editingFilter.action}
                        class="select select-bordered w-full"
                    >
                        <option value="include">Include matching items</option>
                        <option value="exclude">Exclude matching items</option>
                    </select>
                </div>

                <div>
                    <label
                        for="filter-description"
                        class="block text-sm font-medium text-dark-300 mb-1"
                    >
                        Description
                    </label>
                    <input
                        id="filter-description"
                        type="text"
                        bind:value={editingFilter.description}
                        placeholder="Optional description"
                        class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                    />
                </div>
            </div>

            <div class="mt-4 flex justify-end space-x-3">
                {#if editingIndex >= 0}
                    <button
                        class="px-3 py-1.5 text-sm border border-dark-600 rounded-md focus:outline-none hover:bg-base-700"
                        onclick={resetEditingFilter}
                    >
                        Cancel
                    </button>
                {/if}
                <button
                    class="px-3 py-1.5 text-sm bg-primary-600 text-white rounded-md focus:outline-none hover:bg-primary-700 disabled:opacity-50 disabled:cursor-not-allowed"
                    onclick={addFilter}
                    disabled={!editingFilter.name || !editingFilter.pattern}
                >
                    {editingIndex >= 0 ? "Update Filter" : "Add Filter"}
                </button>
            </div>
        </div>
    </div>

    <!-- Help section -->
    <div class="bg-base-850 rounded-lg p-4">
        <h4 class="text-sm font-medium mb-2">Regex Pattern Examples</h4>
        <ul class="text-xs text-dark-300 list-disc pl-5 space-y-1">
            <li>
                <code class="px-1 py-0.5 rounded bg-base-700 text-xs font-mono"
                    >\.jpg$</code
                > - Match URLs ending with .jpg
            </li>
            <li>
                <code class="px-1 py-0.5 rounded bg-base-700 text-xs font-mono"
                    >/product/[0-9]+</code
                > - Match product pages with numeric IDs
            </li>
            <li>
                <code class="px-1 py-0.5 rounded bg-base-700 text-xs font-mono"
                    >^https://example.com/blog/</code
                > - Match only blog pages
            </li>
            <li>
                <code class="px-1 py-0.5 rounded bg-base-700 text-xs font-mono"
                    >(large|medium|small)</code
                > - Match URLs containing any of these words
            </li>
        </ul>
    </div>
</div>
