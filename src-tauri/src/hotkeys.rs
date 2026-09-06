use serde::{Deserialize, Serialize};
use tauri::State;

use crate::AppState;

/// A hotkey binding that maps a keyboard shortcut to a skill name
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct HotkeyBinding {
    pub shortcut: String,  // e.g., "CmdOrCtrl+Shift+D"
    pub skill_name: String, // e.g., "deploy"
    pub description: Option<String>,
}

/// Get the current hotkey bindings from the sidecar settings
#[tauri::command]
pub async fn get_hotkey_bindings(state: State<'_, AppState>) -> Result<Vec<HotkeyBinding>, String> {
    let port = {
        let p = state.wee_port.lock().map_err(|e| e.to_string())?;
        *p
    };

    let running = {
        let r = state.wee_running.lock().map_err(|e| e.to_string())?;
        *r
    };

    if !running {
        return Ok(vec![]);
    }

    // Fetch hotkey settings from the sidecar settings API
    let url = format!("https://localhost:{}/api/settings/hotkeys", port);

    let client = reqwest::Client::builder()
        .danger_accept_invalid_certs(true)
        .build()
        .map_err(|e| format!("Failed to create HTTP client: {}", e))?;

    let response = client
        .get(&url)
        .send()
        .await;

    match response {
        Ok(resp) if resp.status().is_success() => {
            let bindings: Vec<HotkeyBinding> = resp
                .json()
                .await
                .unwrap_or_default();
            Ok(bindings)
        }
        _ => {
            // No hotkeys configured or endpoint not available yet
            Ok(vec![])
        }
    }
}

/// Save hotkey bindings to the sidecar settings
#[tauri::command]
pub async fn save_hotkey_bindings(
    state: State<'_, AppState>,
    bindings: Vec<HotkeyBinding>,
) -> Result<(), String> {
    let port = {
        let p = state.wee_port.lock().map_err(|e| e.to_string())?;
        *p
    };

    let running = {
        let r = state.wee_running.lock().map_err(|e| e.to_string())?;
        *r
    };

    if !running {
        return Err("Wee server is not running".to_string());
    }

    let url = format!("https://localhost:{}/api/settings/hotkeys", port);

    let client = reqwest::Client::builder()
        .danger_accept_invalid_certs(true)
        .build()
        .map_err(|e| format!("Failed to create HTTP client: {}", e))?;

    client
        .put(&url)
        .json(&bindings)
        .send()
        .await
        .map_err(|e| format!("Failed to save hotkey bindings: {}", e))?;

    Ok(())
}

/// Invoke a skill by name via the sidecar API
#[tauri::command]
pub async fn invoke_skill(
    state: State<'_, AppState>,
    skill_name: String,
    session_id: Option<String>,
) -> Result<String, String> {
    let port = {
        let p = state.wee_port.lock().map_err(|e| e.to_string())?;
        *p
    };

    let running = {
        let r = state.wee_running.lock().map_err(|e| e.to_string())?;
        *r
    };

    if !running {
        return Err("Wee server is not running".to_string());
    }

    // Get skill details from sidecar
    let url = format!("https://localhost:{}/api/skills/{}", port, skill_name);

    let client = reqwest::Client::builder()
        .danger_accept_invalid_certs(true)
        .build()
        .map_err(|e| format!("Failed to create HTTP client: {}", e))?;

    let response = client
        .get(&url)
        .send()
        .await
        .map_err(|e| format!("Failed to fetch skill: {}", e))?;

    if !response.status().is_success() {
        return Err(format!("Skill '{}' not found", skill_name));
    }

    let skill: serde_json::Value = response
        .json()
        .await
        .map_err(|e| format!("Failed to parse skill: {}", e))?;

    let _ = session_id; // Reserved for future: auto-invoke in active session

    Ok(serde_json::to_string(&skill).unwrap_or_default())
}
