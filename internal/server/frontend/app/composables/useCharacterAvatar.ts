/**
 * Composable for mapping session IDs to cat-based fallback avatars
 * Uses TPB-inspired cat names and local avatar images
 */

export interface CharacterInfo {
  name: string
  avatar: string
  color: string
}

// Cat characters with Trailer Park Boys-inspired names
// Maps to local images in /avatars/cats/
const fallbackAvatars: CharacterInfo[] = [
  { name: 'Bubbles de Catt', avatar: '/avatars/cats/cat-coder.jpg', color: '#85C1E2' },
  { name: 'Julian Purresso', avatar: '/avatars/cats/cat-coffee.jpg', color: '#F8B88B' },
  { name: 'Inspector Lahisker', avatar: '/avatars/cats/cat-detective.jpg', color: '#F9E79F' },
  { name: 'J-Roc Meowski', avatar: '/avatars/cats/cat-dj.jpg', color: '#BB8FCE' },
  { name: 'Cyrus Rootpaw', avatar: '/avatars/cats/cat-hacker.jpg', color: '#4ECDC4' },
  { name: 'Conky Catsworth', avatar: '/avatars/cats/cat-music.jpg', color: '#F1948A' },
  { name: 'Shadow Rickitty', avatar: '/avatars/cats/cat-ninja.jpg', color: '#45B7D1' },
  { name: 'Captain Shittpaw', avatar: '/avatars/cats/cat-pirate.jpg', color: '#FF6B6B' },
  { name: 'Ricky Lawnchpurr', avatar: '/avatars/cats/cat-rocket.jpg', color: '#FFA07A' },
  { name: 'Randy Catnaps', avatar: '/avatars/cats/cat-sleeping.jpg', color: '#D7BDE2' },
  { name: 'The Liquor Whiskers', avatar: '/avatars/cats/cat-wizard.jpg', color: '#98D8C8' },
  { name: 'Ray Pawston', avatar: '/avatars/cats/cat-zen.jpg', color: '#ABEBC6' },
]

const defaultCharacter: CharacterInfo = {
  name: 'Shadow Rickitty',
  avatar: '/avatars/cats/cat-ninja.jpg',
  color: '#45B7D1'
}

/**
 * Improved hash function using FNV-1a algorithm for better distribution
 */
function improvedHash(str: string): number {
  const FNV_OFFSET_BASIS = 2166136261
  const FNV_PRIME = 16777619

  let hash = FNV_OFFSET_BASIS

  for (let i = 0; i < str.length; i++) {
    hash ^= str.charCodeAt(i)
    hash = Math.imul(hash, FNV_PRIME)
  }

  return Math.abs(hash >>> 0)
}

/**
 * Get character avatar information for a session name or ID
 * Uses a hash to deterministically map session IDs to cat avatars
 */
export function useCharacterAvatar(sessionName?: string | null): CharacterInfo {
  if (!sessionName) {
    return defaultCharacter
  }

  const hash = improvedHash(sessionName)
  const index = hash % fallbackAvatars.length

  return fallbackAvatars[index]
}

/**
 * Get all available characters
 */
export function getAllCharacters(): CharacterInfo[] {
  return [...fallbackAvatars]
}

/**
 * Check if a session name has a mapped character
 */
export function hasCharacter(sessionName?: string | null): boolean {
  return !!sessionName
}
