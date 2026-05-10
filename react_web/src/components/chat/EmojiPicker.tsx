import React from 'react'

const EMOJIS = [
  '😀','😁','😂','🤣','😃','😄','😅','😆',
  '😉','😊','😋','😎','😍','🥰','😘','😗',
  '😙','😚','🙂','🤗','🤩','🤔','🤨','😐',
  '😑','😶','🙄','😏','😣','😥','😮','🤐',
  '😯','😪','😫','😴','😌','😛','😜','😝',
  '🤤','😒','😓','😔','😕','🙃','🤑','😲',
  '🙁','😖','😞','😟','😤','😢','😭','😦',
  '😧','😨','😩','🤯','😬','😰','😱','🥵',
  '🥶','😳','🤪','😵','😠','😡','🤬','😈',
  '👿','💀','💩','🤡','👹','👺','👻','👽',
  '👾','🤖','😺','😸','😹','😻','😼','😽',
  '🙀','😿','😾','👋','🤚','🖐','✋','🖖',
  '👌','🤏','✌️','🤞','🤟','🤘','🤙','👈',
  '👉','👆','🖕','👇','☝️','👍','👎','✊',
]

interface Props {
  onSelect: (emoji: string) => void
}

export default function EmojiPicker({ onSelect }: Props) {
  return (
    <div className="emoji-picker">
      {EMOJIS.map((emoji, i) => (
        <button
          key={i}
          className="emoji-item"
          onClick={() => onSelect(emoji)}
          type="button"
        >
          {emoji}
        </button>
      ))}
    </div>
  )
}
