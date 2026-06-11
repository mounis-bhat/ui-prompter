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
    
    const formData = new URLSearchParams();
    formData.append('content', pre.innerText);

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
