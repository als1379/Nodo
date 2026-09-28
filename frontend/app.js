const API_URL = window.NODO_CONFIG?.apiUrl || "/api";
const TOKEN_KEY = "nodo_access_token";

const elements = Object.fromEntries([
  "auth-view", "auth-form", "auth-error", "auth-submit", "login-tab", "register-tab",
  "identity", "identity-label", "password", "password-hint", "dictionary-view", "account",
  "account-name", "logout", "word-search-form", "word-search-input", "lookup-loading",
  "lookup-error", "lookup-details",
].map((id) => [id.replace(/-([a-z])/g, (_, letter) => letter.toUpperCase()), document.querySelector(`#${id}`)]));

let authMode = "login";
let accessToken = sessionStorage.getItem(TOKEN_KEY);

async function request(path, options = {}, authenticated = true) {
  const headers = new Headers(options.headers || {});
  if (authenticated && accessToken) headers.set("Authorization", `Bearer ${accessToken}`);
  const response = await fetch(`${API_URL}${path}`, { ...options, headers });
  const data = response.status === 204 ? {} : await response.json().catch(() => ({}));
  if (response.status === 401 && authenticated) {
    clearSession();
    showAuth("Your session expired. Please log in again.");
  }
  if (!response.ok) throw new Error(data.error?.message || "The server could not complete the request.");
  return data;
}

function setAuthMode(mode) {
  authMode = mode;
  const registering = mode === "register";
  elements.loginTab.classList.toggle("active", !registering);
  elements.registerTab.classList.toggle("active", registering);
  elements.loginTab.setAttribute("aria-selected", String(!registering));
  elements.registerTab.setAttribute("aria-selected", String(registering));
  elements.identityLabel.textContent = registering ? "Email" : "Email or username";
  elements.identity.type = registering ? "email" : "text";
  elements.identity.autocomplete = registering ? "email" : "username";
  elements.password.autocomplete = registering ? "new-password" : "current-password";
  elements.password.minLength = registering ? 10 : 1;
  elements.passwordHint.hidden = !registering;
  elements.authSubmit.textContent = registering ? "Create account" : "Log in";
  elements.authError.hidden = true;
}

function clearSession() { accessToken = null; sessionStorage.removeItem(TOKEN_KEY); }

function showAuth(message = "") {
  elements.dictionaryView.hidden = true;
  elements.account.hidden = true;
  elements.authView.hidden = false;
  elements.authError.textContent = message;
  elements.authError.hidden = !message;
}

function showApp(user) {
  elements.authView.hidden = true;
  elements.dictionaryView.hidden = false;
  elements.account.hidden = false;
  elements.accountName.textContent = user.username || user.email;
  elements.wordSearchInput.focus();
}

async function handleAuth(event) {
  event.preventDefault();
  elements.authError.hidden = true;
  elements.authSubmit.disabled = true;
  elements.authSubmit.textContent = authMode === "register" ? "Creating account..." : "Logging in...";
  const identity = elements.identity.value.trim();
  const body = authMode === "register"
    ? { email: identity, password: elements.password.value }
    : { login: identity, password: elements.password.value };
  try {
    const result = await request(`/v1/auth/${authMode}`, {
      method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body),
    }, false);
    accessToken = result.token;
    sessionStorage.setItem(TOKEN_KEY, accessToken);
    elements.authForm.reset();
    showApp(result.user);
  } catch (error) {
    elements.authError.textContent = error.message;
    elements.authError.hidden = false;
  } finally {
    elements.authSubmit.disabled = false;
    elements.authSubmit.textContent = authMode === "register" ? "Create account" : "Log in";
  }
}

async function logout() {
  try { await request("/v1/auth/logout", { method: "POST" }); } catch (_) { /* Always clear locally. */ }
  clearSession();
  showAuth();
}
function addText(parent, tag, className, text) {
  const element = document.createElement(tag);
  if (className) element.className = className;
  element.textContent = text;
  parent.append(element);
  return element;
}

