<script>
    import {onMount} from 'svelte';
    import {getAddress} from '@ethersproject/address';
    import {setDefaults as setToast, toast} from 'bulma-toast';
    import {chainData, chainId, connected, defaultEvmStores, selectedAccount, web3} from 'svelte-web3'
    const enableBrowser = () => defaultEvmStores.setBrowserProvider()

    import auth from "./authService";
    import { isAuthenticated, user, user_tasks, tasks } from "./store";

    let auth0Client;

    $: checkAccount = $selectedAccount || '0x0000000000000000000000000000000000000000'
    $: balance = $connected ? $web3.eth.getBalance(checkAccount) : ''

    let address = null;
    let faucetInfo = {
        account: '0x0000000000000000000000000000000000000000',
        network: 'testnet',
        payout: 1,
    };

    $: document.title = `ISLM ${capitalize(faucetInfo.network)} Faucet`;

    onMount(async () => {
        auth0Client = await auth.createClient();

        isAuthenticated.set(await auth0Client.isAuthenticated());
        user.set(await auth0Client.getUser());


        const res = await fetch('/api/info');
        faucetInfo = await res.json();
    });

    setToast({
        position: 'bottom-center',
        dismissible: true,
        pauseOnHover: true,
        closeOnClick: false,
        animate: {in: 'fadeIn', out: 'fadeOut'},
    });

    async function handleRequest() {
        try {
            address = getAddress(checkAccount);
        } catch (error) {
            toast({message: error.reason, type: 'is-warning'});
            return;
        }

        let formData = new FormData();
        formData.append('address', address);
        const res = await fetch('/api/claim', {
            method: 'POST',
            body: formData,
        });
        let message = await res.text();
        let type = res.ok ? 'is-success' : 'is-warning';
        toast({message, type});
    }

    function capitalize(str) {
        const lower = str.toLowerCase();
        return str.charAt(0).toUpperCase() + lower.slice(1);
    }

    function login() {
        auth.loginWithPopup(auth0Client);
    }

    function logout() {
        auth.logout(auth0Client);
    }
</script>

<main>
    <section class="hero is-info is-fullheight">
        <div class="hero-head">
            <nav class="navbar">
                <div class="container">
                    <div class="navbar-brand">
                        <a class="navbar-item" href=".">
              <span class="icon">
                <i class="fa fa-bath"></i>
              </span>
                            <span><b>ISLM Faucet</b></span>
                        </a>
                    </div>
                </div>
                <div class="container">
                    <div class="navbar-item">
                        <span>
                                                    {#if $web3.version}
                            <p class="control">
                                <button
                                        on:click={enableBrowser}
                                        class="button is-primary is-rounded"
                                >
                                    Connect Wallet
                                </button>
                            </p>
                        {/if}
                        </span>
                    </div>
                </div>
                <div class="container">
                    <div class="navbar-item">
                        <span>
                            <p class="control">
                                <button
                                        on:click={login}
                                        class="button is-primary is-rounded"
                                >
                                    Login Github
                                </button>
                            </p>
                        </span>
                    </div>
                </div>
            </nav>
        </div>

        <div class="hero-body">
            <div class="container has-text-centered">
                <div class="column is-6 is-offset-3">
                    {#if $connected}
                        <p>
                            Connected chain: chainId = {$chainId}
                        </p>
                        <p>
                            Selected account: {$selectedAccount || 'not defined'}
                        </p>
                        <p>
                            Balance:
                            {#await balance}
                                <span>waiting...</span>
                            {:then value}
                                <span>{$web3.utils.fromWei(value).substring(0, 4)}</span>
                            {/await} ISLM
                        </p>
                    {/if}
                    <p></p>
                    <h2 class="title">
                        Receive {faucetInfo.payout} ISLM per request
                    </h2>
                    <div>
                        <p class="control">
                            <button
                                    on:click={handleRequest}
                                    class="button is-primary is-rounded"
                            >
                                Request Tokens
                            </button>
                        </p>
                    </div>
                </div>
            </div>
        </div>
    </section>
</main>

<style>
    .hero.is-info {
        background: linear-gradient(rgba(0, 0, 0, 0.5), rgba(0, 0, 0, 0.5)),
        url('/background.webp') no-repeat center center fixed;
        -webkit-background-size: cover;
        -moz-background-size: cover;
        -o-background-size: cover;
        background-size: cover;
    }
</style>
