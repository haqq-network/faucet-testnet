import { writable, derived } from 'svelte/store';

export const isAuthenticated = writable(false);
export const isGithubAuth = writable(false);
export const isRequested = writable(false);
export const lastRequestedTime = writable(null);
export const user = writable({});
export const githubUser = writable({});
export const popupOpen = writable(false);
export const error = writable();
export const tasks = writable([]);
export const loading = writable(false);

export const user_tasks = derived([tasks, user], ([$tasks, $user]) => {
  let logged_in_user_tasks = [];

  if ($user && $user.email) {
    logged_in_user_tasks = $tasks.filter((task) => task.user === $user.email);
  }

  return logged_in_user_tasks;
});
