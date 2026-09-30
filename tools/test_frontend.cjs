// Requires Playwright: npm install --prefix .cache/browser playwright
// Runs against Docker using a temporary account; dictionary responses are fixtures.
const { chromium } = require('../.cache/browser/node_modules/playwright');
const assert = require('node:assert/strict');
const { execFileSync } = require('node:child_process');
const origin = process.env.TEST_FRONTEND_URL || 'http://localhost:5175';
const email = `nodo-ui-${Date.now()}@example.invalid`;
const password = 'Ui-test-password-4829!';
let user;
let browser;
(async () => {
  browser = await chromium.launch({ executablePath: process.env.TEST_BROWSER_PATH || 'C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe', headless: true });
  const page = await browser.newPage({ viewport: { width: 1280, height: 1000 } });
  const errors = [];
  let dictionaryFails = true;
  page.on('pageerror', error => errors.push(error.message));
  await page.route('**/v1/dictionary/**', async route => {
    const url = new URL(route.request().url());
    const word = decodeURIComponent(url.pathname.split('/').pop());
    if (word === 'provaerrore' && dictionaryFails) {
      await route.fulfill({status:502,json:{error:{message:'Temporary dictionary outage. Please retry.'}}});
      return;
    }
    const preview = url.searchParams.get('view') === 'preview';
    const details = { word, source_url: `https://en.wiktionary.org/wiki/${word}`, entries: [{part_of_speech: 'noun', grammar: ['masculine'], definitions: [{meaning: `Meaning of ${word}`, examples: ['Questo mi piace. - I like this.']}]}] };
    if (!preview) {
      // Deliberately slow teaching data must never block the quick meaning.
      await new Promise(resolve => setTimeout(resolve, 800));
      details.learning = {meaning: `Meaning of ${word}`, summary: 'A useful everyday word.', grammar_note: 'Learn the article with the noun.', examples: Array.from({length:4}, () => ({italian:'Questo mi piace.', english:'I like this.'}))};
    }
    try { await route.fulfill({json: {details}}); } catch (_) { /* Page may have closed during a prefetch. */ }
  });
  const registered = await page.request.post(`${origin}/api/v1/auth/register`, {data:{email,password}});
  assert.equal(registered.status(),201,await registered.text());
  const auth = await registered.json(); user = auth.user;
  await page.goto(origin);
  await page.evaluate(token => sessionStorage.setItem('nodo_access_token',token),auth.token);
  await page.reload();
  await page.locator('#start-lesson:enabled').waitFor();
  assert.equal(await page.locator('#stat-learning').innerText(),'0');
  assert.equal(await page.locator('#start-review').isDisabled(),true);
  await page.screenshot({path:'.cache/home-desktop.png',fullPage:true});
  await page.locator('#settings').click();
  await page.locator('#difficulty-target').fill('20');
  await page.locator('#save-settings').click();
  await page.locator('#settings-dialog').waitFor({state:'hidden'});
  const headers = {Authorization:`Bearer ${auth.token}`};
  assert.equal((await (await page.request.get(`${origin}/api/v1/flashcards/overview`,{headers})).json()).difficulty_target,20);
  await page.locator('#start-lesson').click();
  await page.locator('#reveal-card').waitFor();
  assert.equal(await page.locator('#card-status').textContent(),'New');
  const first = await page.locator('#card-word').innerText();
  await page.reload();
  await page.locator('#start-lesson:enabled').click();
  await page.locator('#reveal-card').waitFor();
  assert.equal(await page.locator('#card-word').innerText(),first);
  for(let i=0;i<20;i++) {
    await page.locator('#reveal-card').click();
    await page.locator('#answer-actions').waitFor({timeout:2000});
    assert.match(await page.locator('#card-meaning').innerText(),/^Meaning of/);
    if(i===0) {
      await page.setViewportSize({width:390,height:844});
      await page.screenshot({path:'.cache/card-mobile.png',fullPage:true});
      assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), 'Mobile horizontal overflow');
    }
    await page.locator(i%4 === 0 ? '#dont-know' : '#know').click();
    await page.locator('#next-card').waitFor();
    await page.locator('#next-card').click();
  }
  await page.locator('#session-summary').waitFor();
  assert.equal(await page.locator('#summary-total').innerText(),'20');
  assert.equal(await page.locator('#summary-recalled').innerText(),'15');
  assert.equal(await page.locator('#summary-missed').innerText(),'5');
  await page.locator('#summary-learning').click();
  await page.locator('.library-item').first().waitFor();
  assert.equal(await page.locator('.library-item').count(),20);
  await page.locator('#library-search').fill(first);
  assert((await page.locator('.library-item').count()) >= 1);
  await page.locator('.library-item').first().click();
  await page.locator('#lookup-details .learning-meaning').waitFor();
  await page.locator('#close-dictionary').click();
  await page.locator('#view-learned').click();
  await page.locator('.empty-state').waitFor();
  await page.locator('#nav-home').click();
  await page.locator('#start-review:enabled').click();
  await page.locator('#reveal-card').waitFor();
  assert.equal(await page.locator('#card-status').textContent(),'Review');
  await page.locator('#pause-session').click();
  await page.locator('#start-lesson:enabled').waitFor();
  await page.screenshot({path:'.cache/home-mobile.png',fullPage:true});
  await page.locator('#word-search-input').fill('provaerrore');
  await page.locator('#word-search-form button').click();
  await page.locator('#retry-lesson').waitFor();
  dictionaryFails = false;
  await page.locator('#retry-lesson').click();
  await page.locator('#lookup-details .learning-summary').waitFor();
  await page.locator('#close-dictionary').click();
  await page.locator('#logout').click();
  await page.locator('#auth-view').waitFor();
  assert.equal(await page.evaluate(() => sessionStorage.getItem('nodo_access_token')),null);
  assert.deepEqual(errors,[]);
  console.log('PASS: settings, saved-session resume, 20 answers, summary, searchable libraries, dictionary failure/retry, review-only session, mobile overflow, logout, no browser errors.');
})().catch(error => { console.error(error);process.exitCode=1; }).finally(async () => {
  if(browser) await browser.close();
  if(user && /^[0-9a-f-]{36}$/.test(user.id)) {
    execFileSync('docker',['compose','exec','-T','postgres','psql','-U','nodo','-d','nodo','-v','ON_ERROR_STOP=1','-c',`DELETE FROM users WHERE id='${user.id}' AND email='${email}';`],{stdio:'pipe'});
    console.log('Temporary test account removed.');
  }
});
