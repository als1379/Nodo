const API_URL = window.NODO_CONFIG?.apiUrl || "/api";
const TOKEN_KEY = "nodo_access_token";
const elements = Object.fromEntries([...document.querySelectorAll("[id]")].map(element => [element.id.replace(/-([a-z])/g, (_, letter) => letter.toUpperCase()), element]));
let authMode = "login";
let accessToken = sessionStorage.getItem(TOKEN_KEY);
let generation = 0;
let accountGeneration = 0;
let view = "home";
let session = null;
let cardIndex = 0;
let answerSaved = false;
let savingAnswer = false;
let startingSession = false;
let overview = null;
let libraryWords = [];
let dictionaryWord = "";
let dictionaryGeneration = 0;
const requests = new Set();

async function request(path, options = {}, authenticated = true) {
  const token = accessToken;
  const controller = new AbortController();
  requests.add(controller);
  const timer = setTimeout(() => controller.abort(), path.includes("dictionary") ? 85000 : 15000);
  const headers = new Headers(options.headers || {});
  if (authenticated && token) headers.set("Authorization", `Bearer ${token}`);
  try {
    const response = await fetch(`${API_URL}${path}`, { ...options, headers, signal: controller.signal });
    const data = response.status === 204 ? {} : await response.json().catch(() => ({}));
    if (response.status === 401 && authenticated && token === accessToken) showAuth("Your session expired. Please log in again.");
    if (!response.ok) throw new Error(data.error?.message || "The request could not be completed. Please try again.");
    return data;
  } catch (error) {
    if (error.name === "AbortError") throw new Error("The request was interrupted. Please try again.");
    throw error;
  } finally { clearTimeout(timer); requests.delete(controller); }
}

// Separate queues let dictionary meanings load while the slower teaching model works.
function createWordQueue(concurrency, suffix, needsLesson) {
  const cache = new Map();
  const pending = new Map();
  const jobs = [];
  let active = 0;
  function pump() {
    while (active < concurrency && jobs.length) {
      const job = jobs.shift();
      active++;
      request(`/v1/dictionary/${encodeURIComponent(job.word)}${suffix}`)
        .then(data => {
          if (job.epoch !== accountGeneration) throw new Error("Session changed");
          if (needsLesson && !data.details.learning) throw new Error(data.details.learning_error || "The word lesson is unavailable. Please try again.");
          if (cache.size >= 250) cache.delete(cache.keys().next().value);
          cache.set(job.word, data.details);
          job.resolve(data.details);
        }).catch(job.reject).finally(() => {
          if (pending.get(job.word) === job.promise) pending.delete(job.word);
          active--;
          pump();
        });
    }
  }
  return {
    get(word, priority = true) {
      word = word.trim().normalize("NFC").toLocaleLowerCase("it");
      if (cache.has(word)) return Promise.resolve(cache.get(word));
      if (pending.has(word)) {
        const index = jobs.findIndex(job => job.word === word);
        if (priority && index > 0) jobs.unshift(...jobs.splice(index, 1));
        return pending.get(word);
      }
      let resolve, reject;
      const promise = new Promise((yes, no) => { resolve = yes; reject = no; });
      const job = { word, epoch: accountGeneration, resolve, reject, promise };
      pending.set(word, promise);
      if (priority) jobs.unshift(job); else jobs.push(job);
      pump();
      return promise;
    },
    clear() {
      cache.clear(); pending.clear();
      jobs.splice(0).forEach(job => job.reject(new Error("Session changed")));
    }
  };
}
const meanings = createWordQueue(4, "?view=preview", false);
const lessons = createWordQueue(1, "", true);

function setAuthMode(mode) {
  authMode = mode;
  const registering = mode === "register";
  elements.loginTab.classList.toggle("active", !registering);
  elements.registerTab.classList.toggle("active", registering);
  elements.loginTab.setAttribute("aria-selected", String(!registering));
  elements.registerTab.setAttribute("aria-selected", String(registering));
  elements.identityLabel.textContent = registering ? "Email" : "Email or username";
  elements.identity.type = registering ? "email" : "text";
  elements.password.autocomplete = registering ? "new-password" : "current-password";
  elements.password.minLength = registering ? 10 : 1;
  elements.passwordHint.hidden = !registering;
  elements.authSubmit.textContent = registering ? "Create account" : "Log in";
  elements.authError.hidden = true;
}

