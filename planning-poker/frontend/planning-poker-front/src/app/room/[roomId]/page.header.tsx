import { styles } from "./page.styles";

type HeaderProps = {
  handleBackToHome: () => void;
  generateShareableLink: () => string;
}

export default function Header({ handleBackToHome, generateShareableLink, children }: HeaderProps & { children: React.ReactNode }) {
  return (
    <div>
      {/* Room Header */}
      <div style={styles.roomHeader}>
        <button onClick={handleBackToHome} style={styles.backButton}>
          ← Back to Home
        </button>
        <div style={styles.roomInfo}>
          <span
            style={{
              ...styles.roomCode,
              cursor: 'pointer',
            }}
            onMouseEnter={e => {
              (e.target as HTMLElement).style.backgroundColor = '#e0e7ff';
              (e.target as HTMLElement).style.cursor = 'pointer';
            }}
            onMouseLeave={e => {
              (e.target as HTMLElement).style.backgroundColor = '#f3f4f6';
            }}
            title="Click to copy shareable link"
            onClick={async () => {
              await navigator.clipboard.writeText(generateShareableLink())

              const toast = document.createElement('div');
              toast.textContent = 'Shareable link copied!';
              Object.assign(toast.style, {
                position: 'fixed',
                top: '2rem',
                left: '50%',
                transform: 'translateX(-50%)',
                background: '#3b82f6',
                color: 'white',
                padding: '0.75rem 1.5rem',
                borderRadius: '0.5rem',
                boxShadow: '0 4px 12px rgba(0,0,0,0.12)',
                fontSize: '1rem',
                zIndex: 9999,
                opacity: '0',
                transition: 'opacity 0.3s'
              });
              document.body.appendChild(toast);
              setTimeout(() => {
                toast.style.opacity = '1';
              }, 10);
              setTimeout(() => {
                toast.style.opacity = '0';
                setTimeout(() => document.body.removeChild(toast), 300);
              }, 1800);
            }}
          >Click here to share the room</span>
        </div>
      </div>
      {children}
    </div>
  )
}
