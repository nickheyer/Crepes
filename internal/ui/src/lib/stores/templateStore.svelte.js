import { writable } from 'svelte/store';
import { addToast } from './uiStore.svelte';
import { 
  fetchTemplates, 
  fetchTemplateExamples,
  createTemplate, 
  updateTemplate, 
  deleteTemplate, 
  createJobFromTemplate 
} from '$lib/utils/api';

// TEMPLATE STORE STATE - USING SVELTE 5 RUNES
export const state = $state({
  templates: [],
  templatesLoading: false,
  selectedTemplate: null,
  templateExamples: {}
});

// LOAD TEMPLATES
export async function loadTemplates() {
  state.templatesLoading = true;
  try {
    const { fetchTemplates } = await import('$lib/utils/api');
    const { addToast } = await import('$lib/stores/uiStore.svelte');
    
    const data = await fetchTemplates();
    
    // NORMALIZE TEMPLATE DATA
    const templateArray = Array.isArray(data) ? data : [];
    const normalizedTemplates = templateArray.map(template => ({
      ...template,
      // ENSURE ARRAYS AND OBJECTS
      selectors: Array.isArray(template.selectors) ? template.selectors : [],
      filters: Array.isArray(template.filters) ? template.filters : [],
      rules: template.rules || {},
      processing: template.processing || {
        thumbnails: true,
        metadata: true,
        deduplication: true
      },
      tags: Array.isArray(template.tags) ? template.tags : []
    }));
    
    state.templates = normalizedTemplates;
    return normalizedTemplates;
  } catch (error) {
    const { addToast } = await import('$lib/stores/uiStore.svelte');
    addToast(`FAILED TO LOAD TEMPLATES: ${error.message}`, 'error');
    return [];
  } finally {
    state.templatesLoading = false;
  }
}

// LOAD TEMPLATE EXAMPLES
export async function loadTemplateExamples() {
  try {
    const { fetchTemplateExamples } = await import('$lib/utils/api');
    const data = await fetchTemplateExamples();
    state.templateExamples = data;
    return data;
  } catch (error) {
    console.error("FAILED TO LOAD TEMPLATE EXAMPLES:", error);
    return {};
  }
}

// CREATE TEMPLATE
export async function createNewTemplate(templateData) {
  try {
    const { createTemplate } = await import('$lib/utils/api');
    const { addToast } = await import('$lib/stores/uiStore.svelte');
    
    // ENSURE ID IS SET BY BACKEND
    const dataToSend = { ...templateData };
    if (dataToSend.id) {
      delete dataToSend.id;
    }
    
    const newTemplate = await createTemplate(dataToSend);
    state.templates = [newTemplate, ...state.templates];
    
    addToast('TEMPLATE CREATED SUCCESSFULLY', 'success');
    return newTemplate;
  } catch (error) {
    const { addToast } = await import('$lib/stores/uiStore.svelte');
    addToast(`FAILED TO CREATE TEMPLATE: ${error.message}`, 'error');
    throw error;
  }
}

// UPDATE TEMPLATE
export async function updateExistingTemplate(templateId, templateData) {
  try {
    const { updateTemplate } = await import('$lib/utils/api');
    const { addToast } = await import('$lib/stores/uiStore.svelte');
    
    const updatedTemplate = await updateTemplate(templateId, templateData);
    state.templates = state.templates.map(template => 
      template.id === templateId ? {...template, ...updatedTemplate} : template
    );
    
    addToast('TEMPLATE UPDATED SUCCESSFULLY', 'success');
    return updatedTemplate;
  } catch (error) {
    const { addToast } = await import('$lib/stores/uiStore.svelte');
    addToast(`FAILED TO UPDATE TEMPLATE: ${error.message}`, 'error');
    throw error;
  }
}

// DELETE TEMPLATE
export async function removeTemplate(templateId) {
  try {
    const { deleteTemplate } = await import('$lib/utils/api');
    const { addToast } = await import('$lib/stores/uiStore.svelte');
    
    await deleteTemplate(templateId);
    state.templates = state.templates.filter(template => template.id !== templateId);
    
    addToast('TEMPLATE DELETED SUCCESSFULLY', 'success');
  } catch (error) {
    const { addToast } = await import('$lib/stores/uiStore.svelte');
    addToast(`FAILED TO DELETE TEMPLATE: ${error.message}`, 'error');
    throw error;
  }
}

// SELECT TEMPLATE
export function selectTemplate(template) {
  state.selectedTemplate = template;
}

// CREATE JOB FROM TEMPLATE
export async function createJobFromTemplateId(templateId) {
  try {
    const { createJobFromTemplate } = await import('$lib/utils/api');
    const { addToast } = await import('$lib/stores/uiStore.svelte');
    
    const job = await createJobFromTemplate(templateId);
    addToast('JOB CREATED SUCCESSFULLY FROM TEMPLATE', 'success');
    return job;
  } catch (error) {
    const { addToast } = await import('$lib/stores/uiStore.svelte');
    addToast(`FAILED TO CREATE JOB FROM TEMPLATE: ${error.message}`, 'error');
    throw error;
  }
}

// USE TEMPLATE EXAMPLE
export async function useTemplateExample(exampleKey) {
  try {
    // FIRST CHECK IF EXAMPLES ARE LOADED
    if (Object.keys(state.templateExamples).length === 0) {
      await loadTemplateExamples();
    }
    
    // CHECK IF THE EXAMPLE EXISTS
    if (state.templateExamples[exampleKey]) {
      return state.templateExamples[exampleKey];
    } else {
      throw new Error('TEMPLATE EXAMPLE NOT FOUND');
    }
  } catch (error) {
    const { addToast } = await import('$lib/stores/uiStore.svelte');
    addToast(`FAILED TO LOAD TEMPLATE EXAMPLE: ${error.message}`, 'error');
    throw error;
  }
}