use tauri::{
    image::Image,
    menu::{Menu, MenuItem, PredefinedMenuItem},
    tray::TrayIconBuilder,
    App, Manager,
};

pub fn setup_tray(app: &App) -> Result<(), Box<dyn std::error::Error>> {
    let show = MenuItem::with_id(app, "show", "Show Wee", true, None::<&str>)?;
    let separator = PredefinedMenuItem::separator(app)?;
    let hooks_status = MenuItem::with_id(app, "hooks_status", "Hooks: No activity", false, None::<&str>)?;
    let skills_menu = MenuItem::with_id(app, "skills", "Skills Manager", true, None::<&str>)?;
    let separator2 = PredefinedMenuItem::separator(app)?;
    let quit = MenuItem::with_id(app, "quit", "Quit", true, None::<&str>)?;

    let menu = Menu::with_items(
        app,
        &[&show, &separator, &hooks_status, &skills_menu, &separator2, &quit],
    )?;

    let icon = Image::from_path("icons/icon.png").unwrap_or_else(|_| {
        Image::from_bytes(include_bytes!("../icons/icon.png"))
            .expect("Failed to load tray icon")
    });

    TrayIconBuilder::new()
        .icon(icon)
        .menu(&menu)
        .tooltip("Wee — AI Agent Control Center")
        .on_menu_event(|app, event| match event.id.as_ref() {
            "show" => {
                if let Some(window) = app.get_webview_window("main") {
                    let _ = window.show();
                    let _ = window.set_focus();
                }
            }
            "skills" => {
                // Navigate to skills page
                if let Some(window) = app.get_webview_window("main") {
                    let _ = window.show();
                    let _ = window.set_focus();
                    let _ = window.eval("window.location.hash = ''; window.location.pathname = '/skills';");
                }
            }
            "quit" => {
                app.exit(0);
            }
            _ => {}
        })
        .build(app)?;

    Ok(())
}
