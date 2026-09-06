use serde::Serialize;
use std::sync::Mutex;
use tauri::{AppHandle, Manager, State, Url};
use tauri_plugin_dialog::DialogExt;

mod hooks;
mod hotkeys;
mod sidecar;
mod tray;

const WEE_PORT: u16 = 3334;

#[derive(Default)]
pub struct AppState {
    pub wee_port: Mutex<u16>,
    pub wee_running: Mutex<bool>,
}

#[derive(Clone, Serialize)]
struct StartupProgress {
    step: String,
    message: String,
    done: bool,
}

#[tauri::command]
async fn get_dashboard_url(state: State<'_, AppState>) -> Result<String, String> {
    let port = state.wee_port.lock().map_err(|e| e.to_string())?;
    Ok(format!("https://localhost:{}", *port))
}

#[tauri::command]
async fn get_status(state: State<'_, AppState>) -> Result<bool, String> {
    let running = state.wee_running.lock().map_err(|e| e.to_string())?;
    Ok(*running)
}

#[tauri::command]
async fn select_folder(app: AppHandle) -> Result<Option<String>, String> {
    let (tx, rx) = tokio::sync::oneshot::channel();

    app.dialog()
        .file()
        .set_title("Select Project Folder")
        .pick_folder(move |folder| {
            let _ = tx.send(folder);
        });

    rx.await
        .map_err(|e| e.to_string())
        .map(|f| f.map(|p| p.to_string()))
}

async fn start_wee_server(app: AppHandle) -> Result<(), String> {
    let state = app.state::<AppState>();
    let port = WEE_PORT;

    {
        let mut p = state.wee_port.lock().map_err(|e| e.to_string())?;
        *p = port;
    }

    // Start the wee sidecar
    sidecar::start_wee(&app, port).await?;

    {
        let mut running = state.wee_running.lock().map_err(|e| e.to_string())?;
        *running = true;
    }

    // Navigate the main webview to the wee server
    let url = format!("https://localhost:{}", port);
    eprintln!("Navigating to: {}", url);

    if let Some(main_window) = app.get_webview_window("main") {
        // Let the splash animation play for 1.5 seconds
        tokio::time::sleep(std::time::Duration::from_millis(1500)).await;

        // Smooth fade-out of the splash screen
        let _ = main_window.eval(r#"
            (function() {
                // Update status to "READY"
                var s = document.getElementById('status-text');
                if (s) s.textContent = 'SYSTEM READY';
                // Complete the loading bar
                var bar = document.querySelector('.loader-bar');
                if (bar) { bar.style.transition = 'width 0.3s ease-out'; bar.style.width = '100%'; }
                // Fade out after a beat
                setTimeout(function() {
                    document.body.style.transition = 'opacity 0.4s ease-in-out';
                    document.body.style.opacity = '0';
                }, 300);
            })();
        "#);

        // Wait for fade-out to complete (300ms delay + 400ms transition)
        tokio::time::sleep(std::time::Duration::from_millis(700)).await;

        // Navigate to the app
        let parsed_url = Url::parse(&url).map_err(|e| e.to_string())?;
        main_window
            .navigate(parsed_url)
            .map_err(|e| format!("Failed to navigate: {}", e))?;

        // Wait for the page to load
        tokio::time::sleep(std::time::Duration::from_millis(1100)).await;

        // Inject CSS to add padding for macOS overlay title bar
        let _ = main_window.eval(r#"
            (function() {
                var style = document.createElement('style');
                style.textContent = `
                    /* Offset for macOS overlay title bar traffic lights */
                    .navbar, nav.navbar {
                        padding-top: 38px !important;
                        -webkit-app-region: drag;
                    }
                    .navbar button, .navbar a, .navbar select, .navbar input,
                    .nav-right, .nav-right *, .sidebar-toggle, .user-menu, .user-menu *,
                    .nav-logo-link, .shortcuts-button, .project-selector, .project-selector * {
                        -webkit-app-region: no-drag;
                    }
                `;
                document.head.appendChild(style);
                // Re-apply on navigation
                var observer = new MutationObserver(function() {
                    if (!document.querySelector('#tauri-titlebar-fix')) {
                        var s = document.createElement('style');
                        s.id = 'tauri-titlebar-fix';
                        s.textContent = style.textContent;
                        document.head.appendChild(s);
                    }
                });
                observer.observe(document.documentElement, { childList: true, subtree: true });
            })();
        "#);
    }

    Ok(())
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_notification::init())
        .plugin(tauri_plugin_single_instance::init(|app, _args, _cwd| {
            if let Some(window) = app.get_webview_window("main") {
                let _ = window.set_focus();
            }
        }))
        .plugin(tauri_plugin_store::Builder::default().build())
        .plugin(tauri_plugin_dialog::init())
        .plugin(tauri_plugin_window_state::Builder::default().build())
        .plugin(tauri_plugin_deep_link::init())
        .manage(AppState::default())
        .invoke_handler(tauri::generate_handler![
            get_dashboard_url,
            get_status,
            select_folder,
            hooks::send_hook_notification,
            hooks::get_hook_status,
            hooks::get_recent_hook_executions,
            hotkeys::get_hotkey_bindings,
            hotkeys::save_hotkey_bindings,
            hotkeys::invoke_skill,
        ])
        .setup(|app| {
            // Set up tray
            tray::setup_tray(app)?;

            // Start the wee server in the background
            let handle = app.handle().clone();
            tauri::async_runtime::spawn(async move {
                if let Err(e) = start_wee_server(handle.clone()).await {
                    eprintln!("Failed to start Wee server: {}", e);
                    if let Some(main_window) = handle.get_webview_window("main") {
                        let _ = main_window.show();
                    }
                }
            });

            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
