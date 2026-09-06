use serde::{Deserialize, Serialize};
use tauri::{AppHandle, State};
use tauri_plugin_notification::NotificationExt;

use crate::AppState;

/// Hook execution event received from the Go sidecar
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct HookEvent {
    pub event_name: String,
    pub session_id: Option<String>,
    pub matcher: Option<String>,
    pub hook_type: String,
    pub command: Option<String>,
    pub exit_code: Option<i32>,
    pub blocked: bool,
    pub duration_ms: Option<i64>,
}

/// Hook status summary for tray icon
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct HookStatus {
    pub total_executions: i64,
    pub blocked_count: i64,
    pub failed_count: i64,
    pub status: String, // "green", "yellow", "red"
}

/// Send a native OS notification for a hook event
#[tauri::command]
pub async fn send_hook_notification(
    app: AppHandle,
    title: String,
    body: String,
    severity: Option<String>,
) -> Result<(), String> {
    let _ = severity; // Reserved for future icon customization

    app.notification()
        .builder()
        .title(&title)
        .body(&body)
        .show()
        .map_err(|e| format!("Failed to send notification: {}", e))?;

    Ok(())
}

/// Fetch hook execution stats from the Go sidecar API
#[tauri::command]
pub async fn get_hook_status(state: State<'_, AppState>) -> Result<HookStatus, String> {
    let port = {
        let p = state.wee_port.lock().map_err(|e| e.to_string())?;
        *p
    };

    let running = {
        let r = state.wee_running.lock().map_err(|e| e.to_string())?;
        *r
    };

    if !running {
        return Ok(HookStatus {
            total_executions: 0,
            blocked_count: 0,
            failed_count: 0,
            status: "green".to_string(),
        });
    }

    // Fetch stats from the Go sidecar's hook executions API
    let url = format!("https://localhost:{}/api/hooks/executions/stats", port);

    // Use reqwest with TLS verification disabled (self-signed certs)
    let client = reqwest::Client::builder()
        .danger_accept_invalid_certs(true)
        .build()
        .map_err(|e| format!("Failed to create HTTP client: {}", e))?;

    let response = client
        .get(&url)
        .send()
        .await
        .map_err(|e| format!("Failed to fetch hook stats: {}", e))?;

    if !response.status().is_success() {
        return Ok(HookStatus {
            total_executions: 0,
            blocked_count: 0,
            failed_count: 0,
            status: "green".to_string(),
        });
    }

    let stats: serde_json::Value = response
        .json()
        .await
        .map_err(|e| format!("Failed to parse hook stats: {}", e))?;

    let total = stats["total_executions"].as_i64().unwrap_or(0);
    let blocked = stats["blocked_count"].as_i64().unwrap_or(0);
    let failed = stats["failed_count"].as_i64().unwrap_or(0);

    let status = if blocked > 0 {
        "red".to_string()
    } else if failed > 0 {
        "yellow".to_string()
    } else {
        "green".to_string()
    };

    Ok(HookStatus {
        total_executions: total,
        blocked_count: blocked,
        failed_count: failed,
        status,
    })
}

/// Fetch recent hook executions from the Go sidecar API
#[tauri::command]
pub async fn get_recent_hook_executions(
    state: State<'_, AppState>,
    limit: Option<i32>,
) -> Result<Vec<HookEvent>, String> {
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

    let limit = limit.unwrap_or(10);
    let url = format!(
        "https://localhost:{}/api/hooks/executions?limit={}",
        port, limit
    );

    let client = reqwest::Client::builder()
        .danger_accept_invalid_certs(true)
        .build()
        .map_err(|e| format!("Failed to create HTTP client: {}", e))?;

    let response = client
        .get(&url)
        .send()
        .await
        .map_err(|e| format!("Failed to fetch hook executions: {}", e))?;

    if !response.status().is_success() {
        return Ok(vec![]);
    }

    let data: serde_json::Value = response
        .json()
        .await
        .map_err(|e| format!("Failed to parse hook executions: {}", e))?;

    let executions = data["executions"]
        .as_array()
        .map(|arr| {
            arr.iter()
                .filter_map(|v| {
                    Some(HookEvent {
                        event_name: v["event_name"].as_str()?.to_string(),
                        session_id: v["session_id"].as_str().map(|s| s.to_string()),
                        matcher: v["matcher"].as_str().map(|s| s.to_string()),
                        hook_type: v["hook_type"].as_str()?.to_string(),
                        command: v["command"].as_str().map(|s| s.to_string()),
                        exit_code: v["exit_code"].as_i64().map(|i| i as i32),
                        blocked: v["blocked"].as_bool().unwrap_or(false),
                        duration_ms: v["duration_ms"].as_i64(),
                    })
                })
                .collect()
        })
        .unwrap_or_default();

    Ok(executions)
}
