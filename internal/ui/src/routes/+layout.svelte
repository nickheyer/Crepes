<script>
  import "../app.css";
  import { onMount } from "svelte";
  import Sidebar from "$lib/components/common/Sidebar.svelte";
  import Header from "$lib/components/common/Header.svelte";
  import Toast from "$lib/components/common/Toast.svelte";
  import { state as uiState, initTheme } from "$lib/stores/uiStore.svelte";
  
  let currentPage = $state(getPageFromUrl());
  let { children } = $props();
  
  function getPageFromUrl() {
    if (typeof window === "undefined") return "dashboard";
    const path = window.location.pathname;
    if (path === "/") return "dashboard";
    return path.split("/")[1] || "dashboard";
  }

  $effect.pre(() => {
    initTheme();
  });

  $effect.root(() => {
    currentPage = getPageFromUrl();
  });
  
  // Also listen for URL changes directly
  onMount(() => {
    const updateCurrentPage = () => {
      currentPage = getPageFromUrl();
    };
    
    window.addEventListener("popstate", updateCurrentPage);
    
    // Set up a MutationObserver to detect URL changes via pushState/replaceState
    if (typeof MutationObserver !== 'undefined' && document.querySelector("head > title")) {
      const observer = new MutationObserver(updateCurrentPage);
      observer.observe(document.querySelector("head > title"), { subtree: true, childList: true });
      
      return () => {
        window.removeEventListener("popstate", updateCurrentPage);
        observer.disconnect();
      };
    }
    
    return () => {
      window.removeEventListener("popstate", updateCurrentPage);
    };
  });
</script>

<div id="theme-wrapper" data-theme={uiState.currentTheme}>
  <div class="drawer lg:drawer-open min-h-screen bg-base-300">
    <input id="main-drawer" type="checkbox" class="drawer-toggle" />
    <div class="drawer-content flex flex-col">
      <Header {currentPage} />
      <main class="flex-1 overflow-y-auto p-4 md:p-6">
        <div class="max-w-7xl mx-auto">
          {@render children()}
        </div>
      </main>
    </div>
    <div class="drawer-side">
      <Sidebar {currentPage} />
    </div>
  </div>
  
  {#if uiState.toasts.length > 0}
    <div class="toast toast-end toast-bottom z-600">
      {#each uiState.toasts as toast (toast.id)}
        <Toast 
          type={toast.type} 
          message={toast.message} 
          id={toast.id} 
          duration={toast.duration} 
        />
      {/each}
    </div>
  {/if}
</div>
