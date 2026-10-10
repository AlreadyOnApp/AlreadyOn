import assert from 'node:assert/strict';
import { test } from 'node:test';
import { safeNext } from '../src/users/safeNext.ts';

// Exercise the browser URL parser, including characters stripped during navigation.
test('login redirects stay on the current origin after URL normalization', () => {
  const origin = 'https://alreadyon.example';
  for (const next of ['https://other.example', '//other.example', '/\\other.example', '/\t/other.example', '/\n/other.example', '/\r/other.example', 'javascript:alert(1)', 'relative', '//[']) {
    assert.equal(safeNext(`?next=${encodeURIComponent(next)}`, origin), null, JSON.stringify(next));
  }
  assert.equal(safeNext('', origin), null);
  for (const next of ['/api/docs/', '/guide?channel=101#now', '/%2Fother.example', '/a/../guide']) {
    assert.equal(safeNext(`?next=${encodeURIComponent(next)}`, origin), new URL(next, origin).href);
  }
});
