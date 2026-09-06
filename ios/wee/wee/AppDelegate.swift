//
//  AppDelegate.swift
//  wee
//
//  Application delegate for configuration
//

import UIKit

class AppDelegate: NSObject, UIApplicationDelegate {
    func application(
        _ application: UIApplication,
        didFinishLaunchingWithOptions launchOptions: [UIApplication.LaunchOptionsKey: Any]? = nil
    ) -> Bool {
        // Configure URL session for self-signed certificates
        // This allows the app to connect to localhost:3333 with self-signed certs

        // Configure any other app-level settings here

        return true
    }

    func application(
        _ application: UIApplication,
        didDiscardSceneSessions sceneSessions: Set<UISceneSession>
    ) {
        // Called when the user discards a scene session
        // This is called on iOS 13.5 and later
    }
}
