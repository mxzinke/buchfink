import React from 'react';
import { Browser, System } from '@wailsio/runtime';
import { REFERENCE_SOURCE, STATUTES } from '../../utils/norms';

export const GOBD_URL = 'https://amtliche-handbuecher.bundesfinanzministerium.de/ao/2025/Anhaenge/BMF-Schreiben-und-gleichlautende-Laendererlasse/Anhang-33/anhang-33.html';

export const SourceLink: React.FC<{ href: string; children: React.ReactNode }> = ({ href, children }) => (
  <a
    href={href}
    target="_blank"
    rel="noopener noreferrer"
    className="underline underline-offset-2 text-accent-text hover:text-accent"
    onClick={(event) => {
      // Die Desktop-Webview bleibt in der Anwendung. Im Browser genügt der Anker.
      if (System.IsMac() || System.IsWindows() || System.IsLinux()) {
        event.preventDefault();
        void Browser.OpenURL(href).catch(() => window.open(href, '_blank', 'noopener,noreferrer'));
      }
    }}
  >
    {children}
  </a>
);

const REFERENCE = new RegExp(REFERENCE_SOURCE, 'g');

function linkText(value: string): React.ReactNode {
  const parts: React.ReactNode[] = [];
  let end = 0;
  for (const match of value.matchAll(REFERENCE)) {
    const index = match.index!;
    parts.push(value.slice(end, index));
    const href = match[0].startsWith('GoBD') ? GOBD_URL
      : match[3] ? `https://eur-lex.europa.eu/legal-content/DE/TXT/?uri=CELEX:32016R0679#art_${match[3]}`
      : `https://www.gesetze-im-internet.de/${STATUTES[match[2]]}/__${match[1]}.html`;
    parts.push(<SourceLink key={index} href={href}>{match[0]}</SourceLink>);
    end = index + match[0].length;
  }
  return end ? <>{parts}{value.slice(end)}</> : value;
}

/** Verlinkt Quellen im Erklärungstext. Vorhandene Links und Code bleiben erhalten. */
export function legalText(children: React.ReactNode): React.ReactNode {
  return React.Children.map(children, (child) => {
    if (typeof child === 'string') return linkText(child);
    if (!React.isValidElement<{ children?: React.ReactNode }>(child)) return child;
    if (child.type === 'a' || child.type === 'code' || child.type === SourceLink) return child;
    if (!child.props.children) return child;
    return React.cloneElement(child, {}, legalText(child.props.children));
  });
}