function showAuth(message = "") {
  generation++; accountGeneration++; dictionaryGeneration++;
  requests.forEach(controller => controller.abort());
  meanings.clear(); lessons.clear();
  accessToken = null; sessionStorage.removeItem(TOKEN_KEY);
  session = null; overview = null; libraryWords = [];
  elements.learningView.hidden = true; elements.account.hidden = true; elements.authView.hidden = false;
  elements.dictionaryDialog.close(); elements.settingsDialog.close();
  elements.authError.textContent = message; elements.authError.hidden = !message;
}

async function showApp(user) {
  elements.authView.hidden = true; elements.learningView.hidden = false; elements.account.hidden = false;
  elements.accountName.textContent = user.username || user.email;
  await showHome();
}

async function handleAuth(event) {
  event.preventDefault();
  elements.authError.hidden = true; elements.authSubmit.disabled = true;
  const identity = elements.identity.value.trim();
  try {
    const body = authMode === "register" ? { email: identity, password: elements.password.value } : { login: identity, password: elements.password.value };
    const result = await request(`/v1/auth/${authMode}`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) }, false);
    accessToken = result.token; sessionStorage.setItem(TOKEN_KEY, accessToken);
    elements.authForm.reset(); await showApp(result.user);
  } catch (error) { elements.authError.textContent = error.message; elements.authError.hidden = false; }
  finally { elements.authSubmit.disabled = false; }
}

function setView(next) {
  generation++; view = next;
  for (const [key, name] of [["studyHome", "home"], ["flashcard", "study"], ["sessionSummary", "summary"], ["wordLibrary", "library"]]) elements[key].hidden = next !== name;
  elements.cardsError.hidden = true; elements.cardsLoading.hidden = true;
  return generation;
}
function showError(error, epoch = generation) {
  if (epoch !== generation || !accessToken) return;
  elements.cardsError.textContent = error.message; elements.cardsError.hidden = false;
}
function reviewTime(due) {
  const time = new Date(due);
  if (!Number.isFinite(time.getTime())) return "Review scheduled";
  const minutes = Math.ceil((time - new Date()) / 60000);
  if (minutes <= 0) return "Ready to review";
  if (minutes < 60) return `Review in ${minutes} min`;
  if (minutes < 1440) return `Review in ${Math.ceil(minutes / 60)} hours`;
  return `Review on ${time.toLocaleDateString(undefined, { month: "short", day: "numeric" })}`;
}
async function showHome() {
  const epoch = setView("home");
  elements.startLesson.disabled = true; elements.startReview.disabled = true;
  try {
    const data = await request("/v1/flashcards/overview");
    if (epoch !== generation) return;
    overview = data;
    elements.statPracticed.textContent = data.practiced_24h;
    elements.statLearning.textContent = data.learning;
    elements.statLearned.textContent = data.learned;
    elements.reviewCopy.textContent = data.due ? `${data.due} words are ready for review. Strengthen your vocabulary before adding more.` : data.review_available ? `${data.review_available} words are still growing. Give them another practice, with no new vocabulary.` : "Your review queue starts after you practice your first words.";
    elements.startReview.disabled = data.review_available === 0;
    elements.startLesson.disabled = false;
    elements.nextReview.textContent = data.next_review ? `Next scheduled practice: ${reviewTime(data.next_review).toLowerCase()}.` : "Begin with a short practice. Your progress is saved as you go.";
  } catch (error) { showError(error, epoch); if(epoch===generation) elements.startLesson.disabled = false; }
}

async function loadCards(mode) {
  if (startingSession) return;
  startingSession = true;
  const epoch = setView("loading");
  elements.cardsLoading.hidden = false;
  elements.loadingCopy.textContent = mode === "review" ? "Gathering words to review..." : "Preparing your practice...";
  try {
    const data = await request("/v1/flashcards/sessions", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ mode }) });
    if (epoch !== generation) return;
    session = data; cardIndex = 0;
    if (!session.cards?.length) throw new Error("No words are available right now. Return to your practice later.");
    session.cards.forEach(card => { void meanings.get(card.word, false).catch(() => {}); });
    setView("study"); renderCard();
  } catch (error) {
    if (epoch === generation) { setView("home"); showError(error); }
  } finally { startingSession = false; }
}
function currentCard() { return session?.cards[cardIndex]; }
function isCurrent(epoch, card) { return epoch === generation && view === "study" && currentCard() === card; }

