<script>
    import { createEventDispatcher } from 'svelte';
    const dispatch = createEventDispatcher();

    let username = '';
    let password = '';
    let showPassword = false;
    let errorMessage = '';
    let loading = false;

    async function handleLogin() {
        if (!username || !password) {
            errorMessage = 'Username dan password wajib diisi!';
            return;
        }

        loading = true;
        errorMessage = '';

        try {
            const res = await fetch('/api/login', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ username, password })
            });
            const data = await res.json();
            if (data.status === 'success') {
                dispatch('loginSuccess');
            } else {
                errorMessage = data.message || 'Login gagal';
            }
        } catch (e) {
            errorMessage = 'Gagal menghubungi server';
        } finally {
            loading = false;
        }
    }
</script>

<div class="login-wrapper">
    <div class="login-card">
        <div class="brand-section">
            <div class="logo-box">
                <img src="./assets/favicon.png" alt="ProjectX" class="logo-img">
            </div>
            <h1 class="brand-title">ProjectX</h1>
            <div class="brand-subtitle">WEB UI AUTHENTICATION</div>
        </div>

        {#if errorMessage}
            <div class="error-banner">
                <i class="fa-solid fa-triangle-exclamation"></i>
                <span>{errorMessage}</span>
            </div>
        {/if}

        <form on:submit|preventDefault={handleLogin}>
            <div class="form-group">
                <label for="login-username">Username</label>
                <div class="input-box">
                    <i class="fa-solid fa-user"></i>
                    <input
                        id="login-username"
                        type="text"
                        placeholder="Masukkan username"
                        bind:value={username}
                        autocomplete="username"
                    />
                </div>
            </div>

            <div class="form-group">
                <label for="login-password">Password</label>
                <div class="input-box">
                    <i class="fa-solid fa-lock"></i>
                    <input
                        id="login-password"
                        type={showPassword ? 'text' : 'password'}
                        placeholder="Masukkan password"
                        bind:value={password}
                        autocomplete="current-password"
                    />
                    <button
                        type="button"
                        class="btn-eye"
                        on:click={() => showPassword = !showPassword}
                        tabindex="-1"
                    >
                        <i class="fa-solid {showPassword ? 'fa-eye-slash' : 'fa-eye'}"></i>
                    </button>
                </div>
            </div>

            <button type="submit" class="btn-submit" disabled={loading}>
                {#if loading}
                    <i class="fa-solid fa-spinner fa-spin"></i> Masuk...
                {:else}
                    <i class="fa-solid fa-right-to-bracket"></i> Masuk Dashboard
                {/if}
            </button>
        </form>

        <div class="footer-note">
            Default: <code>admin</code> / <code>admin</code>
        </div>
    </div>
</div>

<style>
    .login-wrapper {
        min-height: 100vh;
        display: flex;
        align-items: center;
        justify-content: center;
        padding: 20px;
        background: var(--bg);
    }

    .login-card {
        width: 100%;
        max-width: 400px;
        background: var(--paper);
        border: 2px solid var(--line);
        border-radius: 20px;
        padding: 28px 24px;
        box-shadow: var(--shadow);
    }

    .brand-section {
        display: flex;
        flex-direction: column;
        align-items: center;
        margin-bottom: 24px;
        text-align: center;
    }

    .logo-box {
        width: 64px;
        height: 64px;
        border-radius: 16px;
        border: 2px solid var(--line);
        background: var(--primary);
        box-shadow: 3px 3px 0 var(--line);
        display: flex;
        align-items: center;
        justify-content: center;
        margin-bottom: 12px;
        transform: rotate(-3deg);
    }

    .logo-img {
        width: 44px;
        height: 44px;
        object-fit: contain;
    }

    .brand-title {
        font-size: 24px;
        font-weight: 950;
        text-transform: uppercase;
        letter-spacing: -0.5px;
        line-height: 1;
        margin: 0;
    }

    .brand-subtitle {
        font-size: 9px;
        font-weight: 900;
        color: var(--muted);
        letter-spacing: 1.5px;
        margin-top: 6px;
    }

    .error-banner {
        background: var(--pink);
        color: #fff;
        border: 2px solid var(--line);
        border-radius: 10px;
        padding: 10px 14px;
        font-size: 11px;
        font-weight: 900;
        display: flex;
        align-items: center;
        gap: 8px;
        margin-bottom: 16px;
        box-shadow: 2px 2px 0 var(--line);
    }

    .form-group {
        margin-bottom: 16px;
    }

    label {
        display: block;
        font-size: 10px;
        font-weight: 900;
        text-transform: uppercase;
        letter-spacing: 0.8px;
        color: var(--muted);
        margin-bottom: 6px;
    }

    .input-box {
        position: relative;
        display: flex;
        align-items: center;
        background: var(--panel-alt);
        border: 2px solid var(--line);
        border-radius: 12px;
        padding: 0 14px;
        transition: border-color 0.15s;
    }
    .input-box:focus-within {
        border-color: var(--cyan);
        box-shadow: 2px 2px 0 var(--cyan);
    }

    .input-box i {
        color: var(--muted);
        font-size: 14px;
        margin-right: 10px;
    }

    input {
        flex: 1;
        background: transparent;
        border: none;
        padding: 12px 0;
        font-size: 13px;
        font-weight: 700;
        color: var(--ink);
        font-family: inherit;
        outline: none;
    }

    .btn-eye {
        background: transparent;
        border: none;
        cursor: pointer;
        padding: 4px;
        color: var(--muted);
    }

    .btn-submit {
        width: 100%;
        margin-top: 8px;
        padding: 14px;
        background: var(--primary);
        color: var(--primary-text);
        border: 2px solid var(--line);
        border-radius: 12px;
        font-size: 13px;
        font-weight: 950;
        text-transform: uppercase;
        letter-spacing: 0.5px;
        cursor: pointer;
        box-shadow: 3px 3px 0 var(--line);
        transition: all 0.1s ease;
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 8px;
        font-family: inherit;
    }
    .btn-submit:hover {
        transform: translate(-2px, -2px);
        box-shadow: 5px 5px 0 var(--line);
        filter: brightness(1.1);
    }
    .btn-submit:active {
        transform: translate(2px, 2px);
        box-shadow: none;
    }
    .btn-submit:disabled {
        opacity: 0.6;
        pointer-events: none;
    }

    .footer-note {
        text-align: center;
        font-size: 11px;
        font-weight: 700;
        color: var(--muted);
        margin-top: 20px;
    }

    code {
        background: var(--panel-alt);
        padding: 2px 6px;
        border-radius: 4px;
        border: 1px solid var(--line);
        color: var(--cyan);
        font-family: var(--mono);
    }
</style>
