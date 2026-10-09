import { useState } from 'react';

export default function PhraseInput({ values, onChange }: { values: string[]; onChange: (v: string[]) => void }) {
  const [draft, setDraft] = useState('');
  const commit = () => {
    const p = draft.trim();
    if (p && !values.some((v) => v.toLowerCase() === p.toLowerCase())) onChange([...values, p]);
    setDraft('');
  };
  return (
    <div
      className="phrase-input"
      style={{
        display: 'flex',
        flexWrap: 'wrap',
        gap: 6,
      }}
    >
      {values.map((p, i) => (
        <span
          key={i}
          style={{
            display: 'inline-flex',
            alignItems: 'center',
            gap: 6,
            borderRadius: 20,
            background: 'var(--bg-elevated, #343d55)',
            padding: '3px 10px',
          }}
        >
          {p}
          <button type="button" aria-label={'Remove phrase ' + p} onClick={() => onChange(values.filter((_, j) => j !== i))}>
            ×
          </button>
        </span>
      ))}
      <input
        aria-label="Title Contains"
        value={draft}
        maxLength={100}
        placeholder="Type a word or phrase…"
        onChange={(e) => setDraft(e.target.value)}
        onBlur={commit}
        onKeyDown={(e) => {
          if (e.key === 'Enter') {
            e.preventDefault();
            commit();
          } else if (e.key === 'Tab') {
            commit();
          }
        }}
      />
    </div>
  );
}
