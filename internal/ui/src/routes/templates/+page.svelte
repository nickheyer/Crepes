<script>
    import { onMount } from "svelte";
    import Card from "$lib/components/common/Card.svelte";
    import Button from "$lib/components/common/Button.svelte";
    import { 
        templates, 
        templatesLoading, 
        loadTemplates, 
        createJobFromTemplate,
        removeTemplate
    } from "$lib/stores/templateStore";
    import { addToast } from "$lib/stores/uiStore";
    import { formatDate } from "$lib/utils/formatters";
    import { jobWizardState, updateJobWizardStep, setJobWizardStep } from "$lib/stores/jobStore";
    import JobWizard from "$lib/components/jobs/JobWizard.svelte";

    import { Plus, Trash, FileText, Play } from 'lucide-svelte';

    // Local state
    let loading = $state(true);
    let newTemplateModal = $state(false);
    let confirmingDelete = $state(null);
    let editingTemplate = $state(null);

    onMount(async () => {
        try {
            await loadTemplates();
        } catch (error) {
            console.error("Error loading templates:", error);
        } finally {
            loading = false;
        }
    });

    function openNewTemplateModal() {
        // Reset wizard state with minimal data for a template
        jobWizardState.set({
            step: 1,
            data: {
                name: '',
                baseUrl: '',
                description: '',
                selectors: [],
                filters: [],
                schedule: null,
                processing: {
                    thumbnails: true,
                    metadata: true
                }
            },
            isTemplate: true
        });
        newTemplateModal = true;
    }

    function editTemplate(template) {
        // Populate wizard with template data
        updateJobWizardStep(1, template);
        setJobWizardStep(1);
        editingTemplate = template.id;
        newTemplateModal = true;
    }

    function confirmDelete(id) {
        confirmingDelete = id;
    }

    function cancelDelete() {
        confirmingDelete = null;
    }

    async function handleDeleteTemplate(id) {
        try {
            await removeTemplate(id);
            confirmingDelete = null;
            addToast('Template deleted successfully', 'success');
        } catch (error) {
            addToast(`Failed to delete template: ${error.message}`, 'error');
        }
    }

    async function handleCreateJob(template) {
        try {
            await createJobFromTemplate(template);
            addToast('Job created successfully from template', 'success');
            window.location.href = '/jobs';
        } catch (error) {
            addToast(`Failed to create job: ${error.message}`, 'error');
        }
    }

    function handleTemplateWizardSuccess(event) {
        const templateData = event.detail.template;
        
        if (editingTemplate) {
            addToast('Template updated successfully', 'success');
            editingTemplate = null;
        } else {
            addToast('Template created successfully', 'success');
        }
        
        newTemplateModal = false;
        loadTemplates(); // Reload templates
    }
</script>

<svelte:head>
    <title>Templates | Crepes</title>
</svelte:head>

<section>
    <div class="flex justify-between items-center mb-6">
        <div>
            <h1 class="text-2xl font-bold mb-2">Job Templates</h1>
            <p class="text-dark-300">Create and manage reusable job templates</p>
        </div>
        <Button variant="primary" onclick={openNewTemplateModal}>
            <Plus class="h-5 w-5 mr-2" />
            Create Template
        </Button>
    </div>

    {#if loading}
        <div class="py-20 flex justify-center">
            <div class="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-primary-500"></div>
        </div>
    {:else if $templates.length === 0}
        <Card class="text-center py-12">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-16 w-16 mx-auto text-dark-500 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
            </svg>
            <h3 class="text-lg font-medium mb-2">No templates found</h3>
            <p class="text-dark-400 mb-4">Create your first job template</p>
            <Button variant="primary" onclick={openNewTemplateModal}>Create Template</Button>
        </Card>
    {:else}
        <div class="space-y-6">
            {#each $templates as template (template.id)}
                <Card class="hover:shadow-lg transition-shadow">
                    <div class="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
                        <div class="flex-1">
                            <h3 class="font-medium text-lg">{template.name || 'Unnamed Template'}</h3>
                            <p class="text-sm text-dark-300 truncate">{template.baseUrl}</p>
                            {#if template.description}
                                <p class="text-sm text-dark-400 mt-2">{template.description}</p>
                            {/if}
                            <div class="mt-3 flex flex-wrap gap-2">
                                {#if template.tags && template.tags.length > 0}
                                    {#each template.tags as tag}
                                        <span class="px-2 py-0.5 bg-base-700 rounded-full text-xs">
                                            {tag}
                                        </span>
                                    {/each}
                                {/if}
                                <span class="px-2 py-0.5 bg-base-700 rounded-full text-xs">
                                    {template.selectors?.length || 0} selectors
                                </span>
                                <span class="text-xs text-dark-400 ml-2">
                                    Created: {formatDate(template.createdAt || new Date())}
                                </span>
                            </div>
                        </div>
                        <div class="flex flex-wrap gap-2">
                            <Button 
                                variant="primary" 
                                size="sm" 
                                onclick={() => handleCreateJob(template)}
                            >
                                <Play class="h-5 w-5 mr-1" />
                                Create Job
                            </Button>
                            <Button 
                                variant="outline" 
                                size="sm" 
                                onclick={() => editTemplate(template)}
                            >
                                <FileText class="h-5 w-5 mr-1" />
                                Edit
                            </Button>
                            {#if confirmingDelete === template.id}
                                <div class="flex items-center space-x-2">
                                    <span class="text-sm text-danger-400">Confirm?</span>
                                    <Button 
                                        variant="danger" 
                                        size="sm" 
                                        onclick={() => handleDeleteTemplate(template.id)}
                                    >
                                        Yes
                                    </Button>
                                    <Button 
                                        variant="outline" 
                                        size="sm" 
                                        onclick={cancelDelete}
                                    >
                                        No
                                    </Button>
                                </div>
                            {:else}
                                <Button 
                                    variant="outline" 
                                    size="sm" 
                                    onclick={() => confirmDelete(template.id)}
                                >
                                    <Trash class="h-5 w-5 mr-1 text-danger-400" />
                                    Delete
                                </Button>
                            {/if}
                        </div>
                    </div>
                </Card>
            {/each}
        </div>
    {/if}
</section>

<!-- Template Creation/Editing Modal -->
{#if newTemplateModal}
    <div class="fixed inset-0 z-50 overflow-y-auto">
        <div class="flex items-center justify-center min-h-screen">
            <div
                class="fixed inset-0 bg-black bg-opacity-75 transition-opacity"
                onclick={() => newTemplateModal = false}
                onkeydown={() => {}}
                role="button"
                aria-label="Close modal"
                tabindex="0"
            ></div>
            <div class="relative bg-base-800 rounded-lg overflow-hidden shadow-xl transform transition-all w-full max-w-5xl">
                <div class="px-6 py-4 border-b border-dark-700">
                    <h2 class="text-xl font-semibold">
                        {editingTemplate ? 'Edit Template' : 'Create Template'}
                    </h2>
                </div>
                
                <!-- Reuse JobWizard but adjust for templates -->
                <JobWizard 
                    isTemplate={true}
                    initialData={editingTemplate ? $templates.find(t => t.id === editingTemplate) : null}
                    on:success={handleTemplateWizardSuccess}
                    on:cancel={() => newTemplateModal = false}
                />
            </div>
        </div>
    </div>
{/if}
