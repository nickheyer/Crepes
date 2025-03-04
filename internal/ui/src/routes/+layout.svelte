<script>
  import { createToastSystem } from '$lib/components';

  const TOASTS = createToastSystem();
  let toastQueue = $state(TOASTS.toastQueue);
  let { children } = $props();
  
  $effect(() => {
    if (typeof window !== 'undefined') {
      window.showToast = TOASTS.showToast;
    }
  });
</script>

<div class="bg-gray-900 text-white min-h-screen flex flex-col">
  <main class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8 py-6 w-full">
    {@render children()}
  </main>
  
  <!-- TOAST NOTIFICATIONS -->
  <div class="fixed bottom-4 right-4 z-50 flex flex-col gap-2">
    {#each toastQueue as toast (toast.id)}
      <div 
        class="px-4 py-2 rounded-md shadow-lg text-white flex items-center gap-2 animate-fade-in" 
        style:background-color={toast.type === 'success' ? '#10B981' : toast.type === 'error' ? '#EF4444' : '#3B82F6'}
        role="alert"
        aria-live="assertive"
      >
        <!-- TOAST ICON -->
        {#if toast.type === 'success'}
          <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
            <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd" />
          </svg>
        {:else if toast.type === 'error'}
          <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
            <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd" />
          </svg>
        {:else}
          <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
            <path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2h-1V9a1 1 0 00-1-1z" clip-rule="evenodd" />
          </svg>
        {/if}
        <span>{toast.message}</span>
      </div>
    {/each}
  </div>
</div>

<style>
  :global(body) {
    margin: 0;
    padding: 0;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen,
      Ubuntu, Cantarell, 'Open Sans', 'Helvetica Neue', sans-serif;
    background-color: #111827;
    color: white;
  }
  
  @keyframes fadeIn {
    from { opacity: 0; transform: translateY(10px); }
    to { opacity: 1; transform: translateY(0); }
  }
  
  .animate-fade-in {
    animation: fadeIn 0.3s ease-out;
  }
</style>
