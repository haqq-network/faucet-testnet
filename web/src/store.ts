import { writable } from 'svelte/store';

type User = {
  email: string;
};

export const isAuthenticated = writable(false);
export const isGithubAuth = writable(false);
export const isTokenRequested = writable(false);
export const isChecked = writable(false);
export const lastRequestedTime = writable(null);
export const user = writable<User | undefined>();
export const githubUser = writable({});
export const popupOpen = writable(false);
export const loadingSpinner = writable(false);
export const timer = writable('');
