const API_URL = window.NODO_CONFIG?.apiUrl || "/api";
const TOKEN_KEY = "nodo_access_token";

const elements = Object.fromEntries([
  "auth-view", "auth-form", "auth-error", "auth-submit", "login-tab", "register-tab",
  "identity", "identity-label", "password", "password-hint", "quiz-view", "account",
  "account-name", "logout", "loading", "error", "error-message", "retry", "question",
  "word", "options", "feedback", "next",
].map((id) => [id.replace(/-([a-z])/g, (_, letter) => letter.toUpperCase()), document.querySelector(`#${id}`)]));

let authMode = "login";
let accessToken = sessionStorage.getItem(TOKEN_KEY);
let currentQuestion = null;

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
  elements.quizView.hidden = true;
  elements.account.hidden = true;
  elements.authView.hidden = false;
  elements.authError.textContent = message;
  elements.authError.hidden = !message;
}

function showApp(user) {
  elements.authView.hidden = true;
  elements.quizView.hidden = false;
  elements.account.hidden = false;
  elements.accountName.textContent = user.username || user.email;
  loadQuestion();
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

function showQuizError(message) {
  elements.loading.hidden = true;
  elements.question.hidden = true;
  elements.errorMessage.textContent = message;
  elements.error.hidden = false;
}

function renderQuestion(question) {
  currentQuestion = question;
  elements.loading.hidden = true;
  elements.error.hidden = true;
  elements.question.hidden = false;
  elements.feedback.hidden = true;
  elements.feedback.className = "feedback";
  elements.next.hidden = true;
  elements.word.textContent = question.word;
  elements.options.replaceChildren();
  question.options.forEach((meaning, index) => {
    const button = document.createElement("button");
    button.type = "button";
    button.className = "option";
    button.innerHTML = `<span class="option-index">${String.fromCharCode(65 + index)}</span><span></span>`;
    button.lastElementChild.textContent = meaning;
    button.addEventListener("click", () => submitAnswer(index, button));
    elements.options.append(button);
  });
}

async function loadQuestion() {
  elements.error.hidden = true;
  elements.question.hidden = true;
  elements.loading.hidden = false;
  try { renderQuestion((await request("/v1/quiz/word")).question); }
  catch (error) { if (accessToken) showQuizError(error.message); }
}

async function submitAnswer(option, selectedButton) {
  const buttons = [...elements.options.querySelectorAll("button")];
  buttons.forEach((button) => { button.disabled = true; });
  elements.feedback.textContent = "Checking...";
  elements.feedback.hidden = false;
  try {
    const data = await request(`/v1/quiz/word/${currentQuestion.id}/answer`, {
      method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ option }),
    });
    selectedButton.classList.add(data.correct ? "correct" : "incorrect");
    elements.feedback.textContent = data.correct ? "Correct - nicely done." : `Not quite. The answer is "${data.correct_answer}".`;
    elements.feedback.classList.add(data.correct ? "success" : "failure");
    elements.next.hidden = false;
  } catch (error) {
    buttons.forEach((button) => { button.disabled = false; });
    if (accessToken) { elements.feedback.textContent = error.message; elements.feedback.classList.add("failure"); }
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
elements.next.addEventListener("click", loadQuestion);
elements.retry.addEventListener("click", loadQuestion);
setAuthMode("login");
restoreSession();
