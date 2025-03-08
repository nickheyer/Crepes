<script>
    import { onMount } from "svelte";
    import { createEventDispatcher } from "svelte";
    import Button from "$lib/components/common/Button.svelte";
    import VisualSelector from "../VisualSelector.svelte";
    
    // Create dispatch function
    const dispatch = createEventDispatcher();
    
    // Props
    let { formData = {} } = $props();
    
    // Local state
    let selectors = $state(formData.selectors || []);
    let currentView = $state("list"); // 'list' or 'visual'
    let newSelector = $state({
        id: "",
        name: "",
        type: "css",
        value: "",
        attributeSource: "",
        attribute: "src",
        description: "",
        purpose: "assets",
        priority: 0,
        isOptional: false,
        urlPattern: "",
    });
    let editingIndex = $state(-1);
    let visualUrl = $state(formData.baseUrl || "");
    let isValid = $state(false);
    let shouldUpdateFormData = $state(false);
    
    // Purpose options
    const purposeOptions = [
        {
            id: "assets",
            label: "Media/Assets",
            description: "Extract images, videos, or other media assets",
        },
        {
            id: "links",
            label: "Links",
            description: "Follow links to new pages for crawling",
        },
        {
            id: "pagination",
            label: "Pagination",
            description: "Navigate through paginated content",
        },
        {
            id: "metadata",
            label: "Metadata",
            description: "Extract metadata like titles, descriptions, etc.",
        },
    ];
    
    onMount(() => {
        resetNewSelector();
        validate();
    });
    
    function handleSelectionChange(event) {
        const selection = event.detail;
        if (selection && selection.elements && selection.elements.length > 0) {
            const cssPath = selection.elements[0].cssPath;
            if (editingIndex >= 0) {
                selectors[editingIndex].value = cssPath;
                selectors[editingIndex].purpose = selection.type || "assets";
                // Set default attribute based on purpose
                selectors[editingIndex].attribute = getDefaultAttributeForPurpose(selection.type || "assets");
            } else {
                newSelector.value = cssPath;
                newSelector.purpose = selection.type || "assets";
                // Set default attribute based on purpose
                newSelector.attribute = getDefaultAttributeForPurpose(selection.type || "assets");
            }
            shouldUpdateFormData = true;
            updateFormData();
        }
    }
    
    // Get default attribute based on purpose
    function getDefaultAttributeForPurpose(purpose) {
        switch(purpose) {
            case "links":
            case "pagination":
                return "href";
            case "assets":
                return "src";
            case "metadata":
                return "text";
            default:
                return "src";
        }
    }
    
    // Set default attribute when purpose changes
    function handlePurposeChange(e) {
        const purpose = e.target.value;
        newSelector.attribute = getDefaultAttributeForPurpose(purpose);
        shouldUpdateFormData = true;
        updateFormData();
    }
    
    // Add or update a selector
    function addSelector() {
        if (!newSelector.name || !newSelector.value) return;
        if (editingIndex >= 0) {
            // Update existing selector
            selectors[editingIndex] = { ...newSelector };
            editingIndex = -1;
        } else {
            // Add new selector
            selectors = [...selectors, { ...newSelector }];
        }
        // Reset form
        resetNewSelector();
        shouldUpdateFormData = true;
        updateFormData();
    }
    
    // Edit a selector
    function editSelector(index) {
        newSelector = { ...selectors[index] };
        editingIndex = index;
        // Switch to form view if in visual mode
        if (currentView === "visual") {
            currentView = "list";
        }
    }
    
    // Remove a selector
    function removeSelector(index) {
        selectors = selectors.filter((_, i) => i !== index);
        if (editingIndex === index) {
            resetNewSelector();
            editingIndex = -1;
        }
        shouldUpdateFormData = true;
        updateFormData();
    }
    
    // Reset the new selector form
    function resetNewSelector() {
        newSelector = {
            id: generateId(),
            name: "",
            type: "css",
            value: "",
            attributeSource: "",
            attribute: "src",
            description: "",
            purpose: "assets",
            priority: 0,
            isOptional: false,
            urlPattern: "",
        };
        editingIndex = -1;
    }
    
    // Generate a random ID
    function generateId() {
        return "sel_" + Math.random().toString(36).substring(2, 11);
    }
    
    // Switch between list and visual views
    function switchView(view) {
        currentView = view;
    }
    
    // Validate the step
    function validate() {
        // Check if we have at least one selector
        const hasLinks = selectors.some((sel) => sel.purpose === "links");
        const hasAssets = selectors.some((sel) => sel.purpose === "assets");
        isValid = selectors.length > 0 && hasLinks && hasAssets;
        dispatch("validate", isValid);
        return isValid;
    }
    
    // Update form data and validate without causing a reactive loop
    function updateFormData() {
        if (!shouldUpdateFormData) return;
        
        const updatedData = {
            ...formData,
            selectors: [...selectors],
        };
        
        const isValid = validate();
        if (isValid) {
            dispatch("update", updatedData);
        }
        
        // Reset the flag to prevent loops
        shouldUpdateFormData = false;
    }

</script>

