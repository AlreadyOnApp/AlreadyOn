// Binary settings can use either the compact sliding switch used in dense
// filter forms or the labeled On/Off treatment used by top-level settings.
export default function SettingToggle({ checked, onChange, label, text = false }: { checked: boolean; onChange: (value: boolean) => void; label: string; text?: boolean }) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      aria-label={label}
      className={(text ? 'admin-badge setting-text-toggle' : 'setting-toggle') + (checked ? (text ? ' admin-badge-on' : ' is-on') : '')}
      onClick={() => onChange(!checked)}
    >
      {text ? (checked ? 'On' : 'Off') : <span />}
    </button>
  );
}