function renderCard() {
  const card = currentCard(); if (!card) return;
  answerSaved = false; savingAnswer = false;
  elements.cardsError.hidden = true;
  elements.cardProgress.textContent = `${session.mode === "review" ? "Review" : "Practice"} · ${session.answered + 1} of ${session.total}`;
  elements.sessionProgress.max = session.total; elements.sessionProgress.value = session.answered;
  elements.cardStatus.textContent = card.review ? "Review" : "New";
  elements.cardStatus.className = `card-status ${card.review ? "review" : "new"}`;
  elements.cardDifficulty.textContent = card.difficulty_score == null ? "" : `Difficulty ${Math.round(card.difficulty_score)} / 100`;
  elements.cardWord.textContent = card.word; elements.cardPos.textContent = card.part_of_speech;
  elements.cardAnswer.hidden = true; elements.answerActions.hidden = true; elements.answerFeedback.hidden = true;
  elements.cardMeaning.textContent = ""; elements.cardGrammar.textContent = ""; elements.cardExample.hidden = true;
  elements.revealCard.hidden = false; elements.revealCard.disabled = false; elements.revealCard.textContent = "Reveal meaning";
  elements.dontKnow.disabled = false; elements.know.disabled = false;
  elements.cardInstruction.textContent = card.review ? "Bring the meaning to mind. Then reveal it to check." : "Do you already know this word? Try recalling its meaning.";
  elements.listenWord.hidden = !("speechSynthesis" in window);
  elements.lessonStatus.textContent = "";
  // Prepare only the next few teaching lessons; all quick meanings are already queued.
  session.cards.slice(cardIndex, cardIndex + 4).forEach(item => { void lessons.get(item.word, false).catch(() => {}); });
  void meanings.get(card.word).catch(() => {});
  elements.revealCard.focus({ preventScroll: true });
}

function fillAnswer(details, card, enhanced = false) {
  const entry = details.entries?.find(entry => entry.part_of_speech === card.part_of_speech) || details.entries?.[0];
  const definition = entry?.definitions?.[0];
  const meaning = details.learning?.meaning || definition?.meaning;
  if (!meaning) throw new Error("No meaning is available for this word yet. Please retry.");
  elements.cardMeaning.textContent = meaning;
  elements.cardGrammar.textContent = details.learning?.grammar_note || (entry?.grammar || []).join(" \u00b7 ");
  const example = details.learning?.examples?.[0];
  const raw = definition?.examples?.[0];
  const italian = example?.italian || (raw ? raw.split(/\s(?:\u2014|-)\s/, 2)[0] : "");
  const english = example?.english || (raw ? raw.split(/\s(?:\u2014|-)\s/, 2)[1] || "" : "");
  elements.cardExample.hidden = !italian;
  elements.cardExampleIt.replaceChildren();
  if (italian) addItalianTokens(elements.cardExampleIt, italian);
  elements.cardExampleEn.textContent = english;
  elements.lessonStatus.textContent = enhanced ? "Try saying your own sentence with this word." : "Extra examples and learning notes are being prepared.";
}
async function revealCard() {
  const card = currentCard(), epoch = generation;
  if (!card || view !== "study" || answerSaved || elements.revealCard.disabled) return;
  elements.revealCard.disabled = true; elements.revealCard.textContent = "Fetching meaning..."; elements.cardsError.hidden = true;
  try {
    const details = await meanings.get(card.word);
    if (!isCurrent(epoch, card)) return;
    fillAnswer(details, card, !!details.learning);
    elements.cardAnswer.hidden = false; elements.answerActions.hidden = false; elements.revealCard.hidden = true;
    if (!details.learning) {
      lessons.get(card.word).then(lesson => { if(isCurrent(epoch, card)) fillAnswer(lesson, card, true); })
        .catch(() => { if(isCurrent(epoch, card)) elements.lessonStatus.textContent = "Extra learning notes are unavailable. Open the word lesson to retry."; });
    }
  } catch (error) {
    if (isCurrent(epoch, card)) { showError(error); elements.revealCard.textContent = "Retry meaning"; }
  } finally { if(isCurrent(epoch, card)) elements.revealCard.disabled = false; }
}
async function answerCard(known) {
  const card = currentCard(), epoch = generation, savedSession = session;
  if (!card || view !== "study" || elements.cardAnswer.hidden || answerSaved || savingAnswer) return;
  savingAnswer = true; elements.dontKnow.disabled = true; elements.know.disabled = true;
  try {
    const progress = await request(`/v1/flashcards/sessions/${savedSession.id}/answers`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ word_id: card.id, known }) });
    if (!isCurrent(epoch, card)) return;
    answerSaved = true; savedSession.answered++; if(known) savedSession.recalled++;
    elements.sessionProgress.value = savedSession.answered;
    elements.answerActions.hidden = true; elements.answerFeedback.hidden = false;
    elements.feedbackCopy.textContent = `${known ? "Remembered. Keep building that connection." : "Still learning - take a moment with the meaning and example."} ${reviewTime(progress.due)}.`;
    elements.nextCard.textContent = cardIndex + 1 === session.cards.length ? "See my progress" : "Next word";
    elements.nextCard.focus({ preventScroll: true });
  } catch (error) { showError(error, epoch); }
  finally { if(isCurrent(epoch, card)) { savingAnswer = false; elements.dontKnow.disabled = false; elements.know.disabled = false; } }
}
function nextCard() {
  if (!answerSaved || view !== "study") return;
  if (++cardIndex < session.cards.length) { renderCard(); return; }
  setView("summary");
  elements.summaryTitle.textContent = session.mode === "review" ? "Your words are getting stronger." : "A little more Italian is yours.";
  elements.summaryCopy.textContent = "Here is what you practiced in this session, including any answers saved earlier.";
  elements.summaryTotal.textContent = session.answered;
  elements.summaryRecalled.textContent = session.recalled;
  elements.summaryMissed.textContent = session.answered - session.recalled;
  elements.summaryLearning.focus({ preventScroll: true });
}

