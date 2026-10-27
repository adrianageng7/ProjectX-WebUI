<script>
    import { onMount } from 'svelte';

    let iframeEl;
    let iframeSrc = './filemanager.html';

    function getQueryString() {
        const fullUrl = window.location.href;
        if (fullUrl.includes('?')) {
            return fullUrl.substring(fullUrl.indexOf('?') + 1);
        }
        return '';
    }

    function updateIframe() {
        const qs = getQueryString();
        const t = Date.now();
        const targetSrc = qs ? `./filemanager.html?${qs}&v=${t}` : `./filemanager.html?v=${t}`;

        if (iframeEl && iframeEl.contentWindow && iframeEl.contentWindow.loadDir) {
            const params = new URLSearchParams(qs);
            const edit = params.get('edit');
            const dir = params.get('dir');
            if (edit && iframeEl.contentWindow.openEditor) {
                iframeEl.contentWindow.openEditor(edit);
            } else if (dir) {
                iframeEl.contentWindow.loadDir(dir);
            }
        } else {
            iframeSrc = targetSrc;
        }
        syncTheme();
    }

    function syncTheme() {
        try {
            if (iframeEl && iframeEl.contentDocument) {
                const theme = document.documentElement.getAttribute('data-theme') || 'dark';
                const color = document.documentElement.getAttribute('data-color') || 'default';
                iframeEl.contentDocument.documentElement.setAttribute('data-theme', theme);
                iframeEl.contentDocument.documentElement.setAttribute('data-color', color);
            }
        } catch (e) {}
    }

    onMount(() => {
        updateIframe();
        const onHashChange = () => {
            if (window.location.hash.includes('/filemanager')) {
                updateIframe();
            }
        };
        window.addEventListener('hashchange', onHashChange);
        return () => window.removeEventListener('hashchange', onHashChange);
    });
</script>

<div class="frame-wrap">
    <iframe
        bind:this={iframeEl}
        src={iframeSrc}
        title="File Manager"
        class="embed-frame"
        allow="clipboard-read; clipboard-write"
        on:load={syncTheme}
    ></iframe>
</div>

<style>
    .frame-wrap {
        width: 100%;
        height: 100%;
        overflow: hidden;
        background: var(--bg);
    }
    .embed-frame {
        width: 100%;
        height: 100%;
        border: none;
        display: block;
    }
</style>
