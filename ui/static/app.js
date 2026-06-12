const pageInfo = {
    figma: {
        title: "Figma Engine",
        subtitle: "Extract layout blueprints directly from Figma URLs."
    },
    image: {
        title: "Vision Engine",
        subtitle: "Generate blueprints from design mockups or screenshots."
    },
    settings: {
        title: "Settings",
        subtitle: "Configure API keys and default directories."
    }
};

let isLoading = false;

function navigatePage(event, tabId, updateHash = true) {
    if (event) {
        event.preventDefault();
    }
    
    if (isLoading) {
        showMessage('error', 'Please wait until the current operation finishes before navigating.');
        return;
    }
    
    if (updateHash && window.location.hash !== '#' + tabId) {
        window.history.pushState(null, '', '#' + tabId);
    }
    
    // Fallback if View Transitions API is not supported
    if (!document.startViewTransition) {
        updateDOM(tabId);
        return;
    }
    
    // Smooth transition between sections
    document.startViewTransition(() => updateDOM(tabId));
}

function updateDOM(tabId) {
    // Update active content
    document.querySelectorAll('.page-section').forEach(el => el.classList.remove('active'));
    const targetSection = document.getElementById(tabId);
    if (targetSection) targetSection.classList.add('active');
    
    // Update active nav link
    document.querySelectorAll('.nav-link').forEach(el => el.classList.remove('active'));
    const targetNav = document.querySelector(`.nav-link[onclick*="'${tabId}'"]`);
    if (targetNav) targetNav.classList.add('active');
    
    // Update header texts
    const info = pageInfo[tabId];
    if (info) {
        const titleEl = document.getElementById('page-title');
        const subtitleEl = document.getElementById('page-subtitle');
        if (titleEl) titleEl.innerText = info.title;
        if (subtitleEl) subtitleEl.innerText = info.subtitle;
    }
    
    clearMessages();
    const resultContainer = document.getElementById('dynamic-result-container');
    if (resultContainer) {
        resultContainer.style.display = 'none';
    }
}

/* ---------------- Toast System ---------------- */

const toastIcons = {
    success: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"></polyline></svg>`,
    error: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"></circle><line x1="12" y1="8" x2="12" y2="12"></line><line x1="12" y1="16" x2="12.01" y2="16"></line></svg>`,
    loading: `<span class="toast-spinner"></span>`
};

function getToastContainer() {
    let container = document.getElementById('toast-container');
    if (!container) {
        container = document.createElement('div');
        container.id = 'toast-container';
        document.body.appendChild(container);
    }
    return container;
}

function spawnToast(type, text, duration) {
    const container = getToastContainer();
    const toast = document.createElement('div');
    toast.className = `toast toast-${type}`;
    toast.innerHTML = `<span class="toast-icon">${toastIcons[type] || ''}</span><span class="toast-text"></span>`;
    toast.querySelector('.toast-text').textContent = text;
    container.appendChild(toast);
    
    // Trigger enter transition on next frame
    requestAnimationFrame(() => {
        requestAnimationFrame(() => toast.classList.add('visible'));
    });
    
    if (duration > 0) {
        setTimeout(() => dismissToast(toast), duration);
    }
    
    if (type !== 'loading') {
        toast.addEventListener('click', () => dismissToast(toast));
    }
    
    return toast;
}

function dismissToast(toast) {
    if (!toast || toast.dataset.dismissed) return;
    toast.dataset.dismissed = '1';
    toast.classList.remove('visible');
    setTimeout(() => toast.remove(), 350);
}

function showMessage(type, text) {
    if (!text) return;
    spawnToast(type === 'error' ? 'error' : 'success', text, type === 'error' ? 6000 : 4000);
}

function clearMessages() {
    const container = document.getElementById('messages-container');
    if (container) container.innerHTML = '';
}

/* ---------------- Loading Toast ---------------- */

const loadingPhrases = [
    "Reticulating splines...",
    "Extracting pure design essence...",
    "Persuading pixels to align...",
    "Downloading layout blueprints...",
    "Negotiating with Figma's servers...",
    "Crunching the vector math...",
    "Painting with digital colors...",
    "Constructing DOM tree from scratch...",
    "Warming up the neural engines...",
    "Optimizing whitespace logic...",
    "Distilling UI components...",
    "Bribing the rate limiters...",
    "Translating design tokens...",
    "Brewing digital coffee..."
];

let loaderInterval = null;
let loadingToast = null;

