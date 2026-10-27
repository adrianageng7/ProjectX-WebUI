import { mount } from 'svelte'
import './app.css'
import App from './App.svelte'

// Apply theme based on localStorage
const theme = localStorage.getItem('theme') || 'dark';
const color = localStorage.getItem('color') || 'default';
document.documentElement.setAttribute('data-theme', theme);
document.documentElement.setAttribute('data-color', color);

const app = mount(App, {
  target: document.getElementById('app'),
})

export default app