async function showWordLibrary(status) {
  const epoch = setView("library");
  elements.libraryTitle.textContent = status === "learned" ? "Learned words" : "Learning words";
  elements.libraryCopy.textContent = status === "learned" ? "Words recalled successfully at least three times, with a review interval of a week or more. Keep reviewing to retain them." : "Words you’ve practiced and are still building into memory. Tap one to revisit its lesson.";
  elements.libraryList.replaceChildren(); elements.librarySearch.value = ""; libraryWords = [];
  elements.cardsLoading.hidden = false; elements.loadingCopy.textContent = "Loading your vocabulary...";
  try {
    const data = await request(`/v1/flashcards/words?status=${status}`);
    if (epoch !== generation) return;
    libraryWords = data.words; renderLibrary();
  } catch (error) { showError(error, epoch); }
  finally { if(epoch===generation) elements.cardsLoading.hidden = true; }
}
function renderLibrary() {
  elements.libraryList.replaceChildren();
  const query = elements.librarySearch.value.trim().toLocaleLowerCase("it");
  const filtered = libraryWords.filter(word => word.word.toLocaleLowerCase("it").includes(query));
  if (!filtered.length) addText(elements.libraryList, "p", "empty-state", query ? "No words match your search." : "Your vocabulary will grow here as you practice.");
  filtered.forEach(word => {
    const item = document.createElement("button"); item.type = "button"; item.className = "library-item";
    const copy = document.createElement("span");
    addText(copy, "strong", "", word.word).lang = "it";
    addText(copy, "span", "library-stats", `${word.part_of_speech} · remembered ${word.known_count} · still learning ${word.unknown_count}`);
    item.append(copy); addText(item, "span", "due-tag", reviewTime(word.due));
    item.addEventListener("click", () => openDictionary(word.word)); elements.libraryList.append(item);
  });
}

async function openDictionary(word) {
  const cleaned = word.trim().normalize("NFC").toLocaleLowerCase("it"); if(!cleaned) return;
  dictionaryWord = cleaned; const epoch = ++dictionaryGeneration;
  if(!elements.dictionaryDialog.open) elements.dictionaryDialog.showModal();
  elements.lookupDetails.hidden = true; elements.lookupError.hidden = true; elements.retryLesson.hidden = true; elements.lookupLoading.hidden = false;
  const valid = () => epoch === dictionaryGeneration && elements.dictionaryDialog.open && accessToken;
  let complete = false;
  const outcome = lessons.get(cleaned).then(details => ({ details }), error => ({ error }));
  // Either result can arrive first. A slow preview must not hold back a cached lesson.
  void meanings.get(cleaned).then(preview => {
    if(valid() && !complete) { renderDetails(preview, null, elements.lookupDetails); elements.lookupLoading.hidden = true; }
  }).catch(() => {});
  const result = await outcome;
  if(!valid()) return;
  elements.lookupLoading.hidden = true;
  if(result.details) { complete = true; renderDetails(result.details, null, elements.lookupDetails); }
  else { elements.lookupError.textContent = result.error.message; elements.lookupError.hidden = false; elements.retryLesson.hidden = false; }
}

