//
//  AudioRecorder.swift
//  wee
//
//  Service for recording audio and transcribing with Apple's Speech.framework
//

import Foundation
import Combine
import AVFoundation
import Speech

@MainActor
class AudioRecorder: NSObject, ObservableObject, AVAudioRecorderDelegate {
    // MARK: - Published Properties

    @Published var isRecording = false
    @Published var recordingTime: TimeInterval = 0
    @Published var errorMessage: String?
    @Published var isTranscribing = false
    @Published var transcriptionResult: String?

    // MARK: - Properties

    private var audioRecorder: AVAudioRecorder?
    private var audioPlayer: AVAudioPlayer?
    private var timer: Timer?
    private let recordingURL = FileManager.default.temporaryDirectory.appendingPathComponent("voice_memo.m4a")
    private let speechRecognizer = SFSpeechRecognizer(locale: Locale(identifier: "en-US"))

    // Callback for when transcription completes
    var onTranscriptionComplete: ((String) -> Void)?

    // MARK: - Initialization

    override init() {
        super.init()
        setupAudioSession()
    }

    deinit {
        timer?.invalidate()
        _ = try? audioRecorder?.stop()
    }

    // MARK: - Setup

    private func setupAudioSession() {
        let audioSession = AVAudioSession.sharedInstance()
        do {
            try audioSession.setCategory(.record, mode: .measurement, options: [])
            try audioSession.setActive(true, options: .notifyOthersOnDeactivation)
        } catch {
            errorMessage = "Failed to set up audio session: \(error.localizedDescription)"
        }
    }

    // MARK: - Recording Controls

    func startRecording() async {
        // Request microphone permission if needed
        let granted = await requestMicrophonePermission()
        guard granted else {
            errorMessage = "Microphone permission denied"
            return
        }

        errorMessage = nil
        recordingTime = 0

        // Clean up existing recording
        _ = try? FileManager.default.removeItem(at: recordingURL)

        let settings: [String: Any] = [
            AVFormatIDKey: Int(kAudioFormatMPEG4AAC),
            AVSampleRateKey: 16000,
            AVNumberOfChannelsKey: 1,
            AVEncoderAudioQualityKey: AVAudioQuality.high.rawValue
        ]

        do {
            audioRecorder = try AVAudioRecorder(url: recordingURL, settings: settings)
            audioRecorder?.delegate = self

            guard audioRecorder?.record() == true else {
                errorMessage = "Failed to start recording"
                return
            }

            isRecording = true
            startTimer()
        } catch {
            errorMessage = "Failed to create recorder: \(error.localizedDescription)"
        }
    }

    func stopRecording() async {
        guard isRecording else { return }

        audioRecorder?.stop()
        isRecording = false
        stopTimer()


        // Transcribe the recording
        await transcribeRecording()
    }

    // MARK: - Transcription

    private func transcribeRecording() async {
        guard FileManager.default.fileExists(atPath: recordingURL.path) else {
            errorMessage = "Recording file not found"
            return
        }

        let fileSize = try? FileManager.default.attributesOfItem(atPath: recordingURL.path)[.size] as? Int

        isTranscribing = true
        defer { isTranscribing = false }

        do {
            // Check if speech recognition is available
            guard let speechRecognizer = speechRecognizer, speechRecognizer.isAvailable else {
                errorMessage = "Speech recognition not available on this device"
                return
            }

            // Create URL-based recognition request (cleaner for file-based audio)
            let request = SFSpeechURLRecognitionRequest(url: recordingURL)
            request.shouldReportPartialResults = false
            request.requiresOnDeviceRecognition = false  // Allow online if offline fails

            // Perform speech recognition
            let transcription = await withCheckedContinuation { continuation in
                var recognitionTask: SFSpeechRecognitionTask?

                recognitionTask = speechRecognizer.recognitionTask(with: request) { result, error in
                    switch (result, error) {
                    case (let result?, nil):
                        if result.isFinal {
                            let text = result.bestTranscription.formattedString
                            continuation.resume(returning: text)
                            recognitionTask?.cancel()
                        }

                    case (_, let error?):
                        let errorMsg = error.localizedDescription
                        continuation.resume(returning: "")
                        recognitionTask?.cancel()

                    default:
                        break
                    }
                }
            }

            guard !transcription.isEmpty else {
                errorMessage = "Could not understand audio - try speaking more clearly"
                return
            }


            // Update published property and call callback
            self.transcriptionResult = transcription
            self.onTranscriptionComplete?(transcription)

            // Clear the recording file after successful transcription
            _ = try? FileManager.default.removeItem(at: recordingURL)
        } catch {
            errorMessage = "Transcription failed: \(error.localizedDescription)"
        }
    }

    // MARK: - Permissions

    private func requestMicrophonePermission() async -> Bool {
        // Check current permission status
        let status = AVAudioSession.sharedInstance().recordPermission

        switch status {
        case .granted:
            return true
        case .undetermined:
            // Request permission
            return await withCheckedContinuation { continuation in
                // Suppress deprecation warning - we support iOS 15+
                nonisolated(unsafe) let audioSession = AVAudioSession.sharedInstance()
                audioSession.requestRecordPermission { granted in
                    continuation.resume(returning: granted)
                }
            }
        case .denied:
            return false
        @unknown default:
            return false
        }
    }

    // MARK: - Timer

    private func startTimer() {
        stopTimer()
        timer = Timer.scheduledTimer(withTimeInterval: 0.1, repeats: true) { [weak self] _ in
            Task { @MainActor in
                self?.recordingTime += 0.1
            }
        }
    }

    private func stopTimer() {
        timer?.invalidate()
        timer = nil
    }

    // MARK: - Delegates

    nonisolated func audioRecorderDidFinishRecording(_ recorder: AVAudioRecorder, successfully flag: Bool) {
        Task { @MainActor in
            if !flag {
                self.errorMessage = "Recording failed"
            }
        }
    }

    nonisolated func audioRecorderEncodeErrorDidOccur(_ recorder: AVAudioRecorder, error: Error?) {
        Task { @MainActor in
            self.errorMessage = "Recording error: \(error?.localizedDescription ?? "Unknown")"
        }
    }
}

// MARK: - Recording Format Helper

extension TimeInterval {
    var formattedTime: String {
        let minutes = Int(self) / 60
        let seconds = Int(self) % 60
        let centiseconds = Int((self.truncatingRemainder(dividingBy: 1)) * 100)
        return String(format: "%02d:%02d.%02d", minutes, seconds, centiseconds)
    }
}
