function showTab(event, tabId) {
    document.querySelectorAll('.tab-content').forEach(el => el.classList.remove('active'));
    document.querySelectorAll('.tab').forEach(el => el.classList.remove('active'));
    
    document.getElementById(tabId).classList.add('active');
    if (event) {
        event.currentTarget.classList.add('active');
    }
    
    if (window.history.replaceState) {
        const url = new URL(window.location);
        url.searchParams.delete('success');
        window.history.replaceState({path: url.href}, '', url.href);
        document.querySelectorAll('.success-message, .error-message').forEach(el => el.style.display = 'none');
    }
}

async function copyToClipboard() {
    const pre = document.querySelector('.result-box');
    if (!pre) return;
    try {
        await navigator.clipboard.writeText(pre.innerText);
        const btn = document.getElementById('copyBtn');
        const orig = btn.innerText;
        btn.innerText = 'Copied!';
        setTimeout(() => btn.innerText = orig, 2000);
    } catch (err) {
        console.error('Failed to copy', err);
    }
}

async function saveIntent() {
    const pre = document.querySelector('.result-box');
    if (!pre) return;
    const btn = document.getElementById('saveBtn');
    
    let content = pre.innerText;
    
    const attachImage = document.getElementById('attachImage');
    const nukeDir = document.getElementById('nukeDir');
    const imageHash = document.getElementById('imageHash');
    const imageExt = document.getElementById('imageExt');
    
    if (attachImage && attachImage.checked) {
        content += "\n\nI have attached the original image for more context.";
    }
    
    if (nukeDir && nukeDir.checked) {
        content += "\n\nNuke after implementation: true";
    }

    const formData = new URLSearchParams();
    formData.append('content', content);
    
    if (attachImage && attachImage.checked && imageHash) {
        formData.append('attach_image', 'true');
        formData.append('image_hash', imageHash.value);
        formData.append('image_ext', imageExt.value);
    }

    const downloadAssets = document.getElementById('downloadAssets');
    const figmaAssets = document.getElementById('figmaAssets');
    
    if (downloadAssets && downloadAssets.checked && figmaAssets) {
        formData.append('figma_assets', figmaAssets.value);
        content += "\n\nNote: The extracted assets and design.png are available in the local `assets/` directory. Please move these to the appropriate project directory and use them in the code.";
    }

    try {
        const orig = btn.innerText;
        btn.innerText = 'Saving...';
        const res = await fetch('/api/save', {
            method: 'POST',
            body: formData
        });
        
        if (res.ok) {
            btn.innerText = 'Saved!';
        } else {
            const errText = await res.text();
            alert('Error saving: ' + errText);
            btn.innerText = 'Error';
        }
        setTimeout(() => btn.innerText = orig, 2000);
    } catch (err) {
        alert('Request failed');
        btn.innerText = 'Error';
    }
}

document.addEventListener('DOMContentLoaded', () => {
    const fileInput = document.querySelector('.hidden-file-input');
    const fileText = document.querySelector('.file-upload-text');
    
    if (fileInput && fileText) {
        fileInput.addEventListener('change', function() {
            if (this.files && this.files.length > 0) {
                fileText.textContent = this.files[0].name;
                fileText.style.color = 'var(--text-primary)';
            } else {
                fileText.textContent = 'Choose a screenshot or drag it here';
                fileText.style.color = 'var(--text-secondary)';
            }
        });
    }
});

async function pickDirectory(event) {
    try {
        const btn = event.currentTarget;
        const orig = btn.innerText;
        btn.innerText = 'Opening...';
        btn.disabled = true;

        const res = await fetch('/api/pick-dir');
        
        btn.innerText = orig;
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
            btn.innerText = 'Browse';
            btn.disabled = false;
        }
    }
}