function renderDetails(details, detailsError, container = elements.details) {
  container.replaceChildren();
  container.hidden = false;
  if (!details) {
    addText(container, "p", "details-error", detailsError || "Dictionary details are unavailable.");
    return;
  }
  const entry = details.entries?.[0];
  const definition = entry?.definitions?.[0];
  if (!entry || !definition) {
    addText(container, "p", "details-error", "Learning notes are unavailable for this word.");
    return;
  }

  addText(container, "p", "details-label", "Learn this word");
  const lesson = document.createElement("section");
  lesson.className = "learning-card";
  const heading = document.createElement("div");
  heading.className = "learning-heading";
  const headingCopy = document.createElement("div");
  addText(headingCopy, "h2", "learning-word", details.word);
  addText(headingCopy, "p", "learning-pos", entry.part_of_speech || "Italian word");
  heading.append(headingCopy);
  if (entry.pronunciation?.length) addText(heading, "p", "pronunciation", entry.pronunciation[0]);
  lesson.append(heading);

  const meaning = document.createElement("div");
  meaning.className = "lesson-block primary-meaning";
  addText(meaning, "p", "lesson-kicker", "Meaning to remember");
  addText(meaning, "p", "learning-meaning", definition.meaning);
  if (details.learning?.summary) addText(meaning, "p", "learning-summary", details.learning.summary);
  lesson.append(meaning);

  if (details.learning) renderLearningGuide(lesson, details.learning, entry);
  else {
    renderLearningGrammar(lesson, entry);
    if (definition.examples?.length) renderLearningExample(lesson, definition.examples[0]);
  }
  const usefulSynonyms = definition.synonyms?.length ? definition.synonyms : entry.synonyms;
  if (usefulSynonyms?.length) addText(lesson, "p", "learning-related", `Similar: ${usefulSynonyms.slice(0, 3).map(({ word }) => word).join(", ")}`);
  if (!details.learning && details.learning_error) addText(lesson, "p", "details-error", details.learning_error);
  container.append(lesson);

  const full = document.createElement("details");
  full.className = "full-dictionary";
  addText(full, "summary", "full-dictionary-title", "Explore full dictionary details");
  const fullBody = document.createElement("div");
  fullBody.className = "full-dictionary-body";
  details.entries.forEach((entry) => {
    const section = document.createElement("section");
    section.className = "dictionary-entry";
    addText(section, "h2", "part-of-speech", entry.part_of_speech || "Italian word");
    if (entry.grammar?.length) renderChips(section, entry.grammar, "Grammar");
    if (entry.pronunciation?.length) addText(section, "p", "pronunciation", entry.pronunciation.join(" Â· "));
    if (entry.etymology) addText(section, "p", "etymology", entry.etymology);
    const list = document.createElement("ol");
    list.className = "definitions";
    entry.definitions.forEach((definition) => {
      const item = document.createElement("li");
      addText(item, "span", "definition", definition.meaning);
      if (definition.tags?.length) renderChips(item, definition.tags, "Usage");
      if (definition.topics?.length) addText(item, "p", "metadata", `Topics: ${definition.topics.join(", ")}`);
      definition.remarks?.forEach((remark) => addText(item, "p", "remark", remark));
      definition.examples?.forEach((example) => addText(item, "p", "example", `Example: ${example}`));
      renderRelations(item, "Synonyms", definition.synonyms);
      renderRelations(item, "Antonyms", definition.antonyms);
      list.append(item);
    });
    section.append(list);
    renderRelations(section, "Synonyms", entry.synonyms);
    renderRelations(section, "Antonyms", entry.antonyms);
    renderRelations(section, "Related words", entry.related);
    if (entry.forms?.length) renderForms(section, entry.forms, entry.part_of_speech);
    if (entry.categories?.length) addText(section, "p", "metadata", `Notes: ${entry.categories.join(" Â· ")}`);
    fullBody.append(section);
  });
  const source = document.createElement("a");
  source.href = details.source_url;
  source.target = "_blank";
  source.rel = "noreferrer";
  source.textContent = "Open complete entry on Wiktionary";
  fullBody.append(source);
  full.append(fullBody);
  container.append(full);
}

