import { writable } from 'svelte/store';
import { addToast } from './uiStore';
import { fetchTemplates, createTemplate, updateTemplate, deleteTemplate } from '$lib/utils/api';

// Template store state
export const templates = writable([]);
export const templatesLoading = writable(false);
export const selectedTemplate = writable(null);

// Load templates
export async function loadTemplates() {
  templatesLoading.set(true);
  
  try {
    const data = await fetchTemplates();
    templates.set(data);
    return data;
  } catch (error) {
    addToast(`Failed to load templates: ${error.message}`, 'error');
    return [];
  } finally {
    templatesLoading.set(false);
  }
}

// Create template
export async function createNewTemplate(templateData) {
  try {
    const newTemplate = await createTemplate(templateData);
    templates.update(allTemplates => [newTemplate, ...allTemplates]);
    addToast('Template created successfully', 'success');
    return newTemplate;
  } catch (error) {
    addToast(`Failed to create template: ${error.message}`, 'error');
    throw error;
  }
}

// Update template
export async function updateExistingTemplate(templateId, templateData) {
  try {
    const updatedTemplate = await updateTemplate(templateId, templateData);
    templates.update(allTemplates => 
      allTemplates.map(template => template.id === templateId ? {...template, ...updatedTemplate} : template)
    );
    addToast('Template updated successfully', 'success');
    return updatedTemplate;
  } catch (error) {
    addToast(`Failed to update template: ${error.message}`, 'error');
    throw error;
  }
}

// Delete template
export async function removeTemplate(templateId) {
  try {
    await deleteTemplate(templateId);
    templates.update(allTemplates => allTemplates.filter(template => template.id !== templateId));
    addToast('Template deleted successfully', 'success');
  } catch (error) {
    addToast(`Failed to delete template: ${error.message}`, 'error');
    throw error;
  }
}

// Select template
export function selectTemplate(template) {
  selectedTemplate.set(template);
}

// Create job from template
export function createJobFromTemplate(template) {
  // This would typically integrate with the jobStore
  // For now, we'll just return the template data
  return template;
}