async function openSettings() {
  const epoch = accountGeneration;
  elements.settingsError.hidden = true;
  try {
    const data = await request("/v1/flashcards/overview"); if(!accessToken || epoch !== accountGeneration) return;
    elements.difficultyTarget.value = data.difficulty_target; elements.difficultyValue.value = data.difficulty_target;
    elements.settingsDialog.showModal();
  } catch(error) { showError(error); }
}
async function saveSettings(event) {
  event.preventDefault(); elements.saveSettings.disabled = true; elements.settingsError.hidden = true;
  const epoch = accountGeneration;
  try {
    await request("/v1/flashcards/settings", { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ difficulty_target: Number(elements.difficultyTarget.value) }) });
    if(epoch !== accountGeneration || !accessToken) return;
    elements.settingsDialog.close(); if(view === "home") await showHome();
  } catch(error) { if(epoch === accountGeneration) { elements.settingsError.textContent = error.message; elements.settingsError.hidden = false; } }
  finally { elements.saveSettings.disabled = false; }
}

for(const id of ["navHome", "pauseSession", "summaryHome", "closeLibrary"]) elements[id].addEventListener("click", () => { void showHome(); });
elements.authForm.addEventListener("submit", handleAuth);
elements.loginTab.addEventListener("click", () => setAuthMode("login"));
elements.registerTab.addEventListener("click", () => setAuthMode("register"));
elements.logout.addEventListener("click", async () => { try { await request("/v1/auth/logout", {method:"POST"}); } catch (_) { /* Local logout remains available offline. */ } finally { showAuth(); } });
elements.startLesson.addEventListener("click", () => loadCards("learn"));
elements.startReview.addEventListener("click", () => loadCards("review"));
elements.revealCard.addEventListener("click", revealCard);
elements.dontKnow.addEventListener("click", () => answerCard(false));
elements.know.addEventListener("click", () => answerCard(true));
elements.nextCard.addEventListener("click", nextCard);
elements.viewLearning.addEventListener("click", () => showWordLibrary("learning"));
elements.viewLearned.addEventListener("click", () => showWordLibrary("learned"));
elements.summaryLearning.addEventListener("click", () => showWordLibrary("learning"));
elements.librarySearch.addEventListener("input", renderLibrary);
elements.openLesson.addEventListener("click", () => openDictionary(currentCard().word));
elements.retryLesson.addEventListener("click", () => openDictionary(dictionaryWord));
elements.wordSearchForm.addEventListener("submit", event => { event.preventDefault(); void openDictionary(elements.wordSearchInput.value); });
elements.settings.addEventListener("click", openSettings);
elements.settingsForm.addEventListener("submit", saveSettings);
elements.difficultyTarget.addEventListener("input", () => { elements.difficultyValue.value = elements.difficultyTarget.value; });
elements.closeDictionary.addEventListener("click", () => elements.dictionaryDialog.close());
elements.dictionaryDialog.addEventListener("close", () => { dictionaryGeneration++; });
elements.closeSettings.addEventListener("click", () => elements.settingsDialog.close());
for(const dialog of [elements.dictionaryDialog, elements.settingsDialog]) dialog.addEventListener("click", event => { if(event.target === dialog) { const rect=dialog.getBoundingClientRect(); if(event.clientX<rect.left||event.clientX>rect.right||event.clientY<rect.top||event.clientY>rect.bottom) dialog.close(); } });
elements.listenWord.addEventListener("click", () => { if(!currentCard() || !("speechSynthesis" in window)) return; speechSynthesis.cancel(); const speech = new SpeechSynthesisUtterance(currentCard().word); speech.lang = "it-IT"; speech.rate = .85; speechSynthesis.speak(speech); });
document.addEventListener("keydown", event => {
  if(view!=="study" || event.repeat || event.ctrlKey || event.altKey || event.metaKey || elements.dictionaryDialog.open || elements.settingsDialog.open || /INPUT|TEXTAREA|SELECT/.test(event.target.tagName)) return;
  if(event.code === "Space" && event.target.tagName !== "BUTTON") { event.preventDefault(); if(answerSaved) nextCard(); else if(!elements.revealCard.hidden) void revealCard(); }
  if(event.key === "1") void answerCard(false);
  if(event.key === "2") void answerCard(true);
});
setAuthMode("login");
if(accessToken) request("/v1/auth/me").then(result => showApp(result.user)).catch(() => showAuth());
else showAuth();
