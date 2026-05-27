/* @refresh reload */
import { render } from "solid-js/web";
import { Router, Route } from "@solidjs/router";
import "./index.css";
import App from "./App";
import OverviewPage from "./pages/OverviewPage";
import SessionsPage from "./pages/SessionsPage";
import TimelinePage from "./pages/TimelinePage";
import MemoryPage from "./pages/MemoryPage";
import SearchPage from "./pages/SearchPage";

const root = document.getElementById("root");

render(
  () => (
    <Router root={App}>
      <Route path="/" component={OverviewPage} />
      <Route path="/sessions/:id?" component={SessionsPage} />
      <Route path="/timeline" component={TimelinePage} />
      <Route path="/memory/:id?" component={MemoryPage} />
      <Route path="/search" component={SearchPage} />
    </Router>
  ),
  root!
);
