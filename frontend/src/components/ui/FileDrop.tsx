import React, { useEffect, useId, useRef, useState } from 'react';
import { Events } from '@wailsio/runtime';
import { Upload } from 'lucide-react';
import { cn } from './cn';

export interface FileDropProps {
  onFiles: (files: File[]) => void;
  onPaths?: (paths: string[]) => void;
  /** Etwa `.pdf,.xml` oder `application/pdf`. */
  accept?: string;
  multiple?: boolean;
  disabled?: boolean;
  /** Ein Satz, was hier erwartet wird (§15.1). */
  hint?: string;
  className?: string;
}

/**
 * Belege und Kontoauszüge hereinziehen. Eine der wenigen erlaubten Flächen
 * (§6.2, Fall 4): Sie füllt die Stelle, an der sonst eine Liste stünde.
 *
 * Ziehen ist die bequeme, der Klick die verlässliche Variante. Beides führt zum
 * selben Ergebnis, und die Tastatur erreicht das Feld über den Button darin.
 */
export const FileDrop: React.FC<FileDropProps> = ({
  onFiles,
  onPaths,
  accept,
  multiple = true,
  disabled = false,
  hint = 'PDF oder XML hierher ziehen',
  className,
}) => {
  const targetId = useId();
  const input = useRef<HTMLInputElement>(null);
  const [over, setOver] = useState(false);

  useEffect(() => {
    if (!onPaths || disabled) return;
    return Events.On('buchfink:files-dropped', (event) => {
      if (event.data?.targetId !== targetId || !Array.isArray(event.data?.paths)) return;
      setOver(false);
      onPaths(event.data.paths);
    });
  }, [targetId, disabled, onPaths]);

  const take = (list: FileList | null) => {
    if (!list || list.length === 0) return;
    onFiles(Array.from(list));
  };

  return (
    <div
      id={targetId}
      data-file-drop-target={onPaths && !disabled ? '' : undefined}
      onDragOver={(e) => {
        e.preventDefault();
        if (disabled) return;
        setOver(true);
      }}
      onDragLeave={() => setOver(false)}
      onDrop={(e) => {
        e.preventDefault();
        if (disabled) return;
        setOver(false);
        // Wails löst native Drops separat als Dateipfade auf, auch unter Windows.
        if (onPaths && (window as any)._wails?.flags?.enableFileDrop === true) return;
        take(e.dataTransfer.files);
      }}
      className={cn(
        'rounded-card border border-dashed px-6 py-10 text-center',
        'transition-colors duration-120 ease-quiet',
        over ? 'border-accent bg-accent-soft' : 'border-line-strong bg-surface',
        '[&.file-drop-target-active]:border-accent [&.file-drop-target-active]:bg-accent-soft',
        disabled && 'opacity-60',
        className,
      )}
    >
      <div className="mx-auto grid h-12 w-12 place-items-center rounded-control bg-accent-soft text-accent-text">
        <Upload className="w-6 h-6" strokeWidth={1.5} />
      </div>
      <p className="text-body text-ink-muted mt-4">{hint}</p>
      <button
        type="button"
        disabled={disabled}
        onClick={() => input.current?.click()}
        className="mt-3 text-label font-semibold text-accent-text hover:text-accent transition-colors duration-120 ease-quiet disabled:text-ink-faint"
      >
        Datei auswählen
      </button>
      <input
        ref={input}
        type="file"
        accept={accept}
        multiple={multiple}
        disabled={disabled}
        className="hidden"
        onChange={(e) => {
          take(e.target.files);
          e.target.value = '';
        }}
      />
    </div>
  );
};
