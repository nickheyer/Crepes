<script>
    import { onMount, onDestroy } from "svelte";
    import { createEventDispatcher } from "svelte";
    import { isValidUrl } from "$lib/utils/validation";
    import { addToast } from "$lib/stores/uiStore";
    
    // CREATE DISPATCH FUNCTION
    const dispatch = createEventDispatcher();
    
    // PROPS
    let {
        url = '',
        onSelectionChange = null
    } = $props();
    
    // INTERNAL STATE
    let iframeLoaded = $state(false);
    let iframeError = $state(false);
    let loading = $state(true);
    let iframe;
    let selectionMode = $state("select"); // 'select' or 'inspect'
    let hoveredElement = $state(null);
    let selectedElements = $state([]);
    let cssPath = $state("");
    
    // SELECTION TYPES 
    const selectionTypes = [
        { id: "assets", label: "Assets (images, videos)" },
        { id: "links", label: "Links (URLs to follow)" },
        { id: "pagination", label: "Pagination" },
        { id: "metadata", label: "Metadata" }
    ];
    let selectedType = $state("assets");

    // TRACK SELECTED ELEMENT DATA SEPARATELY FROM DOM REFERENCES
    // This helps when elements can't be directly referenced
    let selectedElementsData = $state([]);
    
    // MESSAGE LISTENER
    function handleIframeMessage(event) {
        // HANDLE MESSAGES FROM IFRAME
        if (event.data && event.data.type === 'IFRAME_LOADED') {
            loading = false;
            iframeLoaded = true;
            setupIframeInteraction();
        }
    }
    
    onMount(() => {
        // ADD MESSAGE LISTENER FOR IFRAME COMMUNICATION
        window.addEventListener('message', handleIframeMessage);
        
        if (url && isValidUrl(url)) {
            loadIframe();
        }
    });
    
    onDestroy(() => {
        // REMOVE MESSAGE LISTENER
        window.removeEventListener('message', handleIframeMessage);
    });
    
    function loadIframe() {
        if (!isValidUrl(url)) {
            loading = false;
            iframeError = true;
            addToast("Please enter a valid URL", "error");
            return;
        }
        
        loading = true;
        iframeError = false;
        iframeLoaded = false;
        selectedElements = [];
        selectedElementsData = [];
        cssPath = "";
        
        if (iframe) {
            // USE OUR BACKEND PROXY TO BYPASS CORS
            iframe.src = `/api/proxy?url=${encodeURIComponent(url)}`;
            
            iframe.onload = () => {
                // WAIT FOR IFRAME TO FULLY LOAD
                setTimeout(() => {
                    if (!iframeLoaded) {
                        try {
                            setupIframeInteraction();
                            loading = false;
                            iframeLoaded = true;
                        } catch (error) {
                            console.error("Error setting up iframe:", error);
                            if (error.message && error.message.includes('cross-origin')) {
                                iframeError = true;
                                addToast("Cannot access page content due to security restrictions", "error");
                            }
                        }
                    }
                }, 1000);
            };
            
            iframe.onerror = () => {
                loading = false;
                iframeError = true;
                addToast("Failed to load the webpage", "error");
            };
        }
    }
    
    function setupIframeInteraction() {
        try {
            // ACCESS IFRAME CONTENT
            const iframeDocument = iframe.contentDocument || iframe.contentWindow.document;
            
            // INJECT CSS FOR ELEMENT HIGHLIGHTING
            const styleEl = document.createElement('style');
            styleEl.textContent = `
                .selector-hover {
                    outline: 2px solid rgba(59, 130, 246, 0.7) !important;
                    background-color: rgba(59, 130, 246, 0.1) !important;
                    cursor: pointer !important;
                }
                .selector-selected {
                    outline: 2px solid rgba(16, 185, 129, 0.7) !important;
                    background-color: rgba(16, 185, 129, 0.2) !important;
                }
                
                /* PREVENT LAYOUT SHIFTS */
                body, html {
                    overflow: auto !important;
                    height: auto !important;
                    position: relative !important;
                }
                
                /* ENSURE IFRAME CONTENT DOESN'T AFFECT PARENT SIZE */
                body {
                    min-height: 100vh;
                }
            `;
            iframeDocument.head.appendChild(styleEl);
            
            // DISABLE ALL IFRAME EVENT HANDLERS THAT MIGHT INTERFERE
            const disableOriginalEvents = document.createElement('script');
            disableOriginalEvents.textContent = `
                (function() {
                    // PREVENT DEFAULT BEHAVIORS
                    document.addEventListener('click', function(e) {
                        e.stopPropagation();
                        return false;
                    }, true);
                    
                    // DISABLE MOUSEOVER/MOUSEOUT HANDLERS
                    const originalAddEventListener = EventTarget.prototype.addEventListener;
                    EventTarget.prototype.addEventListener = function(type, listener, options) {
                        if (type === 'mouseover' || type === 'mouseout' || type === 'click') {
                            // Don't add these event listeners from the original page
                            return;
                        }
                        return originalAddEventListener.call(this, type, listener, options);
                    };
                    
                    // DISABLE ALL EXISTING HANDLERS
                    const elements = document.querySelectorAll('*');
                    for (let i = 0; i < elements.length; i++) {
                        elements[i].onclick = null;
                        elements[i].onmouseover = null;
                        elements[i].onmouseout = null;
                    }
                })();
            `;
            iframeDocument.head.appendChild(disableOriginalEvents);
            
            // ADD OUR MOUSEOVER AND CLICK HANDLERS
            iframeDocument.removeEventListener('mouseover', handleMouseOver);
            iframeDocument.removeEventListener('mouseout', handleMouseOut);
            iframeDocument.removeEventListener('click', handleClick, true);
            
            iframeDocument.addEventListener('mouseover', handleMouseOver, true);
            iframeDocument.addEventListener('mouseout', handleMouseOut, true);
            iframeDocument.addEventListener('click', handleClick, true);
            
            iframeLoaded = true;
        } catch (e) {
            console.error("Access issue detected:", e);
            loading = false;
            iframeError = true;
            addToast("Cannot access page content. Using demo mode instead.", "warning");
        }
    }
    
    function handleMouseOver(event) {
        if (selectionMode !== "select" || !event.target) return;
        
        // CLEAR PREVIOUS HOVER
        if (hoveredElement) {
            try {
                hoveredElement.classList.remove("selector-hover");
            } catch (e) {
                // ELEMENT MIGHT BE DETACHED
            }
        }
        
        // SET NEW HOVER
        event.target.classList.add("selector-hover");
        hoveredElement = event.target;
        
        // GENERATE CSS PATH FOR HOVERED ELEMENT
        cssPath = generateCssPath(event.target);
        
        // PREVENT DEFAULT AND STOP PROPAGATION
        event.preventDefault();
        event.stopPropagation();
        return false;
    }
    
    function handleMouseOut(event) {
        if (selectionMode !== "select" || !event.target) return;
        
        // CLEAR HOVER
        event.target.classList.remove("selector-hover");
        if (hoveredElement === event.target) {
            hoveredElement = null;
            cssPath = "";
        }
        
        // PREVENT DEFAULT AND STOP PROPAGATION
        event.preventDefault();
        event.stopPropagation();
        return false;
    }
    
    function handleClick(event) {
        if (selectionMode !== "select" || !event.target) return;
        
        // ALWAYS PREVENT DEFAULT AND STOP PROPAGATION
        event.preventDefault();
        event.stopPropagation();
        
        const target = event.target;
        
        // CHECK IF ELEMENT IS ALREADY SELECTED
        const isSelected = target.classList.contains("selector-selected");
        
        if (isSelected) {
            // UNSELECT ELEMENT
            target.classList.remove("selector-selected");
            selectedElements = selectedElements.filter(el => el !== target);
            
            // REMOVE FROM DATA ARRAY
            const path = generateCssPath(target);
            selectedElementsData = selectedElementsData.filter(data => data.cssPath !== path);
        } else {
            // SELECT ELEMENT
            target.classList.add("selector-selected");
            selectedElements = [...selectedElements, target];
            
            // STORE ELEMENT DATA
            const elementData = {
                cssPath: generateCssPath(target),
                tagName: target.tagName.toLowerCase(),
                text: target.textContent?.trim().substring(0, 100) || ""
            };
            selectedElementsData = [...selectedElementsData, elementData];
        }
        
        // NOTIFY ABOUT SELECTION CHANGE
        updateSelections();
        
        return false;
    }
    
    function updateSelections() {
        // SEND SELECTION DATA TO PARENT
        const selectionData = {
            type: selectedType,
            elements: selectedElementsData
        };
        
        if (onSelectionChange) {
            onSelectionChange({ detail: selectionData });
        }
        
        dispatch("selectionChange", selectionData);
    }
    
    // GENERATE CSS SELECTOR PATH
    function generateCssPath(element) {
        if (!element) return '';
        if (element.tagName.toLowerCase() === 'html') return 'html';
        if (element.tagName.toLowerCase() === 'body') return 'body';
        
        let path = [];
        let current = element;
        
        while (current && current.nodeType === Node.ELEMENT_NODE) {
            let selector = current.tagName.toLowerCase();
            
            // USE ID IF AVAILABLE
            if (current.id) {
                selector += `#${current.id}`;
                path.unshift(selector);
                break;
            }
            
            // USE CLASSES
            const classes = Array.from(current.classList)
                .filter(c => !c.startsWith('selector-'))
                .join('.');
                
            if (classes) {
                selector += `.${classes}`;
            }
            
            // IF THERE ARE SIBLINGS, ADD POSITION
            let siblingCount = 0;
            let siblingIndex = 0;
            let sibling = current;
            
            while (sibling) {
                if (sibling.tagName === current.tagName) {
                    siblingCount++;
                    if (sibling === current) {
                        siblingIndex = siblingCount;
                    }
                }
                sibling = sibling.previousElementSibling;
            }
            
            if (siblingCount > 1) {
                selector += `:nth-of-type(${siblingIndex})`;
            }
            
            path.unshift(selector);
            current = current.parentElement;
        }
        
        return path.join(' > ');
    }
    
    // HANDLE SELECTION TYPE CHANGES
    function handleTypeChange() {
        selectedType = selectedType;
        updateSelections();
    }
    
    // CLEAR SELECTIONS
    function clearSelections() {
        if (selectedElements.length > 0) {
            selectedElements.forEach(el => {
                try {
                    el.classList.remove("selector-selected");
                } catch (e) {
                    // ELEMENT MIGHT HAVE BEEN REMOVED FROM DOM
                }
            });
            selectedElements = [];
            selectedElementsData = [];
            cssPath = "";
            
            dispatch("selectionChange", { type: selectedType, elements: [] });
            addToast("Selections cleared", "info");
        }
    }
    
    // DEMO MODE FUNCTIONS (WHEN PROXY FAILS)
    function useExampleSelector(selectorType, selectorValue) {
        selectedType = selectorType;
        
        const demoSelection = {
            type: selectorType,
            elements: [{
                cssPath: selectorValue,
                tagName: selectorValue.split(' ').pop().split('.')[0].split('#')[0],
                text: "Example element"
            }]
        };
        
        if (onSelectionChange) {
            onSelectionChange({ detail: demoSelection });
        }
        
        dispatch("selectionChange", demoSelection);
        
        addToast(`Selected: ${selectorValue}`, "success");
    }
