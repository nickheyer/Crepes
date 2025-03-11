<script>
  import { onMount } from 'svelte';
  import { state as jobState, setJobWizardStep, createNewJob, resetJobWizard, updateJobWizardStep } from '$lib/stores/jobStore.svelte';
  import { addToast } from '$lib/stores/uiStore.svelte';
  
  // IMPORT ALL WIZARD STEP COMPONENTS
  import BasicInfoStep from './wizard-steps/BasicInfoStep.svelte';
  import ContentSelectionStep from './wizard-steps/ContentSelectionStep.svelte';
  import FilteringStep from './wizard-steps/FilteringStep.svelte';
  import ProcessingStep from './wizard-steps/ProcessingStep.svelte';
  import ScheduleStep from './wizard-steps/ScheduleStep.svelte';
  import SummaryStep from './wizard-steps/SummaryStep.svelte';
  
  // PROPS
  let {
    isTemplate = false,
    initialData = null,
    isEditing = false,
    onSuccess = () => {},
    onCancel = () => {}
  } = $props();
  
  // STEPS CONFIGURATION
  const steps = [
    { id: 1, name: 'Basic Info', component: BasicInfoStep },
    { id: 2, name: 'Content Selection', component: ContentSelectionStep },
    { id: 3, name: 'Filtering', component: FilteringStep },
    { id: 4, name: 'Processing', component: ProcessingStep },
    { id: 5, name: 'Schedule', component: ScheduleStep },
    { id: 6, name: 'Summary', component: SummaryStep }
  ];
  
  // GET CURRENT STATE FROM STORE
  let currentStep = $derived(jobState.formData.step);
  let formData = $derived(jobState.formData.data);
  
  // VALIDATION STATE
  let stepValid = $state(true);
  let submitting = $state(false);
  
  // INITIALIZE WIZARD WITH PROVIDED DATA
  onMount(() => {
    if (initialData) {
      // POPULATE WIZARD WITH INITIALDATA
      updateJobWizardStep(1, initialData);
    }
  });
  
  // ENSURE BASE URL IS PASSED TO VISUAL SELECTOR
  $effect(() => {
    // WHEN MOVING TO CONTENT SELECTION TAB, MAKE SURE BASE URL IS AVAILABLE
    if (currentStep === 2 && formData.baseUrl) {
      // THE VISUAL SELECTOR WILL USE THIS URL
      console.log("Base URL available for content selection:", formData.baseUrl);
    }
  });
  
  // NAVIGATION FUNCTIONS
  function goToStep(step) {
    if (step < 1 || step > steps.length) return;
    
    // ONLY ALLOW NAVIGATION TO ALREADY VISITED OR NEXT STEP
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
  
  // SUBMIT HANDLER
  async function handleSubmit() {
    if (submitting) return;
    
    try {
      submitting = true;
      // CREATE OR UPDATE JOB BASED ON THE FORMDATA
      const result = await createNewJob(formData);
      
      // SHOW SUCCESS NOTIFICATION
      addToast(isEditing ? 'JOB UPDATED SUCCESSFULLY' : 'JOB CREATED SUCCESSFULLY', 'success');
      
      // RESET WIZARD
      resetJobWizard();
      
      // CALL SUCCESS CALLBACK
      onSuccess(result);
    } catch (error) {
      addToast(`FAILED TO ${isEditing ? 'UPDATE' : 'CREATE'} JOB: ${error.message}`, 'error');
    } finally {
      submitting = false;
    }
  }
  
  // UPDATE STEPVALID BASED ON FORM VALIDATION
  function updateStepValidity(isValid) {
    stepValid = isValid;
  }
  
  // CANCEL HANDLER
  function handleCancel() {
    resetJobWizard();
    onCancel();
  }
  
  // DETERMINE IF SUBMIT BUTTON SHOULD BE SHOWN
  let isLastStep = $derived(currentStep === steps.length);
</script>

<div class="card bg-base-100 shadow-xl">
  <!-- WIZARD HEADER WITH STEPS -->
  <div class="card-body p-4 border-b border-base-300">
    <div class="max-w-7xl mx-auto">
      <!-- STEP INDICATORS -->
      <ul class="steps steps-horizontal w-full">
        {#each steps as step, idx}
          <li 
            class="step {currentStep > step.id ? 'step-primary' : ''} {currentStep === step.id ? 'step-primary' : ''}"
            data-content={currentStep > step.id ? '✓' : step.id}
          >
            <button 
              class="text-inherit bg-transparent border-none p-0 cursor-pointer focus:outline-none"
              onclick={() => goToStep(step.id)}
              style="font-size: inherit; font-weight: inherit;"
            >
              {step.name}
            </button>
          </li>
        {/each}
      </ul>
    </div>
  </div>
  
  <!-- WIZARD CONTENT -->
  <div class="card-body">
    <div class="max-w-3xl mx-auto">
      <!-- STEP CONTENT -->
      <div class="py-4">
        {#if currentStep === 1}
          <BasicInfoStep />
        {:else if currentStep === 2}
          <ContentSelectionStep />
        {:else if currentStep === 3}
          <FilteringStep />
        {:else if currentStep === 4}
          <ProcessingStep />
        {:else if currentStep === 5}
          <ScheduleStep />
        {:else if currentStep === 6}
          <SummaryStep />
        {/if}
      </div>
      
      <!-- NAVIGATION BUTTONS -->
      <div class="divider"></div>
      <div class="flex justify-between">
        <div>
          {#if currentStep > 1}
            <button class="btn btn-outline" onclick={goToPrevStep}>
              <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 mr-1" viewBox="0 0 20 20" fill="currentColor">
                <path fill-rule="evenodd" d="M9.707 16.707a1 1 0 01-1.414 0l-6-6a1 1 0 010-1.414l6-6a1 1 0 011.414 1.414L5.414 9H17a1 1 0 110 2H5.414l4.293 4.293a1 1 0 010 1.414z" clip-rule="evenodd" />
              </svg>
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
              class="btn btn-primary {submitting ? 'loading' : ''}" 
              onclick={handleSubmit} 
              disabled={!stepValid || submitting}
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
              <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 ml-1" viewBox="0 0 20 20" fill="currentColor">
                <path fill-rule="evenodd" d="M10.293 3.293a1 1 0 011.414 0l6 6a1 1 0 010 1.414l-6 6a1 1 0 01-1.414-1.414L14.586 11H3a1 1 0 110-2h11.586l-4.293-4.293a1 1 0 010-1.414z" clip-rule="evenodd" />
              </svg>
            </button>
          {/if}
        </div>
      </div>
    </div>
  </div>
</div>
