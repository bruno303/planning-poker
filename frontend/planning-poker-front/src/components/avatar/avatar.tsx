import styles from './avatar.module.css';

type Participant = {
  id: string;
};

type AvatarDefinition = {
  filename: string;
  label: string;
};

const avatarCatalog = [
  { filename: 'fox.svg', label: 'Fox avatar' },
  { filename: 'dog.svg', label: 'Dog avatar' },
  { filename: 'cat.svg', label: 'Cat avatar' },
  { filename: 'hammer.svg', label: 'Hammer avatar' },
  { filename: 'strawberry.svg', label: 'Strawberry avatar' },
  { filename: 'wrench.svg', label: 'Wrench avatar' },
  { filename: 'apple.svg', label: 'Apple avatar' },
  { filename: 'screwdriver.svg', label: 'Screwdriver avatar' },
  { filename: 'lemon.svg', label: 'Lemon avatar' },
] satisfies readonly AvatarDefinition[];

const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;

function avatarIndex(participantId: string): number {
  const id = participantId.trim();

  if (!uuidPattern.test(id)) {
    return 0;
  }

  let hash = 0;
  for (let index = 0; index < id.length; index += 1) {
    hash = (hash * 31 + id.charCodeAt(index)) | 0;
  }

  return (hash >>> 0) % avatarCatalog.length;
}

export default function Avatar({ participant }: { participant: Participant }) {
  const avatar = avatarCatalog[avatarIndex(participant.id)];

  return (
    <span className={styles.container}>
      <img
        alt={avatar.label}
        className={styles.image}
        height={32}
        src={`/avatars/${avatar.filename}`}
        width={32}
      />
    </span>
  );
}
