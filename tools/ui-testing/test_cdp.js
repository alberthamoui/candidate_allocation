const { chromium } = require('@playwright/test');
chromium.launch().then(async browser => {
  const page = await browser.newPage();
  await page.setContent('<button>Click me</button>');
  const client = await page.context().newCDPSession(page);
  const { nodes } = await client.send('Accessibility.getFullAXTree');
  console.log(nodes.length);
  await browser.close();
});
