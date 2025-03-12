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
    return [];
  } finally {
    state.templatesLoading = false;
  }
}

// LOAD TEMPLATE EXAMPLES
export async function loadTemplateExamples() {
  try {
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
    // ENSURE ID IS SET BY BACKEND
    const dataToSend = { ...templateData };
    if (dataToSend.id) {
      delete dataToSend.id;
    }
    
    const newTemplate = await createTemplate(dataToSend);
    state.templates = [newTemplate, ...state.templates];
    
    return newTemplate;
  } catch (error) {
    
    throw error;
  }
}

// UPDATE TEMPLATE
export async function updateExistingTemplate(templateId, templateData) {
  try {
    const updatedTemplate = await updateTemplate(templateId, templateData);
    state.templates = state.templates.map(template => 
      template.id === templateId ? {...template, ...updatedTemplate} : template
    );
    
    return updatedTemplate;
  } catch (error) {
    
    throw error;
  }
}

// DELETE TEMPLATE
export async function removeTemplate(templateId) {
  try {
    await deleteTemplate(templateId);
    state.templates = state.templates.filter(template => template.id !== templateId);
  } catch (error) {
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
    const job = await createJobFromTemplate(templateId);
    return job;
  } catch (error) {
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
    throw error;
  }
}