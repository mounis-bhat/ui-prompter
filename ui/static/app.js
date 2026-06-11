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

function showMessage(type, text) {
    const container = document.getElementById('messages-container');
    container.innerHTML = '';
    if (!text) return;
    
    const div = document.createElement('div');
    div.className = `msg ${type}`;
    
    let icon = '';
    if (type === 'error') {
        icon = `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"></circle><line x1="12" y1="8" x2="12" y2="12"></line><line x1="12" y1="16" x2="12.01" y2="16"></line></svg>`;
    } else {
        icon = `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"></path><polyline points="22 4 12 14.01 9 11.01"></polyline></svg>`;
    }
    
    div.innerHTML = `${icon}<span>${text}</span>`;
    container.appendChild(div);
}

function clearMessages() {
    document.getElementById('messages-container').innerHTML = '';
}

function toggleLoader(show, text = "Generating Blueprint...") {
    const loader = document.getElementById('loader');
    if (show) {
        const textEl = loader.querySelector('.loader-text');
        if (textEl) textEl.innerText = text;
        loader.classList.add('active');
    } else {
        loader.classList.remove('active');
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
    if (loadingText) toggleLoader(true, loadingText);
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
        if (loadingText) toggleLoader(false);
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
        const figmaInput = document.querySelector('#form-figma input[type="url"]');
        if (figmaBtn) figmaBtn.disabled = !data.FigmaReady;
        if (figmaInput) figmaInput.disabled = !data.FigmaReady;
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

document.addEventListener('DOMContentLoaded', () => {
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
        figmaForm.addEventListener('submit', (e) => handleFormSubmit(e, 'Extracting Figma Blueprint...'));
    }
    
    const imageForm = document.getElementById('form-image');
    if (imageForm) {
        imageForm.addEventListener('submit', (e) => handleFormSubmit(e, 'Analyzing Image Blueprint...'));
    }
    
    const settingsForm = document.getElementById('form-settings');
    if (settingsForm) {
        settingsForm.addEventListener('submit', (e) => handleFormSubmit(e, null)); 
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
    const btn = document.getElementById('saveBtn');
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
        content += "\n\nNote: The extracted assets and design.png are available in the local `" + planDirName + "/assets/` directory. Please move these to the appropriate project directory and use them in the code.";
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
            btn.innerHTML = `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="20 6 9 17 4 12"></polyline></svg> Saved!`;
        } else {
            const errText = await res.text();
            alert('Error saving: ' + errText);
            btn.innerHTML = `Error`;
        }
        setTimeout(() => {
            btn.innerHTML = origHtml;
            btn.disabled = false;
        }, 2000);
    } catch (err) {
        alert('Request failed');
        btn.innerHTML = `Error`;
        btn.disabled = false;
        setTimeout(() => btn.innerHTML = origHtml, 2000);
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
