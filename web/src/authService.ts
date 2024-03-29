import createAuth0Client from '@auth0/auth0-spa-js';
import { isAuthenticated, popupOpen, isGithubAuth, githubUser } from './store';
import config from '../gh-auth.config';
import * as bulmaToast from 'bulma-toast';

async function createClient() {
  let auth0Client = await createAuth0Client({
    domain: config.domain,
    client_id: config.clientId,
  });
  return auth0Client;
}

async function githubData(auth0Client) {
  const data = await auth0Client?.getUser();
  return data || {};
}

async function loginWithPopup(client, options) {
  popupOpen.set(true);
  try {
    await client.loginWithPopup(options);
    githubUser.set(await client.getUser());
    isAuthenticated.set(true);
    bulmaToast.toast({
      duration: 3000,
      message: `hello ${await client
        .getUser()
        .then((data) => data.nickname)}, you've loggined through github`,
      position: 'bottom-right',
      type: 'is-success',
      dismissible: true,
      pauseOnHover: true,
      closeOnClick: false,
      animate: { in: 'fadeIn', out: 'fadeOut' },
    });
    localStorage.setItem(
      'githubUser',
      await client.getUser().then((data) => data.nickname),
    );
    isGithubAuth.set(true);
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
  } finally {
    popupOpen.set(false);
  }
}

async function logout() {
  // return client.logout();
  location.href = '/api/logout';
}

const auth = {
  createClient,
  loginWithPopup,
  logout,
  githubData,
};

export default auth;
