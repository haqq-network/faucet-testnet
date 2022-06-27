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
    chainData,
  } from 'svelte-web3';
  import { Balance } from 'svelte-web3/components';
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
  let chainId = '0xcfdb'; // TODO: load from config
  // let chainId = '0x5'; // TODO: load from config //goerli network 
  let unsubscribeRequestedTime = {};
  let faucetInfo = {
    account: '0x0000000000000000000000000000000000000000',
    network: 'testnet',
    payout: 1,
  };
  let loading = false;

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
    loading = true;
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
      loading = true;
      try {
        const response = await fetch(
          `/api/requested?github=${$githubUser?.nickname}`
        );
        loading = false;
        if (!response.ok) {
          const text = await response.text();
          throw new Error(text);
        }
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
          message: error?.message,
          type: 'is-danger',
        });
        loading = false;
        // console.log(error);
      }
    }
    loading = false;
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
        // bulmaToast.toast({
        //   message: error,
        //   type: 'is-danger',
        // });
        console.log(error);
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
      const response = await fetch('/api/claim', {
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
    loading = true;
    try {
      auth0Client = await auth.createClient();
      await auth.loginWithPopup(auth0Client);
      githubUser.set(await auth0Client.getUser());
      localStorage.setItem('githubConnected', true);
      isAuthenticated.set(await auth0Client.isAuthenticated());
      loading = false;
    } catch (error) {
      bulmaToast.toast({
        message: error.message,
        type: 'is-danger',
      });
      loading = false;
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
          params: [{ chainId: chainId }],
        });
        bulmaToast.toast({
          message: `Switched to Haqq Network Testnet successfully`,
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
                  chainId: chainId, //hexadecimal, 53211 decimal
                  chainName: 'Haqq Network Testnet',
                  nativeCurrency: {
                    name: 'IslamicCoin',
                    symbol: 'ISLMT', // 2-6 characters long
                    decimals: 18,
                  },
                  rpcUrls: ['https://rpc.eth.testedge.haqq.network/'],
                },
              ],
              // params: [ //goerli network 
              //   {
              //     chainId: chainId,
              //     chainName: 'Haqq Network goerli',
              //     nativeCurrency: {
              //       name: 'IslamicCoin',
              //       symbol: 'ISLM',
              //       decimals: 18,
              //     },
              //     rpcUrls: [
              //       'https://goerli.infura.io/v3/4caf0abc1c81486fa2985a9cab3c9497',
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
            <img alt="IslamicCoin" src="logo.svg" width="150" />
          </a>
        </div>
        <div class="navbar-item navbar-end">
          {#if window.ethereum && $connected}
            <a class="button is-hovered is-link accountButton">
              {#await hideAddress()}
                <span> waiting... </span>
              {:then hiddenAddress}
                <Icon icon="logos:metamask-icon" inline={true} class="mr-1" />
                <span class="mr-1">{hiddenAddress}</span>
              {/await}
              <span class="mr-1"> Balance: </span>
              {#await balance}
                <span> waiting... </span>
              {:then value}
                <span>{$web3.utils.fromWei(value).substring(0, 5)} ISLM</span> &nbsp
              {/await}
            </a>
          {/if}
        </div>
        <div class="navbar-item">
          {#if $connected || $isAuthenticated}
            <!-- DROPDOWN START -->
            <div class="dropdown is-hoverable is-right">
              <div class="dropdown-trigger">
                <a
                  class="button dropdown-button p-"
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
                      <Icon
                        icon="akar-icons:github-fill"
                        inline={true}
                        class="is-flex-direction-row is-align-content-center"
                      />
                      <span> Logout </span>
                    </a>
                  {/if}
                  {#if window.ethereum && $connected}
                    <a on:click={disableBrowser} class="dropdown-item m-1">
                      <Icon
                        icon="logos:metamask-icon"
                        inline={true}
                        class="is-flex-direction-row is-align-content-center"
                      />
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

          {#if window.ethereum?.chainId === chainId && $connected}
            <div class="column">
              You connected to Haqq Network
              <figure>
                <img src="haqq.svg" width="300" alt="haqqNetworkLogo" />
              </figure>
            </div>
          {:else if window.ethereum?.chainId !== chainId && $connected}
            <div class="column">
              <button
                class="button is-medium connect m-1"
                on:click={switchChain}
              >
                Switch to TestNow
              </button>
            </div>
          {/if}
          <div class:loading>
            {#if $connected && !$isAuthenticated && !loading}
              <button on:click={login} class="button connect is-medium m-1 ">
                <span class="icon">
                  <i class="fa fa-github" />
                </span>
                <span> Login </span>
              </button>
            {:else if $isAuthenticated && $connected && window.ethereum.chainId === chainId && !$isRequested}
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
        <!-- <p>
          Selected account balance = <Balance address={$selectedAccount} />
          {$chainData.nativeCurrency?.symbol}
        </p> -->
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
    border-radius: 4px;
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
  .accountButton {
    font-family: 'Arial';
    font-weight: 600;
    left: 20px;
  }
  .loading:before {
    content: '';
    position: absolute;
    top: 0;
    right: 0;
    bottom: 0;
    left: 0;
    background: #ffffffd6;
  }
  .loading:after {
    content: '';
    position: absolute;
    left: calc(50% - 44px);
    top: calc(50% - 48px);
    width: 88px;
    height: 88px;
    border: 10px solid #fff;
    border-bottom-color: blue;
    border-radius: 50%;
    box-sizing: border-box;
    animation: rotation 1s linear infinite;
  }
  @keyframes rotation {
    0% {
      transform: rotate(0deg);
    }
    100% {
      transform: rotate(360deg);
    }
  }
  .dropdown-content {
    background-color: #3e56c4;
    padding: 0.5rem;
    border-radius: 4px;
  }
  a.dropdown-item {
    background-color: #3e56c4;
    color: #ffffff;
    display: block;
    font-size: 0.875rem;
    line-height: 1.5;
    padding: 0.375rem 1rem;
    position: relative;
    font-family: 'Arial';
    font-style: normal;
    font-weight: 600;
  }
  a.dropdown-item {
    padding-right: 1rem;
    text-align: inherit;
    white-space: nowrap;
    width: auto;
    border-radius: 4px;
  }

  a.dropdown-item:hover {
    background-color: #6f88f7;
    /* color: #ffffff; */
    color: #363636;
  }

  .dropdown-button {
    background-color: #485fc7;
    color: #fff;
    border-color: #485fc7;
  }
  .dropdown-button:hover {
    background-color: #6f88f7;
    border-color: #6f88f7;
  }
</style>
