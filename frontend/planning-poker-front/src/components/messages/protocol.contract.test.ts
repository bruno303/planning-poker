import { readdirSync, readFileSync } from 'node:fs';
import { dirname, join, relative, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

import { describe, expect, it } from 'vitest';

import { isClientCommand, isServerEvent } from './protocol.generated';

const fixturesRoot = resolve(dirname(fileURLToPath(import.meta.url)), '../../../../../protocol/fixtures');

function jsonFiles(directory: string): string[] {
  return readdirSync(directory, { withFileTypes: true })
    .filter((entry) => entry.isFile() && entry.name.endsWith('.json'))
    .map((entry) => join(directory, entry.name))
    .sort();
}

function readFixture(path: string): unknown {
  const parsed: unknown = JSON.parse(readFileSync(path, 'utf8'));
  return parsed;
}

function fixtureName(path: string): string {
  return relative(fixturesRoot, path);
}

const serverFixtureDir = join(fixturesRoot, 'server');
const clientFixtureDir = join(fixturesRoot, 'client');

describe('server frame fixtures', () => {
  it.each(jsonFiles(serverFixtureDir))('accepts %s', (path) => {
    expect(isServerEvent(readFixture(path)), fixtureName(path)).toBe(true);
  });

  it.each(jsonFiles(join(serverFixtureDir, 'invalid')))('rejects %s', (path) => {
    expect(isServerEvent(readFixture(path)), fixtureName(path)).toBe(false);
  });
});

describe('client frame fixtures', () => {
  it.each(jsonFiles(clientFixtureDir))('accepts %s', (path) => {
    expect(isClientCommand(readFixture(path)), fixtureName(path)).toBe(true);
  });

  it.each(jsonFiles(join(clientFixtureDir, 'invalid')))('rejects %s', (path) => {
    expect(isClientCommand(readFixture(path)), fixtureName(path)).toBe(false);
  });
});
