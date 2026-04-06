#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .setup(|app| {
            if cfg!(debug_assertions) {
                app.handle().plugin(
                    tauri_plugin_log::Builder::default()
                        .level(log::LevelFilter::Info)
                        .build(),
                )?;
            }

            // macOS: add a standard menu so Cmd+Q works and the app behaves natively
            #[cfg(target_os = "macos")]
            {
                use tauri::menu::{MenuBuilder, SubmenuBuilder};
                let menu = MenuBuilder::new(app)
                    .item(
                        &SubmenuBuilder::new(app, "Alexandria")
                            .about(None)
                            .separator()
                            .quit()
                            .build()?,
                    )
                    .item(
                        &SubmenuBuilder::new(app, "Edit")
                            .undo()
                            .redo()
                            .separator()
                            .cut()
                            .copy()
                            .paste()
                            .select_all()
                            .build()?,
                    )
                    .build()?;
                app.set_menu(menu)?;
            }

            Ok(())
        })
        .plugin(tauri_plugin_store::Builder::default().build())
        .plugin(tauri_plugin_http::init())
        .plugin(tauri_plugin_shell::init())
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
