import { writable } from 'svelte/store';

type User = {
  email: string;
};

interface IGithubUser {
  aud: string;
  exp: number;
  iat: number;
  iss: string;
  name: string;
  nickname: string;
  picture: string;
  sub: string;
  updated_at: string;
}

export const isAuthenticated = writable(false);
export const isGithubAuth = writable(false);
export const isTokenRequested = writable(false);
export const isChecked = writable(false);
export const lastRequestedTime = writable(null);
export const user = writable<User | undefined>();
export const githubUser = writable<IGithubUser | {}>({});
export const popupOpen = writable(false);
