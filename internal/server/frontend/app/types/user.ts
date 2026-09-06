/**
 * User information types for frontend
 * Used to resolve user_id from messages to display data
 */

export interface UserInfo {
  /** Username for display */
  username: string
  /** UUID identifier (if available) */
  id?: string
  /** Email address (if available) */
  email?: string
  /** Avatar ID for rendering user avatars */
  avatar_id?: number
  /** Avatar image URL/data */
  avatar_image?: string
  /** Avatar display name */
  avatar_name?: string
  /** Avatar color (hex or named color) */
  avatar_color?: string
}

export interface UserInfoResponse {
  username: string
  id?: string
  email?: string
  avatar_id?: number
  avatar_image?: string
  avatar_name?: string
  avatar_color?: string
}
