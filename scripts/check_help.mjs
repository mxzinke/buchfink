#!/usr/bin/env node
/** Prüft Kurztexte und ihre Zuordnung zu den Hilfestellen im JSX. */
import fs from 'node:fs';
import path from 'node:path';
import ts from '../frontend/node_modules/typescript/lib/typescript.js';

let errors = 0;
function checkFile(file) {
  const source = ts.createSourceFile(file, fs.readFileSync(file, 'utf8'), ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);
  function fail(node, text) {
    errors++;
    console.error(`${file}:${source.getLineAndCharacterOfPosition(node.getStart()).line + 1}: ${text}`);
  }
  function visit(node) {
    if (ts.isJsxOpeningElement(node) || ts.isJsxSelfClosingElement(node)) {
      const name = node.tagName.getText(source);
      const props = node.attributes.properties;
      const attr = (key) => props.find((p) => p.name?.getText(source) === key);
      const isHelp = name === 'Help';
      if (isHelp || attr('explain')) {
        const summary = attr(isHelp ? 'summary' : 'helpSummary');
        if (!summary) fail(node, 'Hilfetext ohne kurze Tooltip-Erklärung.');
        else if (summary.initializer && ts.isStringLiteral(summary.initializer)) {
          const text = summary.initializer.text;
          if (!text.trim() || text.length > 200 || /§|\b(?:GoBD|Abs\.|Art\.\s*\d|XBRL|TaxKey)\b/.test(text)) {
            fail(summary, 'Tooltip braucht einen kurzen Satz ohne Gesetzesverweis oder Implementierungsdetails.');
          }
        }
      }
      for (const key of ['label', 'title', 'hint', 'context', 'line']) {
        const prop = attr(key);
        if (prop?.initializer && ts.isStringLiteral(prop.initializer) && /§|GoBD\s+Rz/.test(prop.initializer.text)) {
          fail(prop, 'Gesetzesverweis gehört in die Erklärung im Detaildialog.');
        }
      }
    }
    ts.forEachChild(node, visit);
  }
  visit(source);
}
function walk(dir) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const file = path.join(dir, entry.name);
    if (entry.isDirectory()) walk(file);
    else if (file.endsWith('.tsx')) checkFile(file);
  }
}
walk('frontend/src');
if (errors) process.exitCode = 1;
else console.log('Hilfestellen: kurze Tooltips vorhanden, keine Norm in statischen Feldbeschriftungen.');
