<script>
  import {
    onMount,
    afterUpdate,
    beforeUpdate,
    setContext,
    getContext,
  } from 'svelte';
  import { getAddress } from '@ethersproject/address';
  import * as bulmaToast from 'bulma-toast';
  import 'animate.css';
  import {
    chainData,
    connected,
    defaultEvmStores,
    selectedAccount,
    chainId,
    web3,
    allChainsData,
  } from 'svelte-web3';
  import auth from './authService';
  import {
    isAuthenticated,
    user,
    user_tasks,
    tasks,
    isGithubAuth,
    githubUser,
  } from './store';
  import Icon from '@iconify/svelte';
  import Footer from './components/Footer.svelte';
  import { toggle_class } from 'svelte/internal';
  // import detectEthereumProvider from '@metamask/detect-provider';

  let auth0Client;

  $: checkAccount =
    $selectedAccount || '0x0000000000000000000000000000000000000000';
  $: balance = $connected ? $web3.eth.getBalance(checkAccount) : '';

  let address = null;
  let faucetInfo = {
    account: '0x0000000000000000000000000000000000000000',
    network: 'testnet',
    payout: 1,
  };

  $: document.title = `ISLM ${capitalize(faucetInfo.network)} Faucet`;

  // onMount hook
  onMount(async () => {
    if (localStorage.getItem('metaMaskConnected')) {
      console.log('kek');
      await defaultEvmStores.setProvider();
      auth0Client = await auth.createClient();
    }
    if (localStorage.getItem('githubConnected')) {
      auth0Client = await auth.createClient();
      githubUser.set(await auth0Client.getUser());
      isAuthenticated.set(await auth0Client.isAuthenticated());
    }
    const response = await fetch(
      `http://localhost:8080/api/requested?github=${await $githubUser.nickname}`
    );
    console.log(response, '<------------ response');
    console.log($githubUser, '<------------ githubUser');

    // const res = await fetch('/api/info');
    // faucetInfo = await res.json();
  });

  // afterUpdate hook
  afterUpdate(async () => {
    if (localStorage.getItem('metamaskWallet') !== (await userWallet())) {
      localStorage.setItem('metamaskWallet', await userWallet());
    }
  });

  //connect metamask wallet
  const enableBrowser = async () => {
    await defaultEvmStores.setProvider();
    localStorage.setItem('metamaskWallet', await userWallet());
    localStorage.setItem('metaMaskConnected', $connected);
    defaultEvmStores.setProvider();
    bulmaToast.toast({
      duration: 1000,
      message: `${await userWallet()} connected successfully`,
      position: 'bottom-right',
      type: 'is-success',
      dismissible: true,
      pauseOnHover: true,
      closeOnClick: false,
      animate: { in: 'fadeIn', out: 'fadeOut' },
    });
  };

  // disconnect Metamask wallet ONLY ON FRONT
  const disableBrowser = async () => {
    await defaultEvmStores.disconnect();
    localStorage.removeItem('metamaskWallet');
    localStorage.removeItem('metaMaskConnected');
    bulmaToast.toast({
      duration: 1000,
      message: 'Wallet disconected',
      position: 'bottom-right',
      type: 'is-danger',
      dismissible: true,
      pauseOnHover: true,
      closeOnClick: false,
      animate: { in: 'fadeIn', out: 'fadeOut' },
    });
  };

  // getting Metamask wallet address
  const userWallet = async () => {
    const wallet = await window.ethereum?.selectedAddress;
    return wallet;
  };

  async function handleRequest() {
    try {
      address = getAddress(checkAccount);
    } catch (error) {
      toast({ message: error.reason, type: 'is-warning' });
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
    toast({ message, type });
  }

  function capitalize(str) {
    const lower = str.toLowerCase();
    return str.charAt(0).toUpperCase() + lower.slice(1);
  }

  // login github
  async function login() {
    try {
      auth0Client = await auth.createClient();
      await auth.loginWithPopup(auth0Client);
      githubUser.set(await auth0Client.getUser());
      localStorage.setItem('githubConnected', true);
      isAuthenticated.set(await auth0Client.isAuthenticated());
    } catch (error) {
      bulmaToast.toast({
        duration: 3000,
        message: error.message,
        position: 'bottom-right',
        type: 'is-danger',
        dismissible: true,
        pauseOnHover: true,
        closeOnClick: false,
        animate: { in: 'fadeIn', out: 'fadeOut' },
      });
    }
  }

  // logout github
  function logout() {
    auth.logout(auth0Client);
    localStorage.removeItem('githubUser');
    localStorage.removeItem('githubConnected');
    githubUser.set({});
  }

  // toggle dropdown button
  function toggle() {
    document.querySelector('.dropdown').classList.toggle('is-active');
  }

  // hide metamask middle symbols
  const hideAddress = async () => {
    let result = await userWallet();
    return `${result?.slice(0, 4)}...${result?.slice(38, 42)}`;
  };

  // detect and switch chain
  async function switchChain() {
    // Check if MetaMask is installed
    // MetaMask injects the global API into window.ethereum
    if (window.ethereum) {
      try {
        // check if the chain to connect to is installed
        await window.ethereum.request({
          method: 'wallet_switchEthereumChain',
          // chainId must be in hexadecimal numbers, hardcoded rn
          params: [{ chainId: '0x2BE3' }],
        });
        bulmaToast.toast({
          message: `Switched to Polygon Mainnet successfully`,
          position: 'bottom-right',
          type: 'is-primary',
          dismissible: true,
          pauseOnHover: true,
          closeOnClick: false,
          animate: { in: 'fadeIn', out: 'fadeOut' },
        });
      } catch (error) {
        // This error code indicates that the chain has not been added to MetaMask
        // if it is not, then install it into the user MetaMask
        if (error.code === 4902) {
          try {
            await window.ethereum.request({
              // ACTUAL NETWORK SETUP FROM https://islamiccoin.net/metamask-instruction

              method: 'wallet_addEthereumChain',
              params: [
                {
                  chainId: '0x2BE3', //hexadecimal, 11235 decimal
                  chainName: 'Haqq Network',
                  nativeCurrency: {
                    name: 'IslamicCoin',
                    symbol: 'ISLM', // 2-6 characters long
                    decimals: 18,
                  },
                  rpcUrls: ['https://rpc.eth.haqq.network'],
                },
              ],

              // method: 'wallet_addEthereumChain',
              // params: [
              //   {
              //     chainId: '0x89',
              //     chainName: 'Polygon Mainnet',
              //     nativeCurrency: {
              //       name: 'MATIC',
              //       symbol: 'MATIC', // 2-6 characters long
              //       decimals: 18,
              //     },
              //     rpcUrls: [
              //       'https://polygon-rpc.com',
              //       'https://rpc-mainnet.matic.network',
              //       'https://rpc-mainnet.maticvigil.com',
              //       'https://rpc-mainnet.matic.quiknode.pro',
              //       'https://matic-mainnet.chainstacklabs.com',
              //       'https://matic-mainnet-full-rpc.bwarelabs.com',
              //       'https://matic-mainnet-archive-rpc.bwarelabs.com',
              //       'https://poly-rpc.gateway.pokt.network/',
              //       'https://rpc.ankr.com/polygon',
              //       'https://rpc-mainnet.maticvigil.com/',
              //     ],
              //   },
              // ],
            });
          } catch (addError) {
            bulmaToast.toast({
              message: addError,
              position: 'bottom-right',
              type: 'is-danger',
              dismissible: true,
              pauseOnHover: true,
              closeOnClick: false,
              animate: { in: 'fadeIn', out: 'fadeOut' },
            });
          }
        }
        // console.error(error);
        bulmaToast.toast({
          message: error,
          position: 'bottom-right',
          type: 'is-danger',
          dismissible: true,
          pauseOnHover: true,
          closeOnClick: false,
          animate: { in: 'fadeIn', out: 'fadeOut' },
        });
      }
    } else {
      // if no window.ethereum then MetaMask is not installed
      alert(
        'MetaMask is not installed. Please consider installing it: https://metamask.io/download.html'
      );
    }
  }

  // const account = window.ethereum;
  // const accountInterval = setInterval(function () {
  //   if (window.ethereum?.accounts[0] !== account) {
  //     account = window.ethereum?.accounts[0];
  //     updateInterface();
  //   }
  // }, 100);
  // ethereum.request({ method: 'eth_requestAccounts' });
  // const provider = await detectEthereumProvider();

  // if (provider) {
  //   // From now on, this should always be true:
  //   // provider === window.ethereum
  //   startApp(provider); // initialize your app
  // } else {
  //   console.log('Please install MetaMask!');
  // }
</script>

<main>
  <section class="hero is-info is-fullheight ">
    <div class="hero-head">
      <nav class="navbar">
        <div class="navbar-brand">
          <a class="navbar-item" href=".">
            <img src="logo.svg" width="150" alt="IslamicCoin" />
          </a>
        </div>
        <div class="navbar-item navbar-end">
          <span>
            <!-- {#if !window.ethereum}
              <p class="control">
                <button type="button" class="button connect is-medium">
                  <span class="icon">
                    <Icon icon="logos:metamask-icon" inline={true} />
                  </span>
                  <a href="https://metamask.io/download/">
                    Download Metamask
                  </a>
                </button>
              </p>
            {/if} -->
            {#if window.ethereum && $connected}
              <div class="navbar-item ">
                <div class="button is-link is-rounded is-hovered accountButton">
                  {#await hideAddress()}
                    <span> waiting... </span>
                  {:then hiddenAddress}
                    <Icon icon="logos:metamask-icon" inline={true} /> &nbsp
                    {hiddenAddress} &nbsp
                  {/await}
                  <span> Balance: &nbsp</span>
                  {#await balance}
                    <span> waiting... </span>
                  {:then value}
                    <span>{$web3.utils.fromWei(value).substring(0, 5)}</span> &nbsp
                  {/await}
                  <span> ISLM </span>
                </div>
              </div>
            {/if}
          </span>
        </div>
        <div class="navbar-item">
          <!-- {#if !$isAuthenticated}
            <button on:click={login} class="button connect is-medium">
              <span class="icon">
                <i class="fa fa-github" />
              </span>
              <span> Login </span>
            </button>
          {/if} -->
          {#if $connected || $isAuthenticated}
            <!-- DROPDOWN START -->
            <div class="dropdown is-hoverable is-right">
              <div class="dropdown-trigger">
                <button
                  class="button is-medium"
                  aria-haspopup="true"
                  aria-controls="dropdown-menu"
                >
                  <span class="icon is-small">
                    <i class="fa fa-angle-down" aria-hidden="true" />
                  </span>
                </button>
              </div>
              <div class="dropdown-menu" id="dropdown-menu" role="menu">
                <div class="dropdown-content ">
                  {#if $isAuthenticated}
                    <button on:mousedown={logout} class="dropdown-item">
                      {#if !$githubUser.picture}
                        <span> waiting for pic... </span>
                      {:else}
                        <img
                          src={$githubUser.picture}
                          alt="avatar"
                          class="avatar icon"
                        />
                      {/if}
                      <span class="icon">
                        <i class="fa fa-github" />
                      </span>
                      <span> Logout </span>
                    </button>
                  {/if}
                  {#if window.ethereum && $connected}
                    <button on:mousedown={disableBrowser} class="dropdown-item">
                      <span class="icon">
                        <Icon icon="logos:metamask-icon" inline={true} />
                      </span>
                      <span> Disconnect Wallet </span>
                    </button>
                  {/if}
                </div>
              </div>
            </div>
          {/if}
          <!-- DROPDOWN END -->
        </div>
      </nav>
    </div>

    <div class="hero-body has-text-centered is-justify-content-center">
      <div
        class=" has-text-weight-bold is-size-3 is-align-content-center glass"
      >
        <div class="container p-6">
          {#if !window.ethereum}
            <p class="control">
              <button type="button" class="button connect is-medium">
                <span class="icon">
                  <Icon icon="logos:metamask-icon" inline={true} />
                </span>
                <a href="https://metamask.io/download/"> Download Metamask </a>
              </button>
            </p>
          {/if}
          {#if window.ethereum && !$connected}
            <p class="control">
              <button
                on:click={enableBrowser}
                class="button connect is-medium "
              >
                <span class="icon">
                  <Icon icon="logos:metamask-icon" inline={true} />
                </span>
                <span> Connect Wallet </span>
              </button>
            </p>
          {/if}

          {#if $connected && !$isAuthenticated}
            <!-- CHAIN ID CHECK -->
            <figure>
              <button on:click={login} class="button connect is-medium m-1">
                <span class="icon">
                  <i class="fa fa-github" />
                </span>
                <span> Login </span>
              </button>
            </figure>
          {/if}
          {#if window.ethereum?.chainId === '0x2be3' && $connected}
            <figure>
              Current network is Haqq Network
              <!-- {$chainData.name} -->
              <img
                src="haqq.svg"
                style="width: 300px;"
                alt="Chain logo"
                class="m-auto"
              />
            </figure>
          {:else if window.ethereum?.chainId !== '0x2be3'}
            <button class="button is-medium connect m-1" on:click={switchChain}>
              Switch to TestNow
            </button>
          {/if}
          <!-- CHAIN ID CHECK -->
          <!-- <p>
              Selected account: {$selectedAccount || 'not defined'}
            </p> -->

          <div>
            {#if $isAuthenticated && $connected && window.ethereum.chainId === '0x2be3'}
              <button
                on:click={handleRequest}
                class="button is-medium connect "
              >
                Request Tokens
              </button>
            {/if}
          </div>
        </div>
      </div>
    </div>
    <Footer />
  </section>
</main>

<style>
  .hero.is-info {
    background: linear-gradient(rgba(255, 255, 255, 0), rgba(0, 0, 0, 0)),
      url('/backgroundd2.jpg') no-repeat center center fixed;
    -webkit-background-size: cover;
    -moz-background-size: cover;
    -o-background-size: cover;
    background-size: cover;
    font-family: 'Arial';
  }
  .glass {
    font-family: 'Arial';
    line-height: 140%;
    color: #010504;
    background: linear-gradient(
      135deg,
      rgba(255, 255, 255, 0.1),
      rgba(255, 255, 255, 0)
    );
    backdrop-filter: blur(3px);
    -webkit-backdrop-filter: blur(10px);
    border-radius: 10px;
    border: 1px solid rgba(255, 255, 255, 0.18);
    box-shadow: 0 8px 32px 0 rgba(0, 0, 0, 0.37);
  }
  .connect {
    font-family: 'Arial';
    font-style: normal;
    font-weight: 500;
    font-size: 16px;
    line-height: 140%;
    text-align: center;
    letter-spacing: 0.02em;
    background-color: #28ff98;
    color: #010504;
  }

  .avatar {
    border-radius: 50%;
  }
  .accountButton {
    color: #010504;
    font-family: 'Arial';
    font-style: normal;
    font-weight: 600;
    font-size: 16px;
    line-height: 140%;
    display: flex;
    justify-content: center;
    align-items: center;
    height: 40px;
  }
</style>
