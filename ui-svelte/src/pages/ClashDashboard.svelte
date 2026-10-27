<script>
    import { onMount } from 'svelte';
    let iframeEl;

    function syncIframeTheme() {
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
        syncIframeTheme();
    });
</script>

<div class="frame-wrap">
    <iframe
        bind:this={iframeEl}
        src="./clash-dashboard/index.html"
        title="Clash Dashboard"
        class="embed-frame"
        on:load={syncIframeTheme}
    ></iframe>
</div>

<style>
    .frame-wrap {
        width: 100%;
        height: 100%;
        overflow: hidden;
        position: relative;
    }
    .embed-frame {
        width: 100%;
        height: 100%;
        border: none;
        display: block;
    }
</style>
