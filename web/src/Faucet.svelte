<script>
  import { onMount, afterUpdate, onDestroy } from 'svelte';
  import { getAddress } from '@ethersproject/address';
  import * as bulmaToast from 'bulma-toast';
  import 'animate.css';
  import {
    connected,
    defaultEvmStores,
    selectedAccount,
    web3,
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

  $: checkAccount =
    $selectedAccount || '0x0000000000000000000000000000000000000000';
  $: balance = $connected ? $web3.eth.getBalance(checkAccount) : '';

  let auth0Client;
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

  // default settings for popUps
  bulmaToast.setDefaults({
    duration: 1500,
    position: 'bottom-right',
    dismissible: true,
    pauseOnHover: true,
    closeOnClick: false,
    animate: { in: 'fadeIn', out: 'fadeOut' },
  });

  // onMount hook
  onMount(async () => {
    unsubscribeRequestedTime = lastRequestedTime.subscribe(handleRequestTime);
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
      try {
        const response = await fetch(
          `/api/requested?github=${$githubUser?.nickname}`
        );
        const claimInfo = await response?.json();
        let currentTime = Math.floor(new Date().getTime() / 1000);
        let nextClaimTime = claimInfo.last_requested_time + 60 * 60 * 24;
        lastRequestedTime.set(claimInfo.last_requested_time);
        currentTime >= nextClaimTime
          ? isRequested.set(false)
          : isRequested.set(true);
        // const res = await fetch('/api/info');
        // faucetInfo = await res.json();
      } catch (error) {
        bulmaToast.toast({
          message: error.message,
          type: 'is-danger',
        });
        // console.log(error);
      }
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
        } else {
          const claimInfo = await response.json();
          let currentTime = Math.floor(new Date().getTime() / 1000);
          let nextClaimTime = claimInfo.last_requested_time + 60 * 60 * 24;
          lastRequestedTime.set(claimInfo.last_requested_time);
          currentTime >= nextClaimTime
            ? isRequested.set(false)
            : isRequested.set(true);
        }
      } catch (error) {
        bulmaToast.toast({
          message: error.message,
          type: 'is-danger',
        });
        // console.log(error);
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
        clearInterval(countdown);
        isRequested.set(false);
        document.getElementById('timer').innerText = '';
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
      message: `${await userWallet()} connected successfully`,
      type: 'is-success',
    });
  };

  // disconnect Metamask wallet ONLY ON FRONT
  const disableBrowser = async () => {
    await defaultEvmStores.disconnect();
    localStorage.removeItem('metamaskWallet');
    localStorage.removeItem('metaMaskConnected');
    bulmaToast.toast({
      message: 'Wallet disconected',
      type: 'is-danger',
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
      if (!response.ok) {
        const text = await response.text();
        bulmaToast.toast({
          message: text,
          type: 'is-danger',
        });
        throw new Error(text);
      } else {
        isRequested.set(true);
        bulmaToast.toast({
          message: 'User received 1 ISLM',
          type: 'is-success',
        });
      }
    } catch (error) {
      console.log(error);
      bulmaToast.toast({
        message: error.message,
        type: 'is-danger',
      });
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
        message: error.message,
        type: 'is-danger',
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
          // chainId must be in hexadecimal numbers, hardcoded rn to haqqNetwork
          params: [{ chainId: '0x2BE3' }],
        });
        bulmaToast.toast({
          message: `Switched to Polygon Mainnet successfully`,
          type: 'is-primary',
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
              type: 'is-danger',
            });
          }
        }
        console.error(error);
        bulmaToast.toast({
          message: error,
          type: 'is-danger',
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
  <!-- svelte-ignore a11y-missing-attribute -->
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
                <a
                  class="button is-medium"
                  aria-haspopup="true"
                  aria-controls="dropdown-menu"
                >
                  <span class="icon is-small">
                    <i class="fa fa-angle-down" aria-hidden="true" />
                  </span>
                </a>
              </div>
              <div class="dropdown-menu" id="dropdown-menu" role="menu">
                <div class="dropdown-content p-1">
                  {#if $isAuthenticated}
                    <a on:click={logout} class="dropdown-item m-1">
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
                    </a>
                  {/if}
                  {#if window.ethereum && $connected}
                    <a on:click={disableBrowser} class="dropdown-item m-1">
                      <span class="icon">
                        <Icon icon="logos:metamask-icon" inline={true} />
                      </span>
                      <span> Disconnect Wallet </span>
                    </a>
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
        <div class="container p-5">
          {#if !window.ethereum}
            <div class="control">
              <button type="button" class="button connect is-medium">
                <span class="icon">
                  <Icon icon="logos:metamask-icon" inline={true} />
                </span>
                <a href="https://metamask.io/download/"> Download Metamask </a>
              </button>
            </div>
          {/if}
          {#if window.ethereum && !$connected}
            <div class="control">
              <button
                on:click={enableBrowser}
                class="button connect is-medium "
              >
                <span class="icon">
                  <Icon icon="logos:metamask-icon" inline={true} />
                </span>
                <span> Connect Wallet </span>
              </button>
            </div>
          {/if}

          {#if window.ethereum?.chainId === '0x2be3' && $connected}
            <div class="column">
              You connected to Haqq Network
              <figure>
                <img src="haqq.svg" width="300" alt="haqqNetworkLogo" />
              </figure>
            </div>
          {:else if window.ethereum?.chainId !== '0x2be3' && $connected}
            <div class="column">
              <button
                class="button is-medium connect m-1"
                on:click={switchChain}
              >
                Switch to TestNow
              </button>
            </div>
          {/if}
          {#if $connected && !$isAuthenticated}
            <button on:click={login} class="button connect is-medium m-1">
              <span class="icon">
                <i class="fa fa-github" />
              </span>
              <span> Login </span>
            </button>
          {/if}
          <div>
            {#if $isAuthenticated && $connected && window.ethereum.chainId === '0x2be3' && !$isRequested}
              <button
                on:click={handleRequest}
                class="button is-medium connect "
              >
                Request Tokens
              </button>
            {/if}
            <div id="timer" />
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
