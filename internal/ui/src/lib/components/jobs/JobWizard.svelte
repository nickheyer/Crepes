<script>
  import { onMount } from 'svelte';
  import { createEventDispatcher } from 'svelte';
  import { jobWizardState, updateJobWizardStep, setJobWizardStep, createNewJob, resetJobWizard } from '$lib/stores/jobStore';
  import { addToast } from '$lib/stores/uiStore';
  import { isValidUrl } from '$lib/utils/validation';
  import { CheckCircle, XCircle, ArrowRight, ArrowLeft } from 'lucide-svelte';
  
  // Import all wizard step components
  import BasicInfoStep from './wizard-steps/BasicInfoStep.svelte';
  import ContentSelectionStep from './wizard-steps/ContentSelectionStep.svelte';
  import FilteringStep from './wizard-steps/FilteringStep.svelte';
  import ProcessingStep from './wizard-steps/ProcessingStep.svelte';
  import ScheduleStep from './wizard-steps/ScheduleStep.svelte';
  import SummaryStep from './wizard-steps/SummaryStep.svelte';
  
  // Create dispatch function
  const dispatch = createEventDispatcher();
  
  // Props
  let {
      isTemplate = false,
      initialData = null,
      isEditing = false
  } = $props();
  
  // Steps configuration
  const steps = [
    { id: 1, name: 'Basic Info', component: BasicInfoStep },
    { id: 2, name: 'Content Selection', component: ContentSelectionStep },
    { id: 3, name: 'Filtering', component: FilteringStep },
    { id: 4, name: 'Processing', component: ProcessingStep },
    { id: 5, name: 'Schedule', component: ScheduleStep },
    { id: 6, name: 'Summary', component: SummaryStep }
  ];
  
  // Get current state from store
  let currentStep = $derived($jobWizardState.step);
  let formData = $derived($jobWizardState.data);
  
  // Validation state
  let stepValid = $state(true); // Default to true to avoid blocking navigation
  
  // Initialize wizard with provided data (if editing)
  onMount(() => {
    if (initialData) {
      // Populate wizard with initialData
      updateJobWizardStep(1, initialData);
    }
  });
  
  // Navigation functions
  function goToStep(step) {
    if (step < 1 || step > steps.length) return;
    // Only allow navigation to already visited or next step
    if (step <= currentStep + 1) {
      setJobWizardStep(step);
    }
  }
  
  function goToNextStep() {
    if (currentStep < steps.length) {
      setJobWizardStep(currentStep + 1);
    }
  }
  
  function goToPrevStep() {
    if (currentStep > 1) {
      setJobWizardStep(currentStep - 1);
    }
  }
  
  // Submit handler
  async function handleSubmit() {
    try {
      // Create or update job based on the formData
      const result = await createNewJob(formData);
      // Show success notification
      addToast(isEditing ? 'Job updated successfully' : 'Job created successfully', 'success');
      // Reset wizard
      resetJobWizard();
      // Emit success event
      dispatch('success', { job: result });
    } catch (error) {
      addToast(`Failed to ${isEditing ? 'update' : 'create'} job: ${error.message}`, 'error');
    }
  }
  
  // Update stepValid based on current step validation
  function updateStepValidity(isValid) {
    // Use setTimeout to break potential reactive loop
    setTimeout(() => {
      stepValid = isValid;
    }, 0);
  }
  
  // Cancel handler
  function handleCancel() {
    resetJobWizard();
    dispatch('cancel');
  }
  
  // Determine if submit button should be shown
  let isLastStep = $derived(currentStep === steps.length);
</script>

<div class="card bg-base-100 shadow-xl">
  <!-- Wizard header with steps -->
  <div class="card-body p-4 border-b border-base-300">
    <div class="max-w-7xl mx-auto">
      <!-- Step indicators -->
      <ul class="steps steps-horizontal w-full">
        {#each steps as step, idx}
          <li 
            class="step cursor-pointer {currentStep > step.id ? 'step-primary' : ''} {currentStep === step.id ? 'step-primary' : ''}"
            data-content={currentStep > step.id ? '✓' : step.id}
            onclick={() => goToStep(step.id)}
            onkeydown={(e) => e.key === "Escape" && closeMenu()}
            transition:fade={{ duration: 200 }}
            role="tab"
            aria-label="stepcursor"
            tabindex="0"
          >
            {step.name}
          </li>
        {/each}
      </ul>
    </div>
  </div>
  
  <!-- Wizard content -->
  <div class="card-body">
    <div class="max-w-3xl mx-auto">
      <!-- Step content -->
      <div class="py-4">
        {#if currentStep === 1}
          <BasicInfoStep
            {formData}
            on:update={(e) => updateJobWizardStep(1, e.detail)}
            on:validate={(e) => updateStepValidity(e.detail)}
          />
        {:else if currentStep === 2}
          <ContentSelectionStep
            {formData}
            on:update={(e) => updateJobWizardStep(2, e.detail)}
            on:validate={(e) => updateStepValidity(e.detail)}
          />
        {:else if currentStep === 3}
          <FilteringStep
            {formData}
            on:update={(e) => updateJobWizardStep(3, e.detail)}
            on:validate={(e) => updateStepValidity(e.detail)}
          />
        {:else if currentStep === 4}
          <ProcessingStep
            {formData}
            on:update={(e) => updateJobWizardStep(4, e.detail)}
            on:validate={(e) => updateStepValidity(e.detail)}
          />
        {:else if currentStep === 5}
          <ScheduleStep
            {formData}
            on:update={(e) => updateJobWizardStep(5, e.detail)}
            on:validate={(e) => updateStepValidity(e.detail)}
          />
        {:else if currentStep === 6}
          <SummaryStep
            {formData}
            on:update={(e) => updateJobWizardStep(6, e.detail)}
            on:validate={(e) => updateStepValidity(e.detail)}
          />
        {/if}
      </div>
      
      <!-- Navigation buttons -->
      <div class="divider"></div>
      <div class="flex justify-between">
        <div>
          {#if currentStep > 1}
            <button class="btn btn-outline" onclick={goToPrevStep}>
              <ArrowLeft class="h-4 w-4 mr-1" />
              Back
            </button>
          {/if}
        </div>
        <div class="space-x-2">
          <button class="btn" onclick={handleCancel}>
            Cancel
          </button>
          {#if isLastStep}
            <button 
              class="btn btn-primary" 
              onclick={handleSubmit} 
              disabled={!stepValid}
            >
              {isEditing ? 'Update Job' : 'Create Job'}
            </button>
          {:else}
            <button 
              class="btn btn-primary" 
              onclick={goToNextStep}
              disabled={!stepValid}
            >
              Next
              <ArrowRight class="h-4 w-4 ml-1" />
            </button>
          {/if}
        </div>
      </div>
    </div>
  </div>
</div>
