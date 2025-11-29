import { useEffect, useRef } from 'react';
import { styles } from './styles';

type Props = {
  currentStory: string;
  onChange: (e: React.ChangeEvent<HTMLInputElement>) => void;
  onKeyDown: (e: React.KeyboardEvent<HTMLInputElement>) => void;
};

function FocusableComponent(props: Props) {
  const inputRef = useRef<HTMLInputElement | null>(null);

  useEffect(() => {
    if (inputRef.current) {
      inputRef.current?.focus();
    }    
  }, []);

  return (
    <input
      type="text"
      value={props.currentStory}
      onChange={props.onChange}
      onKeyDown={props.onKeyDown}
      style={styles.input}
      ref={inputRef}
    />
  );
}

export default FocusableComponent;
