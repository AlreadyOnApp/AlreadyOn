#!/usr/bin/env node
'use strict';

const { spawn, spawnSync } = require('node:child_process');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');

const toleranceDB = 1;
const downmix = 'aformat=sample_rates=48000:channel_layouts=stereo';
const filter = `${downmix},loudnorm=I=-16:TP=-1.5:LRA=50:print_format=json`;

function probe(file) {
  const result = spawnSync('ffprobe', ['-v', 'error', '-show_streams', '-show_format', '-of', 'json', file], { encoding: 'utf8', timeout: 15000 });
  if (result.error || result.status !== 0) throw result.error || new Error(result.stderr);
  const info = JSON.parse(result.stdout);
  const audio = info.streams.find(stream => stream.codec_type === 'audio');
  if (!audio) throw new Error('No audio stream');
  return { track: audio.index, codec: audio.codec_name, channels: audio.channels, layout: audio.channel_layout, durationSeconds: Number(info.format.duration) };
}

function measurement(stderr) {
  const start = stderr.lastIndexOf('{');
  const end = stderr.lastIndexOf('}');
  if (start < 0 || end < start) throw new Error('Missing loudnorm result');
  const raw = JSON.parse(stderr.slice(start, end + 1));
  if (raw.input_i === '-inf' && raw.input_tp === '-inf') return { status: 'silent' };
  const result = { status: 'ready', integrated: Number(raw.input_i), truePeak: Number(raw.input_tp), range: Number(raw.input_lra), threshold: Number(raw.input_thresh), offset: Number(raw.target_offset) };
  result.gainDB = -16 - result.integrated;
  if (Object.values(result).some(value => typeof value === 'number' && !Number.isFinite(value))) throw new Error('Non-finite loudness result');
  if (result.integrated < -99 || result.integrated > 0 || result.truePeak < -99 || result.truePeak > 99 || result.range < 0 || result.range > 99 || result.threshold < -99 || result.threshold > 0 || Math.abs(result.offset) > 99) throw new Error('Invalid loudness result');
  return result;
}

function analysisArgs(file, track, seconds) {
  const args = ['-hide_banner', '-nostdin', '-nostats', '-threads', '1', '-filter_threads', '1', '-protocol_whitelist', 'file,pipe'];
  // An input duration limit stops decoding, rather than just truncating filtered output.
  if (seconds) args.push('-t', String(seconds));
  args.push('-i', file, '-map', `0:${track}`, '-vn', '-sn', '-dn', '-af', filter, '-threads', '1', '-f', 'null', '-');
  return args;
}

function descendants(pid) {
  try {
    const children = fs.readFileSync(`/proc/${pid}/task/${pid}/children`, 'utf8').trim().split(/\s+/).filter(Boolean).map(Number);
    return [pid, ...children.flatMap(descendants)];
  } catch { return []; }
}

async function analyze(file, track, seconds) {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'alreadyon-loudness-'));
  const resourceFile = path.join(directory, 'resources.json');
  const args = analysisArgs(file, track, seconds);
  const child = spawn('/usr/bin/time', ['-f', '{"wallSeconds":%e,"userSeconds":%U,"systemSeconds":%S,"peakRSSKiB":%M,"inputBlocks":%I,"outputBlocks":%O}', '-o', resourceFile, 'nice', '-n', '19', 'ionice', '-c', '3', 'ffmpeg', ...args], { stdio: ['ignore', 'ignore', 'pipe'] });
  let stderr = '';
  let logicalReadBytes = 0;
  let physicalReadBytes = 0;
  let peakCPUPercent = 0;
  const previous = new Map();
  const clock = spawnSync('getconf', ['CLK_TCK'], { encoding: 'utf8' });
  const ticksPerSecond = Number(clock.stdout);
  const sample = () => {
    const now = performance.now();
    let logical = 0;
    let physical = 0;
    let cpu = 0;
    for (const pid of descendants(child.pid)) {
      try {
        const stat = fs.readFileSync(`/proc/${pid}/stat`, 'utf8');
        const fields = stat.slice(stat.lastIndexOf(')') + 2).split(' ');
        const ticks = Number(fields[11]) + Number(fields[12]);
        const last = previous.get(pid);
        if (last && ticksPerSecond > 0) cpu += (ticks - last.ticks) / ticksPerSecond / ((now - last.now) / 1000) * 100;
        previous.set(pid, { ticks, now });
        const io = Object.fromEntries(fs.readFileSync(`/proc/${pid}/io`, 'utf8').trim().split('\n').map(line => line.split(':').map(value => value.trim())));
        logical += Number(io.rchar);
        physical += Number(io.read_bytes);
      } catch { /* Processes can exit between samples. */ }
    }
    logicalReadBytes = Math.max(logicalReadBytes, logical);
    physicalReadBytes = Math.max(physicalReadBytes, physical);
    peakCPUPercent = Math.max(peakCPUPercent, cpu);
  };
  const timer = setInterval(sample, 100);
  child.stderr.on('data', data => { stderr = (stderr + data).slice(-65536); });
  try {
    await new Promise((resolve, reject) => {
      child.once('error', reject);
      child.once('close', code => code === 0 ? resolve() : reject(new Error(`FFmpeg exited ${code}: ${stderr.slice(-2000)}`)));
    });
    const resources = JSON.parse(fs.readFileSync(resourceFile, 'utf8'));
    resources.cpuSeconds = resources.userSeconds + resources.systemSeconds;
    resources.averageCPUPercent = resources.wallSeconds > 0 ? resources.cpuSeconds / resources.wallSeconds * 100 : null;
    return { ...measurement(stderr), resources: { ...resources, peakCPUPercent, sampledLogicalReadBytes: logicalReadBytes, sampledPhysicalReadBytes: physicalReadBytes } };
  } finally {
    clearInterval(timer);
    fs.rmSync(directory, { recursive: true, force: true });
  }
}

