<script>
    // PROPS USING RUNES
    let {
      isOpen = false,
      title = 'Confirm Action',
      message = 'Are you sure you want to proceed?',
      confirmText = 'Confirm',
      cancelText = 'Cancel',
      confirmVariant = 'primary', // primary, success, warning, danger
      onConfirm = () => {},
      onCancel = () => {}
    } = $props();
  </script>
  
  {#if isOpen}
    <div class="modal modal-open z-500">
      <div class="modal-box">
        <h3 class="font-bold text-lg">{title}</h3>
        <p class="py-4">{message}</p>
        <div class="modal-action">
          <button 
            class="btn" 
            onclick={onCancel}
            aria-label={cancelText}
          >
            {cancelText}
          </button>
          <button 
            class="btn btn-{confirmVariant}" 
            onclick={onConfirm}
            aria-label={confirmText}
          >
            {confirmText}
          </button>
        </div>
      </div>
      <div 
        class="modal-backdrop bg-black opacity-30"
        onclick={onCancel}
        onkeydown={() => {}}
        role="button"
        aria-label="Close modal"
        tabindex="-1"
      ></div>
    </div>
  {/if}
