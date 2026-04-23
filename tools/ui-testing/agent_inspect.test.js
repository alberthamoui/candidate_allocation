const test = require('node:test');
const assert = require('node:assert/strict');

const {
  parseArgs,
  shouldCaptureScreenshot,
  buildSuccessMessage,
} = require('./agent_inspect');

test('parseArgs keeps screenshot disabled by default', () => {
  const args = parseArgs(['node', 'agent_inspect.js']);

  assert.equal(shouldCaptureScreenshot(args), false);
});

test('parseArgs enables screenshot when explicitly requested', () => {
  const args = parseArgs(['node', 'agent_inspect.js', '--screenshot=true']);

  assert.equal(shouldCaptureScreenshot(args), true);
});

test('buildSuccessMessage omits screenshot when not requested', () => {
  const message = buildSuccessMessage({
    stdout: '',
    pngPath: '/tmp/a.png',
    htmlPath: '/tmp/a.html',
    a11yPath: '/tmp/a_a11y.json',
    captureScreenshot: false,
  });

  assert.ok(message.includes('HTML DOM:       /tmp/a.html'));
  assert.ok(message.includes('Accessibility:  /tmp/a_a11y.json'));
  assert.equal(message.includes('Screenshot:'), false);
});

test('buildSuccessMessage includes screenshot when requested', () => {
  const message = buildSuccessMessage({
    stdout: '',
    pngPath: '/tmp/a.png',
    htmlPath: '/tmp/a.html',
    a11yPath: '/tmp/a_a11y.json',
    captureScreenshot: true,
  });

  assert.ok(message.includes('Screenshot:     /tmp/a.png'));
});
