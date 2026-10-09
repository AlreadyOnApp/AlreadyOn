'use strict';

// Keep experimental sampling checks aligned with the production loudness contract.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const { measurement, analysisArgs } = require('../benchmark-loudness.cjs');

test('sample duration bounds input decoding and selects one audio stream', () => {
  for (const seconds of [60, 300, 12.35]) {
    const args = analysisArgs('/media/file with spaces.mkv', 3, seconds);
    assert.ok(args.indexOf('-t') < args.indexOf('-i'));
    assert.equal(args[args.indexOf('-t') + 1], String(seconds));
    assert.equal(args[args.indexOf('-map') + 1], '0:3');
    for (const flag of ['-vn', '-sn', '-dn']) assert.ok(args.includes(flag));
  }
  assert.ok(!analysisArgs('/media/file.mkv', 0, 0).includes('-t'));
});

test('benchmark filter remains identical to production analysis', () => {
  const worker = fs.readFileSync(path.join(__dirname, '../../server/internal/loudness/worker.go'), 'utf8');
  const constants = fs.readFileSync(path.join(__dirname, '../../server/internal/loudness/loudness.go'), 'utf8');
  const args = analysisArgs('file.mkv', 0, 300);
  const filter = args[args.indexOf('-af') + 1];
  const downmix = /const Downmix = "([^"]+)"/.exec(constants)[1];
  assert.equal(filter, `${downmix},loudnorm=I=-16:TP=-1.5:LRA=50:print_format=json`);
  assert.match(worker, /Downmix\s*\+\s*",loudnorm=I=-16:TP=-1\.5:LRA=50:print_format=json"/);
  assert.ok(constants.includes('const Target = -16.0'));
});

test('measurement reports gain separately from the loudnorm target offset', () => {
  const result = measurement('diagnostic\n{"input_i":"-24","input_tp":"-4","input_lra":"3","input_thresh":"-34","target_offset":"0.5"}\n');
  assert.equal(result.status, 'ready');
  assert.equal(result.gainDB, 8);
  assert.equal(result.offset, 0.5);
});

test('silence cannot be treated as a usable sample', () => {
  assert.deepEqual(measurement('{"input_i":"-inf","input_tp":"-inf"}'), { status: 'silent' });
});

test('malformed and non-finite measurements are rejected', () => {
  for (const raw of ['no JSON', '{}', '{"input_i":"NaN"}', '{"input_i":"-inf","input_tp":"-20"}', '{"input_i":"1","input_tp":"-4","input_lra":"3","input_thresh":"-34","target_offset":"0"}']) {
    assert.throws(() => measurement(raw));
  }
});
