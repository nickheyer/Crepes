/**
 * Check if a URL is valid
 * @param {string} url - URL to validate
 * @returns {boolean} Whether the URL is valid
 */
export function isValidUrl(url) {
    if (!url) return false;

    try {
        const parsed = new URL(url);
        return ['http:', 'https:'].includes(parsed.protocol);
    } catch (e) {
        return false;
    }
}

/**
 * Check if a string is a valid cron expression
 * @param {string} cron - Cron expression to validate
 * @returns {boolean} Whether the cron expression is valid
 */
export function isValidCron(cron) {
    if (!cron) return false;

    // Simple regex for basic cron validation
    // This doesn't validate all possible cron expressions
    const cronRegex = /^(\*|([0-9]|1[0-9]|2[0-9]|3[0-9]|4[0-9]|5[0-9])|\*\/([0-9]|1[0-9]|2[0-9]|3[0-9]|4[0-9]|5[0-9])) (\*|([0-9]|1[0-9]|2[0-3])|\*\/([0-9]|1[0-9]|2[0-3])) (\*|([1-9]|1[0-9]|2[0-9]|3[0-1])|\*\/([1-9]|1[0-9]|2[0-9]|3[0-1])) (\*|([1-9]|1[0-2])|\*\/([1-9]|1[0-2])) (\*|([0-6])|\*\/([0-6]))$/;

    return cronRegex.test(cron);
}

/**
 * Check if a value is empty (null, undefined, empty string, empty array, empty object)
 * @param {*} value - Value to check
 * @returns {boolean} Whether the value is empty
 */
export function isEmpty(value) {
    if (value === null || value === undefined) return true;
    if (typeof value === 'string') return value.trim() === '';
    if (Array.isArray(value)) return value.length === 0;
    if (typeof value === 'object') return Object.keys(value).length === 0;
    return false;
}

/**
 * Validate a selector string (CSS or XPath)
 * @param {string} selector - Selector to validate
 * @param {string} type - Selector type ('css' or 'xpath')
 * @returns {boolean} Whether the selector is valid
 */
export function isValidSelector(selector, type = 'css') {
    if (!selector) return false;

    if (type === 'css') {
        try {
            // Try to parse the selector using DOM API
            document.querySelector(selector);
            return true;
        } catch (e) {
            return false;
        }
    } else if (type === 'xpath') {
        try {
            // Try to evaluate the XPath expression
            document.evaluate(selector, document, null, XPathResult.ANY_TYPE, null);
            return true;
        } catch (e) {
            return false;
        }
    }

    return false;
}

/**
 * Validate a job configuration
 * @param {Object} jobConfig - Job configuration to validate
 * @returns {Object} Validation result with success flag and error messages
 */
export function validateJobConfig(jobConfig) {
    const errors = {};

    // Validate basic required fields
    if (!jobConfig.name) {
        errors.name = 'Job name is required';
    }

    if (!jobConfig.baseUrl) {
        errors.baseUrl = 'Base URL is required';
    } else if (!isValidUrl(jobConfig.baseUrl)) {
        errors.baseUrl = 'Base URL is not valid';
    }

    // Validate selectors if any
    if (jobConfig.selectors && jobConfig.selectors.length > 0) {
        const selectorErrors = [];

        jobConfig.selectors.forEach((selector, index) => {
            const selectorError = {};

            if (!selector.name) {
                selectorError.name = 'Selector name is required';
            }

            if (!selector.value) {
                selectorError.value = 'Selector value is required';
            }

            if (!selector.purpose) {
                selectorError.purpose = 'Selector purpose is required';
            }

            if (Object.keys(selectorError).length > 0) {
                selectorErrors[index] = selectorError;
            }
        });

        if (selectorErrors.length > 0) {
            errors.selectors = selectorErrors;
        }
    }

    // Validate schedule if provided
    if (jobConfig.schedule && !isValidCron(jobConfig.schedule)) {
        errors.schedule = 'Invalid cron schedule format';
    }

    return {
        isValid: Object.keys(errors).length === 0,
        errors
    };
}

/**
 * Validate asset filters
 * @param {Object} filters - Filters to validate
 * @returns {Object} Validation result with success flag and error messages
 */
export function validateAssetFilters(filters) {
    const errors = {};

    // Validate date range if both from and to are provided
    if (filters.dateRange && filters.dateRange.from && filters.dateRange.to) {
        const fromDate = new Date(filters.dateRange.from);
        const toDate = new Date(filters.dateRange.to);

        if (fromDate > toDate) {
            errors.dateRange = 'From date must be before To date';
        }
    }

    return {
        isValid: Object.keys(errors).length === 0,
        errors
    };
}

/**
 * Validate template data
 * @param {Object} templateData - Template data to validate
 * @returns {Object} Validation result with success flag and error messages
 */
export function validateTemplateData(templateData) {
    const errors = {};

    if (!templateData.name) {
        errors.name = 'Template name is required';
    }

    // Validate the job configuration inside the template
    if (templateData.jobConfig) {
        const jobValidation = validateJobConfig(templateData.jobConfig);

        if (!jobValidation.isValid) {
            errors.jobConfig = jobValidation.errors;
        }
    } else {
        errors.jobConfig = 'Job configuration is required';
    }

    return {
        isValid: Object.keys(errors).length === 0,
        errors
    };
}

/**
 * Validate regex pattern
 * @param {string} pattern - Regex pattern string to validate
 * @returns {boolean} Whether the pattern is a valid regex
 */
export function isValidRegex(pattern) {
    if (!pattern) return false;

    try {
        new RegExp(pattern);
        return true;
    } catch (e) {
        return false;
    }
}

/**
 * Validate a form field based on specified rules
 * @param {*} value - Field value to validate
 * @param {Object} rules - Validation rules
 * @returns {Object} Validation result with valid flag and error message
 */
export function validateField(value, rules) {
    // Required validation
    if (rules.required && isEmpty(value)) {
        return { valid: false, message: rules.requiredMessage || 'This field is required' };
    }

    // Min length validation
    if (rules.minLength && typeof value === 'string' && value.length < rules.minLength) {
        return { valid: false, message: `Minimum length is ${rules.minLength} characters` };
    }

    // Max length validation
    if (rules.maxLength && typeof value === 'string' && value.length > rules.maxLength) {
        return { valid: false, message: `Maximum length is ${rules.maxLength} characters` };
    }

    // Pattern validation
    if (rules.pattern && !new RegExp(rules.pattern).test(value)) {
        return { valid: false, message: rules.patternMessage || 'Invalid format' };
    }

    // URL validation
    if (rules.isUrl && !isValidUrl(value)) {
        return { valid: false, message: 'Please enter a valid URL' };
    }

    // Email validation
    if (rules.isEmail && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value)) {
        return { valid: false, message: 'Please enter a valid email address' };
    }

    // Number validation
    if (rules.isNumber && isNaN(Number(value))) {
        return { valid: false, message: 'Please enter a valid number' };
    }

    // Min value validation
    if (rules.min !== undefined && Number(value) < rules.min) {
        return { valid: false, message: `Minimum value is ${rules.min}` };
    }

    // Max value validation
    if (rules.max !== undefined && Number(value) > rules.max) {
        return { valid: false, message: `Maximum value is ${rules.max}` };
    }

    // Custom validation
    if (rules.validate && typeof rules.validate === 'function') {
        const result = rules.validate(value);
        if (result !== true) {
            return { valid: false, message: result || 'Invalid value' };
        }
    }

    return { valid: true, message: '' };
}