function renderLearningGuide(parent, guide, entry) {
  const essentials = document.createElement("div");
  essentials.className = "lesson-block essentials-focus";
  const essentialsTitle = entry.part_of_speech === "verb" ? "Verb essentials"
    : entry.part_of_speech === "noun" ? "Noun essentials"
      : ["adjective", "adj"].includes(entry.part_of_speech) ? "Agreement"
        : "How to use this word";
  addText(essentials, "p", "lesson-kicker", essentialsTitle);
  addText(essentials, "p", "grammar-note", guide.grammar_note);
  parent.append(essentials);

  if (guide.formation_rules?.length) {
    const formation = document.createElement("div");
    formation.className = "lesson-block formation-focus";
    const formationTitle = entry.part_of_speech === "verb" ? "How the forms are built"
      : entry.part_of_speech === "noun" ? "Singular and plural"
        : ["adjective", "adj"].includes(entry.part_of_speech) ? "How agreement changes the word"
          : "Useful rules";
    addText(formation, "p", "lesson-kicker", formationTitle);
    const rules = document.createElement("ul");
    rules.className = "formation-rules";
    guide.formation_rules.forEach((rule) => addText(rules, "li", "", rule));
    formation.append(rules);
    parent.append(formation);
  }

  guide.tables?.forEach((table) => {
    const block = document.createElement("section");
    block.className = "lesson-block learning-table";
    addText(block, "h3", "learning-table-title", table.title);
    if (table.note) addText(block, "p", "learning-table-note", table.note);
    const grid = document.createElement("div");
    grid.className = "learning-forms-grid";
    table.forms.forEach((form) => {
      const item = document.createElement("div");
      item.className = "learning-form";
      addText(item, "span", "learning-form-label", form.label);
      addText(item, "strong", "learning-form-value", form.value);
      grid.append(item);
    });
    block.append(grid);
    parent.append(block);
  });

  if (guide.patterns?.length) {
    const block = document.createElement("div");
    block.className = "lesson-block patterns-focus";
    addText(block, "p", "lesson-kicker", "Useful patterns");
    guide.patterns.forEach((pattern) => addText(block, "p", "pattern", pattern));
    parent.append(block);
  }

  if (guide.examples?.length) {
    const block = document.createElement("div");
    block.className = "lesson-block example-focus";
    addText(block, "p", "lesson-kicker", "See it in context");
    guide.examples.forEach((example) => {
      const pair = document.createElement("div");
      pair.className = "example-pair";
      addText(pair, "p", "example-italian", example.italian);
      addText(pair, "p", "example-translation", example.english);
      block.append(pair);
    });
    parent.append(block);
  }
}

function renderLearningGrammar(parent, entry) {
  const block = document.createElement("div");
  block.className = "lesson-block grammar-focus";
  addText(block, "p", "lesson-kicker", entry.part_of_speech === "verb" ? "How to use it" : "Form to remember");
  const grammar = entry.grammar || [];
  if (entry.part_of_speech === "noun") {
    const feminine = grammar.includes("feminine");
    const masculine = grammar.includes("masculine");
    if (feminine || masculine) addText(block, "p", "grammar-rule", `${feminine ? "la" : "il"} ${feminine ? "Â· feminine" : "Â· masculine"}`);
    const plural = entry.forms?.find((form) => form.tags?.includes("plural"));
    if (plural) addText(block, "p", "key-form", `Plural: ${plural.form}`);
  } else if (entry.part_of_speech === "verb") {
    const auxiliary = entry.forms?.find((form) => form.tags?.includes("auxiliary"));
    const participle = entry.forms?.find((form) => form.tags?.includes("past") && form.tags?.includes("participle") && (!form.tags.includes("feminine")) && (!form.tags.includes("plural")));
    const gerund = entry.forms?.find((form) => form.tags?.includes("gerund"));
    const essentials = [auxiliary && `Auxiliary: ${auxiliary.form}`, participle && `Past participle: ${participle.form}`, gerund && `Gerund: ${gerund.form}`].filter(Boolean);
    essentials.forEach((text) => addText(block, "p", "key-form", text));
    renderPresentTense(block, entry.forms || []);
  } else {
    const important = entry.forms?.filter((form) => form.tags?.some((tag) => ["feminine", "masculine", "singular", "plural"].includes(tag))).slice(0, 4);
    important?.forEach((form) => addText(block, "p", "key-form", `${form.form} Â· ${form.tags.join(" ").replaceAll("-", " ")}`));
  }
  if (grammar.length) renderChips(block, grammar, "Grammar");
  if (block.children.length > 1) parent.append(block);
}