async function main(files) {
  if (!files.length) throw new Error('Usage: node scripts/benchmark-loudness.cjs FILE [FILE ...] | --synthetic');
  for (const input of files) {
    const file = path.resolve(input);
    const info = probe(file);
    if (!Number.isFinite(info.durationSeconds) || info.durationSeconds <= 0) throw new Error('A known positive duration is required for percentage comparison');
    process.stderr.write(`Comparing ${path.basename(file)} (${info.durationSeconds}s, ${info.codec}, ${info.channels} channels)\n`);
    const full = await analyze(file, info.track, 0);
    for (const [sampleMethod, sampleSeconds] of [['60-seconds', 60], ['five-minutes', 300], ['first-tenth', info.durationSeconds / 10]]) {
      const sample = await analyze(file, info.track, sampleSeconds);
      const comparable = full.status === 'ready' && sample.status === 'ready';
      const difference = comparable ? { integratedDB: sample.integrated - full.integrated, offsetDB: sample.offset - full.offset, gainDB: sample.gainDB - full.gainDB } : null;
      const reduction = {};
      for (const field of ['wallSeconds', 'cpuSeconds', 'inputBlocks', 'sampledLogicalReadBytes', 'sampledPhysicalReadBytes']) {
        reduction[field] = full.resources[field] > 0 ? (1 - sample.resources[field] / full.resources[field]) * 100 : null;
      }
      const report = { file: path.basename(file), ...info, sampleMethod, sampleSeconds, toleranceDB, full, sample, difference, withinTolerance: comparable && Math.abs(difference.gainDB) <= toleranceDB && Math.abs(difference.offsetDB) <= toleranceDB, reductionPercent: reduction };
      report.full.resources.mediaSecondsPerWallSecond = info.durationSeconds / full.resources.wallSeconds;
      report.sample.resources.mediaSecondsPerWallSecond = Math.min(sampleSeconds, info.durationSeconds) / sample.resources.wallSeconds;
      process.stdout.write(`${JSON.stringify(report)}\n`);
    }
  }
}

function fixtures(directory) {
  const cases = [
    ['constant-loud', '0.3', 420, 'aac', 2],
    ['constant-quiet', '0.003', 420, 'aac', 2],
    ['quiet-intro', '0.015*(1+9*gte(t,90))', 420, 'aac', 2],
    ['loud-intro', '0.15*(1-0.9*gte(t,90))', 420, 'aac', 2],
    ['silent-intro', '0.15*gte(t,90)', 420, 'aac', 2],
    ['late-rise', '0.015*(1+9*gte(t,330))', 420, 'aac', 2],
    ['silent', '0', 420, 'aac', 2],
    ['short', '0.05', 30, 'aac', 2],
    ['45-minute-surround', '0.05', 2700, 'eac3', 6],
  ];
  return cases.map(([name, amplitude, duration, codec, channels]) => {
    const file = path.join(directory, `${name}.mkv`);
    const source = `aevalsrc='(${amplitude})*sin(2*PI*440*t)':s=48000:d=${duration}`;
    const result = spawnSync('nice', ['-n', '19', 'ionice', '-c', '3', 'ffmpeg', '-v', 'error', '-nostdin', '-threads', '1', '-filter_threads', '1', '-f', 'lavfi', '-i', source, '-ac', String(channels), '-c:a', codec, '-b:a', '192k', '-threads', '1', file], { encoding: 'utf8' });
    if (result.error || result.status !== 0) throw result.error || new Error(result.stderr);
    return file;
  });
}

async function cli(args) {
  if (args.length === 1 && args[0] === '--synthetic') {
    const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'alreadyon-loudness-fixtures-'));
    try { await main(fixtures(directory)); }
    finally { fs.rmSync(directory, { recursive: true, force: true }); }
  } else {
    await main(args);
  }
}

module.exports = { probe, measurement, analysisArgs, analyze };
if (require.main === module) cli(process.argv.slice(2)).catch(error => { console.error(error.message); process.exitCode = 1; });
