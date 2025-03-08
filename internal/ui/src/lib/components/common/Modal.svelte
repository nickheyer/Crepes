<script>
    import { fade, fly } from 'svelte/transition';
    import { X } from "lucide-svelte";
    import Button from './Button.svelte';
    
    let {
        title = '',
        size = 'md', // 'sm', 'md', 'lg', 'xl', 'full'
        showClose = true,
        closeOnOverlayClick = true,
        showFooter = true,
        primaryAction = 'Save',
        secondaryAction = 'Cancel',
        primaryVariant = 'primary',
        disabled = false,
        loading = false,
        children
    } = $props();
    
    // Get size class based on size prop
    const sizeClasses = {
      sm: 'max-w-md',
      md: 'max-w-lg',
      lg: 'max-w-2xl',
      xl: 'max-w-4xl',
      full: 'max-w-full mx-4',
    };
    
    const modalSizeClass = sizeClasses[size] || sizeClasses.md;
    
    // Event functions
    function closeModal() {
      dispatch('close');
    }
    
    function handleOverlayClick() {
      if (closeOnOverlayClick) {
        closeModal();
      }
    }
    
    function handlePrimaryAction() {
      dispatch('primaryAction');
    }
    
    function handleSecondaryAction() {
      dispatch('secondaryAction');
      closeModal();
    }
    
    function handleKeydown(event) {
      if (event.key === 'Escape') {
        closeModal();
      }
    }
  </script>
  
  <svelte:window on:keydown={handleKeydown} />
  
  <div
    class="fixed inset-0 z-50 overflow-y-auto"
    aria-labelledby="modal-title"
    role="dialog"
    aria-modal="true"
  >
    <!-- Background overlay -->
    <div
        class="fixed inset-0 bg-black bg-opacity-75 transition-opacity"
        transition:fade={{ duration: 200 }}
        onclick={handleOverlayClick}
        onkeydown={handleOverlayKey}
        role="button"
        aria-label="Close modal"
        tabindex="0"
    ></div>
    
    <!-- Modal content -->
    <div class="flex min-h-screen items-center justify-center p-4 text-center sm:p-0">
      <div
        class="{modalSizeClass} w-full relative bg-base-800 rounded-lg shadow-xl text-left overflow-hidden transform transition-all"
        transition:fly={{ y: 20, duration: 200 }}
      >
        {#if title || showClose}
          <div class="flex justify-between items-center px-6 py-4 border-b border-dark-700">
            {#if title}
              <h3 class="text-lg font-medium text-white" id="modal-title">{title}</h3>
            {:else}
              <div></div>
            {/if}
            
            {#if showClose}
              <button
                type="button"
                class="text-dark-400 hover:text-white focus:outline-none"
                onclick={closeModal}
              >
                <span class="sr-only">Close</span>
                <X class="h-5 w-5" />
              </button>
            {/if}
          </div>
        {/if}
        
        <!-- Modal body -->
        <div class="p-6">
          {@render children()}
        </div>
        
        {#if showFooter}
          <div class="px-6 py-4 bg-base-850 border-t border-dark-700 flex justify-end space-x-3">
            <Button
              variant="outline"
              onclick={handleSecondaryAction}
            >
              {secondaryAction}
            </Button>
            
            <Button
              variant={primaryVariant}
              {disabled}
              loading={loading}
              onclick={handlePrimaryAction}
            >
              {primaryAction}
            </Button>
          </div>
        {/if}
      </div>
    </div>
  </div>