<div>
    <h2 class="text-xl font-semibold mb-4">Content Selection</h2>
    <p class="text-dark-300 mb-6">
        Define what content to extract with CSS or XPath selectors
    </p>
    <!-- View switcher -->
    <div class="flex border border-dark-700 rounded-lg mb-6 overflow-hidden">
        <button
            class={`flex-1 py-3 px-4 focus:outline-none ${currentView === "list" ? "bg-base-700 text-white" : "bg-base-800 text-dark-300 hover:bg-base-750"}`}
            onclick={() => switchView("list")}
        >
            <svg
                xmlns="http://www.w3.org/2000/svg"
                class="h-5 w-5 inline-block mr-2"
                viewBox="0 0 20 20"
                fill="currentColor"
            >
                <path
                    fill-rule="evenodd"
                    d="M3 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm0 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm0 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm0 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1z"
                    clip-rule="evenodd"
                />
            </svg>
            List View
        </button>
        <button
            class={`flex-1 py-3 px-4 focus:outline-none ${currentView === "visual" ? "bg-base-700 text-white" : "bg-base-800 text-dark-300 hover:bg-base-750"}`}
            onclick={() => switchView("visual")}
        >
            <svg
                xmlns="http://www.w3.org/2000/svg"
                class="h-5 w-5 inline-block mr-2"
                viewBox="0 0 20 20"
                fill="currentColor"
            >
                <path
                    fill-rule="evenodd"
                    d="M3 5a2 2 0 012-2h10a2 2 0 012 2v10a2 2 0 01-2 2H5a2 2 0 01-2-2V5zm11 1H6a1 1 0 00-1 1v6a1 1 0 001 1h8a1 1 0 001-1V7a1 1 0 00-1-1z"
                    clip-rule="evenodd"
                />
            </svg>
            Visual Selector
        </button>
    </div>

    {#if currentView === "list"}
        <!-- Selector list view -->
        <div>
            <!-- Existing selectors -->
            {#if selectors.length > 0}
                <div class="mb-6">
                    <h3 class="text-sm font-medium text-dark-300 mb-3">
                        Current Selectors
                    </h3>
                    <div class="bg-base-800 rounded-lg overflow-hidden">
                        <table class="min-w-full divide-y divide-dark-700">
                            <thead class="bg-base-750">
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
                                        >Selector</th
                                    >
                                    <th
                                        scope="col"
                                        class="px-4 py-3 text-right text-xs font-medium text-dark-300 uppercase tracking-wider"
                                        >Actions</th
                                    >
                                </tr>
                            </thead>
                            <tbody class="divide-y divide-dark-700">
                                {#each selectors as selector, i}
                                    <tr class="hover:bg-base-750">
                                        <td class="px-4 py-3 whitespace-nowrap">
                                            <div class="text-sm font-medium">
                                                {selector.name}
                                            </div>
                                            {#if selector.description}
                                                <div
                                                    class="text-xs text-dark-400"
                                                >
                                                    {selector.description}
                                                </div>
                                            {/if}
                                        </td>
                                        <td class="px-4 py-3 whitespace-nowrap">
                                            <span
                                                class={`px-2 py-1 inline-flex text-xs leading-5 font-semibold rounded-full 
                          ${
                              selector.purpose === "assets"
                                  ? "bg-blue-500 text-blue-100"
                                  : selector.purpose === "links"
                                    ? "bg-green-500 text-green-100"
                                    : selector.purpose === "pagination"
                                      ? "bg-yellow-500 text-yellow-100"
                                      : "bg-purple-500 text-purple-100"
                          }`}
                                            >
                                                {selector.purpose}
                                            </span>
                                        </td>
                                        <td class="px-4 py-3">
                                            <div
                                                class="text-xs font-mono text-dark-300 break-all"
                                            >
                                                {selector.value}
                                            </div>
                                        </td>
                                        <td
                                            class="px-4 py-3 whitespace-nowrap text-right text-sm font-medium"
                                        >
                                            <button
                                                class="text-primary-400 hover:text-primary-300 mr-3"
                                                onclick={() => editSelector(i)}
                                            >
                                                Edit
                                            </button>
                                            <button
                                                class="text-danger-400 hover:text-danger-300"
                                                onclick={() =>
                                                    removeSelector(i)}
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

            <!-- Selector form -->
            <div class="bg-base-800 rounded-lg p-4 mb-6">
                <h3 class="text-sm font-medium mb-4">
                    {editingIndex >= 0 ? "Edit Selector" : "Add New Selector"}
                </h3>
                <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
                    <div>
                        <label
                            for="selector-name"
                            class="block text-sm font-medium text-dark-300 mb-1"
                        >
                            Name <span class="text-danger-500">*</span>
                        </label>
                        <input
                            id="selector-name"
                            type="text"
                            bind:value={newSelector.name}
                            placeholder="E.g., Product Images"
                            class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                        />
                    </div>

                    <div>
                        <label
                            for="selector-purpose"
                            class="block text-sm font-medium text-dark-300 mb-1"
                        >
                            Purpose <span class="text-danger-500">*</span>
                        </label>
                        <select
                            id="selector-purpose"
                            bind:value={newSelector.purpose}
                            class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                        >
                            {#each purposeOptions as option}
                                <option value={option.id}>{option.label}</option
                                >
                            {/each}
                        </select>
                    </div>

                    <div>
                        <label
                            for="selector-type"
                            class="block text-sm font-medium text-dark-300 mb-1"
                        >
                            Selector Type
                        </label>
                        <select
                            id="selector-type"
                            bind:value={newSelector.type}
                            class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                        >
                            <option value="css">CSS Selector</option>
                            <option value="xpath">XPath</option>
                        </select>
                    </div>

                    <div>
                        <label
                            for="selector-attribute"
                            class="block text-sm font-medium text-dark-300 mb-1"
                        >
                            Attribute to Extract
                        </label>
                        <select
                            id="selector-attribute"
                            bind:value={newSelector.attribute}
                            class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                        >
                            <option value="src">src (for images, videos)</option
                            >
                            <option value="href">href (for links)</option>
                            <option value="text">text content</option>
                            <option value="html">HTML content</option>
                            <option value="data-src"
                                >data-src (lazy loading)</option
                            >
                            <option value="alt">alt (image description)</option>
                            <option value="title">title attribute</option>
                        </select>
                    </div>

                    <div class="sm:col-span-2">
                        <label
                            for="selector-value"
                            class="block text-sm font-medium text-dark-300 mb-1"
                        >
                            Selector Value <span class="text-danger-500">*</span
                            >
                        </label>
                        <input
                            id="selector-value"
                            type="text"
                            bind:value={newSelector.value}
                            placeholder="E.g., .product-image img"
                            class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                        />
                    </div>

                    <div class="sm:col-span-2">
                        <label
                            for="selector-url-pattern"
                            class="block text-sm font-medium text-dark-300 mb-1"
                        >
                            URL Pattern (optional)
                        </label>
                        <input
                            id="selector-url-pattern"
                            type="text"
                            bind:value={newSelector.urlPattern}
                            placeholder="E.g., /product/.* (regex)"
                            class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                        />
                        <p class="mt-1 text-xs text-dark-400">
                            Only apply this selector to URLs matching this
                            pattern (regex)
                        </p>
                    </div>

                    <div class="sm:col-span-2">
                        <label
                            for="selector-description"
                            class="block text-sm font-medium text-dark-300 mb-1"
                        >
                            Description
                        </label>
                        <textarea
                            id="selector-description"
                            bind:value={newSelector.description}
                            rows="2"
                            placeholder="Describe what this selector does"
                            class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                        ></textarea>
                    </div>

                    <div class="sm:col-span-2 flex items-center">
                        <input
                            id="selector-optional"
                            type="checkbox"
                            bind:checked={newSelector.isOptional}
                            class="h-4 w-4 text-primary-600 focus:ring-primary-500 border-dark-500 rounded"
                        />
                        <label
                            for="selector-optional"
                            class="ml-2 block text-sm text-dark-300"
                        >
                            Optional (continue even if selector doesn't match)
                        </label>
                    </div>
                </div>

                <div class="mt-4 flex justify-end space-x-3">
                    {#if editingIndex >= 0}
                        <Button variant="outline" onclick={resetNewSelector}>
                            Cancel
                        </Button>
                    {/if}
                    <Button
                        variant="primary"
                        onclick={addSelector}
                        disabled={!newSelector.name || !newSelector.value}
                    >
                        {editingIndex >= 0 ? "Update Selector" : "Add Selector"}
                    </Button>
                </div>
            </div>

            <!-- Help section -->
            <div class="bg-base-850 rounded-lg p-4">
                <h4 class="text-sm font-medium mb-2">Selector Tips</h4>
                <ul class="text-xs text-dark-300 list-disc pl-5 space-y-1">
                    <li>
                        You need at least one <strong>links</strong> selector
                        (to find URLs to crawl) and one <strong>assets</strong> selector
                        (to find content to download)
                    </li>
                    <li>
                        Use Chrome DevTools to help find the right selectors
                        (right-click an element and select "Inspect")
                    </li>
                    <li>
                        For images, use <code
                            class="px-1 py-0.5 rounded bg-base-700 text-xs font-mono"
                            >img</code
                        >
                        with
                        <code
                            class="px-1 py-0.5 rounded bg-base-700 text-xs font-mono"
                            >src</code
                        > attribute
                    </li>
                    <li>
                        For links, use <code
                            class="px-1 py-0.5 rounded bg-base-700 text-xs font-mono"
                            >a</code
                        >
                        with
                        <code
                            class="px-1 py-0.5 rounded bg-base-700 text-xs font-mono"
                            >href</code
                        > attribute
                    </li>
                    <li>
                        Pagination selectors help navigate through multiple
                        pages
                    </li>
                </ul>
            </div>
        </div>
    {:else}
        <!-- Visual selector view -->
        <div class="h-[500px]">
            <VisualSelector
                url={visualUrl}
                onSelectionChange={handleSelectionChange}
            />
        </div>
    {/if}
</div>