function renderPresentTense(parent, forms) {
  const present = forms.filter((form) => form.tags?.includes("present") && form.tags?.includes("indicative") && form.tags.some((tag) => tag.endsWith("-person")));
  if (!present.length) return;
  addText(parent, "p", "mini-table-title", "Present tense");
  const pronouns = { "first-person singular": "io", "second-person singular": "tu", "third-person singular": "lui/lei", "first-person plural": "noi", "second-person plural": "voi", "third-person plural": "loro" };
  const grid = document.createElement("div");
  grid.className = "conjugation-preview";
  present.forEach((form) => {
    const person = form.tags.find((tag) => tag.endsWith("-person"));
    const number = form.tags.find((tag) => tag === "singular" || tag === "plural");
    const item = document.createElement("div");
    addText(item, "span", "pronoun", pronouns[`${person} ${number}`] || "");
    addText(item, "strong", "", form.form);
    grid.append(item);
  });
  parent.append(grid);
}

function renderLearningExample(parent, example) {
  const block = document.createElement("div");
  block.className = "lesson-block example-focus";
  addText(block, "p", "lesson-kicker", "Say it in context");
  const parts = example.split(/\s(?:â€”|-)\s/, 2);
  addText(block, "p", "example-italian", parts[0]);
  if (parts[1]) addText(block, "p", "example-translation", parts[1]);
  parent.append(block);
}

function renderChips(parent, values, label) {
  const row = document.createElement("div");
  row.className = "chips";
  row.setAttribute("aria-label", label);
  values.forEach((value) => addText(row, "span", "chip", value.replaceAll("-", " ")));
  parent.append(row);
}

function renderRelations(parent, label, words) {
  if (words?.length) addText(parent, "p", "relations", `${label}: ${words.map(({ word }) => word).join(", ")}`);
}

function renderForms(parent, forms, partOfSpeech) {
  const container = document.createElement("details");
  container.className = "forms-panel";
  const title = partOfSpeech === "verb" ? "Full conjugation" : "Inflected forms";
  addText(container, "summary", "forms-title", `${title} (${forms.length})`);
  const identityTags = new Set(["first-person", "second-person", "third-person", "singular", "plural", "masculine", "feminine"]);
  const groups = new Map();
  forms.forEach((form) => {
    const key = (form.tags?.filter((tag) => !identityTags.has(tag)) || []).join(" Â· ") || "Other forms";
    if (!groups.has(key)) groups.set(key, []);
    groups.get(key).push(form);
  });
  groups.forEach((groupForms, group) => {
    const block = document.createElement("section");
    block.className = "form-group";
    addText(block, "h3", "form-group-title", group.replaceAll("-", " "));
    const grid = document.createElement("div");
    grid.className = "forms-grid";
    groupForms.forEach((form) => {
      const item = document.createElement("div");
      item.className = "form-item";
      addText(item, "strong", "form-word", form.form);
      const labels = form.tags?.filter((tag) => identityTags.has(tag)) || [];
      if (labels.length) addText(item, "span", "form-tags", labels.join(" ").replaceAll("-", " "));
      grid.append(item);
    });
    block.append(grid);
    container.append(block);
  });
  parent.append(container);
}

async function lookupWord(event) {
  event.preventDefault();
  const word = elements.wordSearchInput.value.trim();
  if (!word) return;
  elements.lookupDetails.hidden = true;
  elements.lookupError.hidden = true;
  elements.lookupLoading.hidden = false;
  try {
    const data = await request(`/v1/dictionary/${encodeURIComponent(word)}`);
    elements.lookupLoading.hidden = true;
    renderDetails(data.details, data.details?.learning_error, elements.lookupDetails);
  } catch (error) {
    elements.lookupLoading.hidden = true;
    elements.lookupError.textContent = error.message;
    elements.lookupError.hidden = false;
  }
}

async function restoreSession() {
  if (!accessToken) { showAuth(); return; }
  try { showApp((await request("/v1/auth/me")).user); }
  catch (_) { if (accessToken) showAuth("Could not restore your session."); }
}

elements.authForm.addEventListener("submit", handleAuth);
elements.loginTab.addEventListener("click", () => setAuthMode("login"));
elements.registerTab.addEventListener("click", () => setAuthMode("register"));
elements.logout.addEventListener("click", logout);
elements.wordSearchForm.addEventListener("submit", lookupWord);
setAuthMode("login");
restoreSession();
