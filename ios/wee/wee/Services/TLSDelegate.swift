//
//  TLSDelegate.swift
//  wee
//
//  URLSessionDelegate for handling self-signed certificates
//

import Foundation

/// URLSessionDelegate that handles self-signed HTTPS certificates for localhost
class TLSDelegate: NSObject, URLSessionDelegate {
    /// Whether to allow self-signed certificates (default: true for localhost)
    var allowSelfSignedCertificates = true

    /// List of allowed hosts for self-signed certificates (default: localhost)
    /// NOTE: Remote URLs like ngrok will use standard certificate validation automatically
    var allowedHosts = Set(["localhost", "127.0.0.1", "::1"])

    func urlSession(
        _ session: URLSession,
        didReceive challenge: URLAuthenticationChallenge,
        completionHandler: @escaping (URLSession.AuthChallengeDisposition, URLCredential?) -> Void
    ) {
        // Handle SSL certificate validation
        guard challenge.protectionSpace.authenticationMethod == NSURLAuthenticationMethodServerTrust else {
            completionHandler(.performDefaultHandling, nil)
            return
        }

        // Check if this is a host we allow self-signed certificates for
        let host = challenge.protectionSpace.host
        let isAllowedHost = allowedHosts.contains(host)

        // If we allow self-signed certs and this is an allowed host, accept the certificate
        if allowSelfSignedCertificates && isAllowedHost {
            if let serverTrust = challenge.protectionSpace.serverTrust {
                let credential = URLCredential(trust: serverTrust)
                completionHandler(.useCredential, credential)
                return
            }
        }

        // Otherwise, perform default handling (standard cert validation)
        completionHandler(.performDefaultHandling, nil)
    }
}

/// Creates a URLSession configured for Wee with self-signed certificate support
func createWeeURLSession() -> URLSession {
    let delegate = TLSDelegate()

    let config = URLSessionConfiguration.default
    config.timeoutIntervalForRequest = 30
    config.timeoutIntervalForResource = 300
    config.waitsForConnectivity = true

    // Disable certificate validation for localhost (self-signed certs)
    config.tlsMinimumSupportedProtocolVersion = .TLSv12

    return URLSession(configuration: config, delegate: delegate, delegateQueue: nil)
}
