import { isChecked, isTokenRequested, lastRequestedTime, timer } from './store';
import * as bulmaToast from 'bulma-toast';

type githubUser = {
  nickname: string;
};

export let countdown: ReturnType<typeof setInterval> | undefined | number =
  null;

async function githubUserRequest(value: githubUser | undefined) {
  if (value?.nickname) {
    try {
      isChecked.set(false);
      const response = await fetch(`/api/requested?github=${value?.nickname}`);
      if (!response.ok) {
        const text = await response.text();
        bulmaToast.toast({
          message: text,
          type: 'is-danger',
        });
        throw new Error(text);
      } else {
        const claimInfo = await response.json();
        let currentTime = Math.floor(new Date().getTime() / 1000);
        let nextClaimTime = claimInfo.last_requested_time + 60 * 60 * 24;
        lastRequestedTime.set(claimInfo.last_requested_time);
        currentTime >= nextClaimTime
          ? isTokenRequested.set(false)
          : isTokenRequested.set(true);
      }
    } catch (error) {
      bulmaToast.toast({
        message: error.text,
        type: 'is-danger',
      });
    } finally {
      isChecked.set(true);
    }
  }
}

const setTimer = (nextClaimTime: number) => {
  let currentTime = Math.floor(new Date().getTime() / 1000);
  const countdownTimer = nextClaimTime - currentTime;
  if (countdownTimer > 0) {
    timer.set(toHHMMSS(countdownTimer));
    isTokenRequested.set(true);
    // document.getElementById('timer').innerText = `${toHHMMSS(timer)}`;
  } else {
    isTokenRequested.set(false);
    clearInterval(countdown);
    // document.getElementById('timer').innerText = '';
  }
};

const handleRequestTime = (value: number | undefined) => {
  if (!value) {
    clearInterval(countdown);
    return;
  }
  const nextClaimTime = value + 60 * 60 * 24;
  setTimer(nextClaimTime);
  countdown = setInterval(() => setTimer(nextClaimTime), 1000);
};

// unix-timestamp to hh:mm:ss
const toHHMMSS = (number: number) => {
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

const utils = {
  githubUserRequest,
  handleRequestTime,
  toHHMMSS,
};

export default utils;
