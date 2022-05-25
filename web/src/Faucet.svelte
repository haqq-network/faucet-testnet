<script>
  import {
    onMount,
    afterUpdate,
    beforeUpdate,
    onDestroy,
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
    lastRequestedTime,
    githubUser,
    isRequested,
  } from './store';
  import Icon from '@iconify/svelte';
  import Footer from './components/Footer.svelte';

  let auth0Client;

  $: checkAccount =
    $selectedAccount || '0x0000000000000000000000000000000000000000';
  $: balance = $connected ? $web3.eth.getBalance(checkAccount) : '';

  let address = null;
  let github = null;
  let countdown = null;
  let unsubscribeRequestedTime = {};
  let faucetInfo = {
    account: '0x0000000000000000000000000000000000000000',
    network: 'testnet',
    payout: 1,
  };

  $: document.title = `ISLM ${capitalize(faucetInfo.network)} Faucet`;

  // onMount hook
  onMount(async () => {
    unsubscribeRequestedTime = lastRequestedTime.subscribe(handleRequestTime);
    console.log(unsubscribeRequestedTime);
    console.log($githubUser);
    console.log($lastRequestedTime);
    console.log($isRequested);
    if (localStorage.getItem('metaMaskConnected')) {
      await defaultEvmStores.setProvider();
      auth0Client = await auth.createClient();
    }
    if (localStorage.getItem('githubConnected')) {
      auth0Client = await auth.createClient();
      githubUser.set(await auth0Client.getUser());
      isAuthenticated.set(await auth0Client.isAuthenticated());
    }
    if ($githubUser?.nickname) {
      const response = await fetch(
        `/api/requested?github=${$githubUser?.nickname}`
      );
      const claimInfo = await response.json();
      lastRequestedTime.set(claimInfo.last_requested_time);
      if ($lastRequestedTime > 0) isRequested.set(true);
      // const res = await fetch('/api/info');
      // faucetInfo = await res.json();
    }
  });

  // onDestroy hook
  onDestroy(() => {
    countdown ?? clearInterval(countdown);
    unsubscribeRequestedTime();
  });

  // afterUpdate hook
  afterUpdate(async () => {
    unsubscribeRequestedTime = lastRequestedTime.subscribe(handleRequestTime);
    console.log($githubUser, 'afterUpdate');
    console.log($lastRequestedTime, 'afterUpdate');
    console.log($isRequested, 'afterUpdate');
    if (localStorage.getItem('metamaskWallet') !== (await userWallet())) {
      localStorage.setItem('metamaskWallet', await userWallet());
    }
    if ($githubUser?.nickname) {
      try {
        const response = await fetch(
          `/api/requested?github=${$githubUser?.nickname}`
        );
        if (!response.ok) {
          const text = await response.text();
          throw new Error(text);
        }
        const claimInfo = await response.json();
        lastRequestedTime.set(claimInfo.last_requested_time);
        if ($lastRequestedTime > 0) isRequested.set(true);
      } catch (error) {
        console.log(error.message);
      }
    }
  });

  // countdown timer
  const handleRequestTime = () => {
    if (!$lastRequestedTime) {
      return;
    }
    const nextClaimTime = $lastRequestedTime + 60 * 60 * 24;
    countdown = setInterval(() => {
      let currentTime = Math.floor(new Date().getTime() / 1000);
      const timer = nextClaimTime - currentTime;
      if (timer > 0) {
        document.getElementById('timer').innerText = `${toHHMMSS(timer)}`;
        isRequested.set(true);
      } else {
        document.getElementById('timer').innerText = '';
        isRequested.set(false);
        clearInterval(countdown);
      }
    }, 1000);
  };

  // connect metamask wallet
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

  // request tokens
  async function handleRequest() {
    try {
      address = getAddress(checkAccount);
      github = $githubUser?.nickname;
      let formData = new FormData();
      formData.append('address', address);
      formData.append('github', github);
      const response = await fetch('http://localhost:8080/api/claim', {
        method: 'POST',
        body: formData,
      });
      console.log(...formData);
      if (!response.ok) {
        const text = await response.text();
        throw new Error(text);
      } else {
        isRequested.set(true);
        bulmaToast.toast({
          duration: 3000,
          message: 'User received 1 ISLM',
          position: 'bottom-right',
          type: 'is-success',
          dismissible: true,
          pauseOnHover: true,
          closeOnClick: false,
          animate: { in: 'fadeIn', out: 'fadeOut' },
        });
      }
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
      console.log(error);
      return;
    }

    // let message = await res.text();
    // let type = res.ok ? 'is-success' : 'is-warning';
    // toast({ message, type });
  }

  // unix-timestamp to hh:mm:ss
  const toHHMMSS = (number) => {
    let sec_num = parseInt(number, 10);
    let hours = Math.floor(sec_num / 3600);
    let minutes = Math.floor((sec_num - hours * 3600) / 60);
    let seconds = sec_num - hours * 3600 - minutes * 60;
    if (hours < 10) {
      hours = '0' + hours;
    }
    if (minutes < 10) {
      minutes = '0' + minutes;
    }
    if (seconds < 10) {
      seconds = '0' + seconds;
    }
    return hours + ':' + minutes + ':' + seconds;
  };

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
    isRequested.set(false);
    lastRequestedTime(null);
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
            {#if $isAuthenticated && $connected && window.ethereum.chainId === '0x2be3' && !$isRequested}
              <button
                on:click={handleRequest}
                class="button is-medium connect "
              >
                Request Tokens
              </button>
            {:else}
              <div id="timer" />
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
