import { writable, derived } from 'svelte/store';
import { addToast } from './uiStore';
import { fetchAssets, fetchAssetDetails, deleteAsset, regenerateThumbnail } from '$lib/utils/api';

// ASSET STORE STATE
export const assets = writable([]);
export const selectedAsset = writable(null);
export const assetsLoading = writable(false);
export const assetViewerOpen = writable(false);

// ASSET FILTER STATE
export const assetFilters = writable({
  type: '',
  jobId: '',
  search: '',
  dateRange: {
    from: null,
    to: null
  },
  sortBy: 'date',
  sortDirection: 'desc'
});

// DERIVED FILTERED ASSETS
export const filteredAssets = derived(
  [assets, assetFilters],
  ([$assets, $filters]) => {
    // ENSURE ASSETS IS ALWAYS AN ARRAY
    let assetArray = Array.isArray($assets) ? $assets : [];
    let result = [...assetArray];
    
    // APPLY TYPE FILTER
    if ($filters.type) {
      result = result.filter(asset => asset.type === $filters.type);
    }
    
    // APPLY JOB FILTER
    if ($filters.jobId) {
      result = result.filter(asset => asset.jobId === $filters.jobId);
    }
    
    // APPLY TEXT SEARCH
    if ($filters.search) {
      const searchLower = $filters.search.toLowerCase();
      result = result.filter(asset => 
        (asset.title && asset.title.toLowerCase().includes(searchLower)) ||
        (asset.description && asset.description.toLowerCase().includes(searchLower)) ||
        (asset.url && asset.url.toLowerCase().includes(searchLower))
      );
    }
    
    // APPLY DATE RANGE FILTER
    if ($filters.dateRange.from) {
      const fromDate = new Date($filters.dateRange.from);
      result = result.filter(asset => 
        asset.date && new Date(asset.date) >= fromDate
      );
    }
    
    if ($filters.dateRange.to) {
      const toDate = new Date($filters.dateRange.to);
      result = result.filter(asset => 
        asset.date && new Date(asset.date) <= toDate
      );
    }
    
    // APPLY SORTING
    result.sort((a, b) => {
      const direction = $filters.sortDirection === 'asc' ? 1 : -1;
      switch ($filters.sortBy) {
        case 'date':
          return direction * ((new Date(a.date || 0)) - (new Date(b.date || 0)));
        case 'title':
          return direction * ((a.title || '').localeCompare(b.title || ''));
        case 'type':
          return direction * ((a.type || '').localeCompare(b.type || ''));
        case 'size':
          return direction * ((a.size || 0) - (b.size || 0));
        default:
          return 0;
      }
    });
    
    return result;
  }
);

// GROUPED ASSETS (BY TYPE)
export const assetsByType = derived(filteredAssets, $assets => {
  const groups = {};
  // ENSURE ASSETS IS ALWAYS AN ARRAY
  const assetArray = Array.isArray($assets) ? $assets : [];
  
  assetArray.forEach(asset => {
    const type = asset.type || 'unknown';
    if (!groups[type]) {
      groups[type] = [];
    }
    groups[type].push(asset);
  });
  return groups;
});

// ASSET COUNT BY TYPE
export const assetCounts = derived(assets, $assets => {
  // ENSURE ASSETS IS ALWAYS AN ARRAY
  const assetArray = Array.isArray($assets) ? $assets : [];
  
  const counts = {
    total: assetArray.length,
    image: 0,
    video: 0,
    audio: 0,
    document: 0,
    unknown: 0
  };
  
  assetArray.forEach(asset => {
    if (counts[asset.type]) {
      counts[asset.type]++;
    } else {
      counts.unknown++;
    }
  });
  
  return counts;
});

// LOAD ASSETS FROM API
export async function loadAssets(filters = {}) {
  assetsLoading.set(true);
  try {
    const data = await fetchAssets(filters);
    // ENSURE WE'RE SETTING AN ARRAY
    assets.set(Array.isArray(data) ? data : []);
    return data;
  } catch (error) {
    addToast(`Failed to load assets: ${error.message}`, 'error');
    // SET EMPTY ARRAY ON ERROR
    assets.set([]);
    return [];
  } finally {
    assetsLoading.set(false);
  }
}

// LOAD ASSET DETAILS
export async function loadAssetDetails(assetId) {
  try {
    const asset = await fetchAssetDetails(assetId);
    selectedAsset.set(asset);
    return asset;
  } catch (error) {
    addToast(`Failed to load asset details: ${error.message}`, 'error');
    throw error;
  }
}

// DELETE AN ASSET
export async function removeAsset(assetId) {
  try {
    await deleteAsset(assetId);
    // UPDATE ASSETS STORE SAFELY
    assets.update(allAssets => {
      // ENSURE ALLASSETS IS AN ARRAY
      const assetArray = Array.isArray(allAssets) ? allAssets : [];
      return assetArray.filter(asset => asset.id !== assetId);
    });
    addToast('Asset deleted successfully', 'success');
  } catch (error) {
    addToast(`Failed to delete asset: ${error.message}`, 'error');
    throw error;
  }
}

// REGENERATE THUMBNAIL FOR AN ASSET
export async function regenerateAssetThumbnail(assetId) {
  try {
    const result = await regenerateThumbnail(assetId);
    // UPDATE ASSET IN STORE SAFELY
    assets.update(allAssets => {
      // ENSURE ALLASSETS IS AN ARRAY
      const assetArray = Array.isArray(allAssets) ? allAssets : [];
      return assetArray.map(asset => 
        asset.id === assetId 
          ? { ...asset, thumbnailPath: result.thumbnailPath } 
          : asset
      );
    });
    
    // UPDATE SELECTED ASSET IF IT'S THE SAME ONE
    selectedAsset.update(current => 
      current && current.id === assetId 
        ? { ...current, thumbnailPath: result.thumbnailPath } 
        : current
    );
    
    addToast('Thumbnail regenerated successfully', 'success');
    return result.thumbnailPath;
  } catch (error) {
    addToast(`Failed to regenerate thumbnail: ${error.message}`, 'error');
    throw error;
  }
}

// UPDATE FILTERS
export function updateFilters(newFilters) {
  assetFilters.update(current => ({
    ...current,
    ...newFilters
  }));
}

// RESET FILTERS
export function resetFilters() {
  assetFilters.set({
    type: '',
    jobId: '',
    search: '',
    dateRange: {
      from: null,
      to: null
    },
    sortBy: 'date',
    sortDirection: 'desc'
  });
}

// OPEN ASSET VIEWER WITH A SPECIFIC ASSET
export function viewAsset(asset) {
  selectedAsset.set(asset);
  assetViewerOpen.set(true);
}

// CLOSE ASSET VIEWER
export function closeAssetViewer() {
  assetViewerOpen.set(false);
}
