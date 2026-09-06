use std::collections::HashMap;
use std::net::TcpStream;
use std::process::Command;
use std::time::Duration;

/// Load environment variables from the user's login shell.
/// macOS GUI apps don't inherit shell env vars, so we source them.
fn load_shell_env() -> HashMap<String, String> {
    let mut env = HashMap::new();

    // Try to get the user's full shell environment via login shell
    let shell = std::env::var("SHELL").unwrap_or_else(|_| "/bin/zsh".to_string());
    if let Ok(output) = Command::new(&shell)
        .args(["-l", "-i", "-c", "env"])
        .output()
    {
        let stdout = String::from_utf8_lossy(&output.stdout);
        for line in stdout.lines() {
            if let Some((key, value)) = line.split_once('=') {
                env.insert(key.to_string(), value.to_string());
            }
        }
    }

    env
}

/// Start the wee Go binary as a sidecar process
pub async fn start_wee(_app: &tauri::AppHandle, port: u16) -> Result<(), String> {
    // Find the wee-server binary next to our own executable
    let current_exe = std::env::current_exe().map_err(|e| format!("Can't find exe: {}", e))?;
    let exe_parent = current_exe.parent().ok_or("No parent dir")?;

    let binary_path = exe_parent.join("wee-server");

    if !binary_path.exists() {
        return Err(format!(
            "wee-server binary not found at: {}. Dir contents: {:?}",
            binary_path.display(),
            std::fs::read_dir(exe_parent)
                .map(|rd| rd
                    .filter_map(|e| e.ok())
                    .map(|e| e.file_name().to_string_lossy().to_string())
                    .collect::<Vec<_>>())
                .unwrap_or_default()
        ));
    }

    // Load the user's shell environment (for API keys, PATH, etc.)
    let shell_env = load_shell_env();

    eprintln!(
        "Starting wee-server from: {} (shell env has {} vars, ANTHROPIC_API_KEY={})",
        binary_path.display(),
        shell_env.len(),
        if shell_env.contains_key("ANTHROPIC_API_KEY") { "set" } else { "not set" }
    );

    // Start with a clean env from the user's shell, then override specific vars
    let mut cmd = Command::new(&binary_path);

    // Forward ALL env vars from the user's shell
    cmd.env_clear();
    for (key, value) in &shell_env {
        cmd.env(key, value);
    }

    // Override/add Tauri-specific vars
    cmd.env("WEE_PORT", port.to_string());
    cmd.env("WEE_NO_BROWSER", "1"); // Don't auto-open browser, Tauri handles the UI

    cmd.spawn()
        .map_err(|e| format!("Failed to spawn wee-server: {}", e))?;

    // Wait for the server to be ready by polling TCP port
    wait_for_ready(port).await?;

    Ok(())
}

/// Poll until the wee server accepts TCP connections on the given port
async fn wait_for_ready(port: u16) -> Result<(), String> {
    let max_attempts = 30;
    let addr = format!("127.0.0.1:{}", port);

    for attempt in 1..=max_attempts {
        if TcpStream::connect_timeout(
            &addr.parse().unwrap(),
            Duration::from_millis(500),
        )
        .is_ok()
        {
            return Ok(());
        }

        if attempt == max_attempts {
            return Err(format!(
                "Wee server did not start after {} attempts",
                max_attempts
            ));
        }
        tokio::time::sleep(Duration::from_secs(1)).await;
    }

    Ok(())
}
