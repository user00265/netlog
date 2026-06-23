import { mount } from "svelte";
import "./app.css";
import App from "./App.svelte";
import { setupPWA } from "./lib/pwa.svelte";

const app = mount(App, { target: document.getElementById("app")! });

setupPWA();

export default app;
