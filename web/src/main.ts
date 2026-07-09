import { mount } from "svelte";
import App from "./app/App.svelte";
import "./app.css";

const target = document.getElementById("app");
if (!target) {
  throw new Error("#app root missing");
}

mount(App, { target });
