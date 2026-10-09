// Shared artwork uses outlined lettering, keeping branding independent of fonts.
import './Brand.css';
import { Link } from 'react-router-dom';

export default function Brand({ tagline = false }: { tagline?: boolean }) {
  return <Link to="/" className={'brand-link' + (tagline ? ' brand-link-full' : '')} aria-label="AlreadyOn home">
    <img className={'brand-logo' + (tagline ? ' brand-logo-full' : '')}
    src={tagline ? '/branding/logo.svg' : '/branding/header.svg'}
    alt={tagline ? 'AlreadyOn — Your library, tuned to you' : 'AlreadyOn'}
    draggable={false} />
  </Link>;
}