</script>

<div class="flex flex-col h-full">
    <div class="mockup-browser border border-base-300 w-full bg-base-300">
        <div class="mockup-browser-toolbar">
            <div class="input flex-1 join">
                <input
                    type="url"
                    bind:value={url}
                    placeholder="https://example.com"
                    class="join-item input input-bordered w-full"
                />
                <button class="btn btn-primary join-item" onclick={loadIframe}>
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 mr-1" viewBox="0 0 20 20" fill="currentColor">
                        <path fill-rule="evenodd" d="M8 4a4 4 0 100 8 4 4 0 000-8zM2 8a6 6 0 1110.89 3.476l4.817 4.817a1 1 0 01-1.414 1.414l-4.816-4.816A6 6 0 012 8z" clip-rule="evenodd" />
                    </svg>
                    Load
                </button>
            </div>
        </div>

        <!-- SELECTOR CONTROLS -->
        <div class="bg-base-200 p-2 border-t border-base-300 flex flex-col sm:flex-row justify-between items-start sm:items-center gap-2">
            <div class="flex items-center gap-2">
                <select 
                    bind:value={selectedType}
                    onchange={handleTypeChange}
                    class="select select-sm select-bordered"
                >
                    {#each selectionTypes as type}
                        <option value={type.id}>{type.label}</option>
                    {/each}
                </select>
                
                <button class="btn btn-sm btn-outline" onclick={clearSelections}>
                    Clear
                </button>
            </div>
            
            {#if cssPath}
                <div class="flex-1 px-2 py-1 bg-base-300 rounded text-xs font-mono overflow-x-auto whitespace-nowrap max-w-full">
                    {cssPath}
                </div>
            {/if}
        </div>
        
        <!-- IFRAME CONTAINER WITH FIXED HEIGHT -->
        <div class="mockup-browser-content bg-base-200 relative" style="height: 500px; overflow: hidden;">
            {#if loading}
                <div class="absolute inset-0 flex items-center justify-center bg-base-100 bg-opacity-75 z-10">
                    <span class="loading loading-spinner loading-lg text-primary"></span>
                </div>
            {/if}
            
            <iframe
                bind:this={iframe}
                title="Web Page Preview"
                class="w-full h-full border-0"
                sandbox="allow-same-origin allow-scripts"
                style="background: white; height: 100%; overflow: auto;"
            ></iframe>
            
            {#if iframeError && !loading}
                <div class="absolute inset-0 flex flex-col items-center justify-center bg-base-100">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-16 w-16 text-error mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                    </svg>
                    <h3 class="text-lg font-medium mb-2">Unable to load page content</h3>
                    <p class="text-center max-w-md mb-4">
                        The page content could not be loaded. This may be due to browser security restrictions or the site blocking our proxy.
                    </p>
                    
                    <div class="card bg-base-200 w-full max-w-md">
                        <div class="card-body">
                            <h3 class="card-title text-sm">Demo Selectors</h3>
                            <div class="grid grid-cols-1 sm:grid-cols-2 gap-2">
                                <button class="btn btn-sm btn-outline" onclick={() => useExampleSelector("assets", "img.product-image")}>
                                    img.product-image
                                </button>
                                <button class="btn btn-sm btn-outline" onclick={() => useExampleSelector("links", "a.product-link")}>
                                    a.product-link
                                </button>
                                <button class="btn btn-sm btn-outline" onclick={() => useExampleSelector("pagination", "a.pagination-next")}>
                                    a.pagination-next
                                </button>
                                <button class="btn btn-sm btn-outline" onclick={() => useExampleSelector("metadata", "h1.product-title")}>
                                    h1.product-title
                                </button>
                            </div>
                        </div>
                    </div>
                </div>
            {/if}
        </div>
    </div>
    
    <!-- SELECTED ELEMENTS DISPLAY -->
    {#if selectedElementsData.length > 0}
        <div class="card bg-base-200 shadow-xl mt-4">
            <div class="card-body">
                <h3 class="card-title">Selected Elements ({selectedElementsData.length})</h3>
                <div class="overflow-x-auto">
                    <table class="table table-zebra w-full">
                        <thead>
                            <tr>
                                <th>Element</th>
                                <th>Selector</th>
                                <th>Text</th>
                                <th>Action</th>
                            </tr>
                        </thead>
                        <tbody>
                            {#each selectedElementsData as data, i}
                                <tr>
                                    <td>{data.tagName}</td>
                                    <td class="font-mono text-xs">{data.cssPath}</td>
                                    <td class="truncate max-w-xs">{data.text || "(no text)"}</td>
                                    <td>
                                        <button 
                                            class="btn btn-sm btn-ghost text-error"
                                            onclick={() => {
                                                // Find and unselect the element if possible
                                                try {
                                                    if (iframe?.contentDocument) {
                                                        const element = iframe.contentDocument.querySelector(data.cssPath);
                                                        if (element) {
                                                            element.classList.remove("selector-selected");
                                                            selectedElements = selectedElements.filter(el => el !== element);
                                                        }
                                                    }
                                                } catch (e) {
                                                    // Continue with removal even if element not found
                                                }
                                                
                                                // Remove from data array
                                                selectedElementsData = selectedElementsData.filter((_, index) => index !== i);
                                                updateSelections();
                                            }}
                                        >
                                            Remove
                                        </button>
                                    </td>
                                </tr>
                            {/each}
                        </tbody>
                    </table>
                </div>
            </div>
        </div>
    {/if}
</div>