function toggleLoader(show, forceText = null) {
    if (show) {
        if (loadingToast) {
            if (forceText) {
                const textEl = loadingToast.querySelector('.toast-text');
                if (textEl) textEl.textContent = forceText;
            }
            return;
        }
        
        let currentIndex = Math.floor(Math.random() * loadingPhrases.length);
        loadingToast = spawnToast('loading', forceText || loadingPhrases[currentIndex], 0);
        
        if (!forceText) {
            loaderInterval = setInterval(() => {
                currentIndex = (currentIndex + 1) % loadingPhrases.length;
                const textEl = loadingToast && loadingToast.querySelector('.toast-text');
                if (!textEl) return;
                textEl.classList.add('fading');
                setTimeout(() => {
                    textEl.textContent = loadingPhrases[currentIndex];
                    textEl.classList.remove('fading');
                }, 200);
            }, 2500);
        }
    } else {
        if (loaderInterval) {
            clearInterval(loaderInterval);
            loaderInterval = null;
        }
        dismissToast(loadingToast);
        loadingToast = null;
    }
}

function renderResult(data) {
    const container = document.getElementById('dynamic-result-container');
    const box = document.getElementById('result-box-content');
    const actions = document.getElementById('result-actions');
    
    if (!data.Result) {
        container.style.display = 'none';
        return;
    }
    
    box.innerText = data.Result;
    
    let html = '';
    
    if (data.ImageHash) {
        html += `
            <label class="checkbox-label">
                <input type="checkbox" id="attachImage" value="1">
                <span class="checkmark"></span>
                Attach original image
            </label>
            <input type="hidden" id="imageHash" value="${data.ImageHash}">
            <input type="hidden" id="imageExt" value="${data.ImageExt || ''}">
        `;
    }
    
    if (data.FigmaAssets) {
        const encodedAssets = data.FigmaAssets.replace(/"/g, '&quot;');
        html += `
            <label class="checkbox-label">
                <input type="checkbox" id="downloadAssets" value="1" checked>
                <span class="checkmark"></span>
                Download Assets & Design
            </label>
            <input type="hidden" id="figmaAssets" value="${encodedAssets}">
        `;
    }
    
    html += `
        <label class="checkbox-label mr-auto">
            <input type="checkbox" id="nukeDir" value="1">
            <span class="checkmark"></span>
            Nuke after implementation
        </label>
        <div class="action-buttons">
            <button type="button" class="btn-secondary" id="copyBtn" onclick="copyToClipboard()">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg>
                Copy
            </button>
            <button type="button" class="btn-primary" id="saveBtn" onclick="saveIntent()">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z"></path><polyline points="17 21 17 13 7 13 7 21"></polyline><polyline points="7 3 7 8 15 8"></polyline></svg>
                Save to .ai-intent
            </button>
        </div>
    `;
    
    actions.innerHTML = html;
    
    if (document.startViewTransition) {
        document.startViewTransition(() => {
            container.style.display = 'block';
        }).finished.then(() => {
            container.scrollIntoView({ behavior: 'smooth', block: 'start' });
        });
    } else {
        container.style.display = 'block';
        container.scrollIntoView({ behavior: 'smooth', block: 'start' });
    }
}

async function handleFormSubmit(e, loadingText) {
    e.preventDefault();
    
    if (isLoading) return;
    
    clearMessages();
    const form = e.target;
    
    isLoading = true;
    toggleLoader(true, loadingText);
    const btn = form.querySelector('button[type="submit"]');
    if (btn) btn.disabled = true;
    
    try {
        const formData = new FormData(form);
        let bodyData;
        
        if (form.enctype === 'multipart/form-data') {
            bodyData = formData;
        } else {
            bodyData = new URLSearchParams(formData);
        }
        
        const res = await fetch(form.action, {
            method: form.method,
            body: bodyData,
            headers: {
                'Accept': 'application/json'
            }
        });
        
        let data;
        try {
            data = await res.json();
        } catch (err) {
            showMessage('error', 'Received an invalid response from the server.');
            return;
        }
        
        if (data.Error) {
            showMessage('error', data.Error);
            if (data.Result) renderResult(data);
        } else if (data.SuccessMessage) {
            showMessage('success', data.SuccessMessage);
            updateSidebarStatus(data);
        } else {
            if (data.Result) renderResult(data);
        }
        
    } catch (err) {
        showMessage('error', 'A network error occurred. Please try again.');
    } finally {
        isLoading = false;
        toggleLoader(false);
        if (btn) btn.disabled = false;
    }
}

function updateSidebarStatus(data) {
    if (data.FigmaReady !== undefined) {
        const figmaEl = document.getElementById('status-figma');
        if (figmaEl) {
            if (data.FigmaReady) figmaEl.classList.add('ready');
            else figmaEl.classList.remove('ready');
        }
        
        // Also update the form button state if needed, though this requires reloading to fully unlock inputs
        const figmaBtn = document.querySelector('#form-figma button[type="submit"]');
        if (figmaBtn) figmaBtn.disabled = !data.FigmaReady;
        document.querySelectorAll('#form-figma input[type="url"]').forEach(inp => {
            inp.disabled = !data.FigmaReady;
        });
        const addUrlBtn = document.getElementById('add-figma-url');
        if (addUrlBtn) addUrlBtn.disabled = !data.FigmaReady;
    }
    
    if (data.VisionReady !== undefined) {
        const visionEl = document.getElementById('status-vision');
        if (visionEl) {
            if (data.VisionReady) visionEl.classList.add('ready');
            else visionEl.classList.remove('ready');
            
            if (data.ModelDisplayName) {
                const textEl = visionEl.querySelector('.status-text');
                if (textEl) textEl.innerText = data.ModelDisplayName;
            }
        }
        
        const visionBtn = document.querySelector('#form-image button[type="submit"]');
        const visionInput = document.querySelector('#form-image input[type="file"]');
        const visionLabel = document.querySelector('#form-image .file-upload-wrapper');
        if (visionBtn) visionBtn.disabled = !data.VisionReady;
        if (visionInput) visionInput.disabled = !data.VisionReady;
        if (visionLabel) {
            if (data.VisionReady) visionLabel.classList.remove('disabled');
            else visionLabel.classList.add('disabled');
        }
    }
}

/* ---------------- Figma URL rows (responsive variants) ---------------- */

const MAX_FIGMA_URLS = 5;

function updateAddUrlButton() {
    const list = document.getElementById('figma-url-list');
    const addBtn = document.getElementById('add-figma-url');
    if (!list || !addBtn) return;
    const atMax = list.querySelectorAll('.url-row').length >= MAX_FIGMA_URLS;
    const figmaDisabled = list.querySelector('input[type="url"]')?.disabled;
    addBtn.disabled = atMax || !!figmaDisabled;
}

function addFigmaUrlRow() {
    const list = document.getElementById('figma-url-list');
    if (!list) return;
    
    const rows = list.querySelectorAll('.url-row');
    if (rows.length >= MAX_FIGMA_URLS) {
        showMessage('error', `You can add up to ${MAX_FIGMA_URLS} URLs.`);
        return;
    }
    
    const row = rows[0].cloneNode(true);
    const input = row.querySelector('input');
    input.value = '';
    input.removeAttribute('required'); // only the first URL is mandatory
    list.appendChild(row);
    input.focus();
    updateAddUrlButton();
}

function initFigmaUrlList() {
    const list = document.getElementById('figma-url-list');
    if (!list) return;
    
    list.addEventListener('click', (e) => {
        const btn = e.target.closest('.url-remove');
        if (!btn) return;
        const rows = list.querySelectorAll('.url-row');
        if (rows.length <= 1) return;
        btn.closest('.url-row').remove();
        updateAddUrlButton();
    });
    
    updateAddUrlButton();
}

document.addEventListener('DOMContentLoaded', () => {
    initFigmaUrlList();

    // Hash Routing
    const handleHash = () => {
        const hash = window.location.hash.substring(1);
        if (pageInfo[hash]) {
            navigatePage(null, hash, false);
        } else {
            navigatePage(null, 'figma', false);
        }
    };
    
    handleHash();
    
    window.addEventListener('hashchange', () => {
        handleHash();
    });

    // Form Interceptors
    const figmaForm = document.getElementById('form-figma');
    if (figmaForm) {
        figmaForm.addEventListener('submit', (e) => handleFormSubmit(e, null));
    }
    
    const imageForm = document.getElementById('form-image');
    if (imageForm) {
        imageForm.addEventListener('submit', (e) => handleFormSubmit(e, null));
    }
    
    const settingsForm = document.getElementById('form-settings');
    if (settingsForm) {
        settingsForm.addEventListener('submit', (e) => handleFormSubmit(e, "Saving..."));
    }

    // File input styling update
    const fileInput = document.querySelector('.hidden-file-input');
    const fileText = document.querySelector('.file-upload-title');
    const uploadIcon = document.querySelector('.upload-icon svg');
    
    if (fileInput && fileText) {
        fileInput.addEventListener('change', function() {
            if (this.files && this.files.length > 0) {
                fileText.textContent = this.files[0].name;
                fileText.style.color = 'var(--text-primary)';
                if (uploadIcon) uploadIcon.style.color = 'var(--accent)';
                document.querySelector('.file-upload-wrapper').classList.add('has-file');
            } else {
                fileText.textContent = 'Drop your image here';
                fileText.style.color = '';
                if (uploadIcon) uploadIcon.style.color = '';
                document.querySelector('.file-upload-wrapper').classList.remove('has-file');
            }
        });
    }
});

async function copyToClipboard() {
    const pre = document.querySelector('.result-box');
    if (!pre) return;
    try {
        await navigator.clipboard.writeText(pre.innerText);
        const btn = document.getElementById('copyBtn');
        const orig = btn.innerHTML;
        btn.innerHTML = `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="20 6 9 17 4 12"></polyline></svg> Copied!`;
        setTimeout(() => btn.innerHTML = orig, 2000);
    } catch (err) {
        console.error('Failed to copy', err);
    }
}

async function saveIntent() {
    const pre = document.querySelector('.result-box');
    if (!pre) return;
    if (isLoading) return;
    const btn = document.getElementById('saveBtn');
    btn.disabled = true;
    
    // Check if downloading assets is requested
    const dlAssets = document.getElementById('downloadAssets');
    const isDownloading = dlAssets && dlAssets.checked;
    
    isLoading = true;
    toggleLoader(true, isDownloading ? null : "Saving...");
    const origHtml = btn.innerHTML;
    
    let content = pre.innerText;
    
    const attachImage = document.getElementById('attachImage');
    const nukeDir = document.getElementById('nukeDir');
    const imageHash = document.getElementById('imageHash');
    const imageExt = document.getElementById('imageExt');
    
    const randomHash = Math.random().toString(36).substring(2, 8);
    const planDirName = "ui-prompter-plan-" + randomHash;

    if (attachImage && attachImage.checked) {
        content += "\n\nI have attached the original image for more context. It is available in the `" + planDirName + "` directory.";
    }
    
    if (nukeDir && nukeDir.checked) {
        content += "\n\nNuke after implementation: Please completely delete the directory `" + planDirName + "` and all its contents when you are done.";
    }

    const formData = new URLSearchParams();
    formData.append('content', content);
    formData.append('plan_dir', planDirName);
    
    if (attachImage && attachImage.checked && imageHash) {
        formData.append('attach_image', 'true');
        formData.append('image_hash', imageHash.value);
        if (imageExt) {
            formData.append('image_ext', imageExt.value);
        }
    }

    const downloadAssets = document.getElementById('downloadAssets');
    const figmaAssets = document.getElementById('figmaAssets');
    
    if (downloadAssets && downloadAssets.checked && figmaAssets) {
        formData.append('figma_assets', figmaAssets.value);
        content += "\n\nNote: The extracted image assets are available in the local `" + planDirName + "/assets/` directory. Move ONLY the contents of `" + planDirName + "/assets/` into the appropriate project directory and reference them in the code. The full-design screenshot(s) (design*.png) in `" + planDirName + "/` are strictly a visual reference for you — DO NOT copy or move them into the project, and DO NOT reference them in the code.";
        formData.set('content', content);
    }

    try {
        btn.innerHTML = `<div class="spinner-small"></div> Saving...`;
        btn.disabled = true;
        const res = await fetch('/api/save', {
            method: 'POST',
            body: formData,
            headers: {
                'Accept': 'application/json'
            }
        });
        
        if (res.ok) {
            let payload = null;
            try {
                payload = await res.json();
            } catch (err) {
                // Non-JSON response; treat as plain success
            }
            
            btn.innerHTML = `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="20 6 9 17 4 12"></polyline></svg> Saved!`;
            
            if (payload && payload.Warnings && payload.Warnings.length > 0) {
                showMessage('error', 'Saved, but with issues: ' + payload.Warnings.join(' | '));
            } else {
                showMessage('success', (payload && payload.Message) || 'Saved successfully.');
            }
        } else {
            const errText = await res.text();
            showMessage('error', 'Error saving: ' + errText);
            btn.innerHTML = `Error`;
        }
        setTimeout(() => {
            btn.innerHTML = origHtml;
            btn.disabled = false;
        }, 2000);
    } catch (err) {
        showMessage('error', 'Request failed. Please try again.');
        btn.innerHTML = `Error`;
        setTimeout(() => {
            btn.innerHTML = origHtml;
            btn.disabled = false;
        }, 2000);
    } finally {
        isLoading = false;
        toggleLoader(false);
    }
}

async function pickDirectory(event) {
    try {
        const btn = event.currentTarget;
        const icon = btn.innerHTML;
        btn.innerHTML = '<div class="spinner-small" style="border-top-color: currentColor"></div>';
        btn.disabled = true;

        const res = await fetch('/api/pick-dir');
        
        btn.innerHTML = icon;
        btn.disabled = false;

        if (res.ok) {
            const dir = await res.text();
            if (dir) {
                document.getElementById('targetDirInput').value = dir;
            }
        } else {
            console.error('Failed to pick directory');
        }
    } catch (err) {
        console.error('Error picking directory', err);
        const btn = event?.currentTarget;
        if (btn) {
            btn.innerHTML = '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path></svg>';
            btn.disabled = false;
        }
    }
}
